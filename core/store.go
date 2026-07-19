package core

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DefaultDBPath returns the default SQLite database location:
// ~/.local/share/tasktracker/tasks.db, falling back to a relative path if
// the home directory can't be determined.
func DefaultDBPath() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "tasktracker.db"
	}
	return filepath.Join(dir, ".local", "share", "tasktracker", "tasks.db")
}

const schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id         TEXT PRIMARY KEY,
	title      TEXT NOT NULL,
	notes      TEXT NOT NULL DEFAULT '',
	done       INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	due_at     TEXT
);

CREATE TABLE IF NOT EXISTS task_dependencies (
	task_id    TEXT NOT NULL,
	blocker_id TEXT NOT NULL,
	PRIMARY KEY (task_id, blocker_id)
);

CREATE TABLE IF NOT EXISTS habits (
	id            TEXT PRIMARY KEY,
	title         TEXT NOT NULL,
	frequency     TEXT NOT NULL,
	interval_days INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS habit_completions (
	habit_id     TEXT NOT NULL,
	completed_at TEXT NOT NULL,
	PRIMARY KEY (habit_id, completed_at)
);
`

// Store is a SQLite-backed repository for Tasks and Habits.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) a SQLite database at path and applies
// the schema. Use ":memory:" for an ephemeral in-memory database.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("core: open store: %w", err)
	}
	// modernc.org/sqlite is a single-connection-safe pure-Go driver;
	// SQLite itself does not handle concurrent writers well, so keep this
	// process to one connection to avoid "database is locked" errors.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("core: apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveTask inserts or updates a task's own fields. It does not touch
// dependency edges; use AddDependency for those.
func (s *Store) SaveTask(ctx context.Context, t *Task) error {
	var dueAt any
	if t.DueAt != nil {
		dueAt = t.DueAt.Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, title, notes, done, created_at, due_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			notes = excluded.notes,
			done = excluded.done,
			due_at = excluded.due_at
	`, string(t.ID), t.Title, t.Notes, boolToInt(t.Done), t.CreatedAt.Format(time.RFC3339), dueAt)
	if err != nil {
		return fmt.Errorf("core: save task %q: %w", t.ID, err)
	}
	return nil
}

// DeleteTask removes a task and any dependency edges that reference it.
func (s *Store) DeleteTask(ctx context.Context, id TaskID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, string(id)); err != nil {
		return fmt.Errorf("core: delete task %q: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_dependencies WHERE task_id = ? OR blocker_id = ?`, string(id), string(id)); err != nil {
		return fmt.Errorf("core: delete task %q dependencies: %w", id, err)
	}
	return tx.Commit()
}

// GetTask loads a single task, including its BlockedBy edges.
func (s *Store) GetTask(ctx context.Context, id TaskID) (*Task, error) {
	tasks, err := s.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, fmt.Errorf("core: task %q not found", id)
}

// ListTasks loads every task with its BlockedBy edges populated.
func (s *Store) ListTasks(ctx context.Context) ([]*Task, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, notes, done, created_at, due_at FROM tasks`)
	if err != nil {
		return nil, fmt.Errorf("core: list tasks: %w", err)
	}
	defer rows.Close()

	byID := make(map[TaskID]*Task)
	var order []TaskID
	for rows.Next() {
		var (
			id, title, notes, createdAt string
			done                        int
			dueAt                       sql.NullString
		)
		if err := rows.Scan(&id, &title, &notes, &done, &createdAt, &dueAt); err != nil {
			return nil, fmt.Errorf("core: scan task: %w", err)
		}
		t := &Task{
			ID:    TaskID(id),
			Title: title,
			Notes: notes,
			Done:  done != 0,
		}
		if t.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("core: parse created_at for %q: %w", id, err)
		}
		if dueAt.Valid {
			parsed, err := time.Parse(time.RFC3339, dueAt.String)
			if err != nil {
				return nil, fmt.Errorf("core: parse due_at for %q: %w", id, err)
			}
			t.DueAt = &parsed
		}
		byID[t.ID] = t
		order = append(order, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	edgeRows, err := s.db.QueryContext(ctx, `SELECT task_id, blocker_id FROM task_dependencies`)
	if err != nil {
		return nil, fmt.Errorf("core: list dependencies: %w", err)
	}
	defer edgeRows.Close()
	for edgeRows.Next() {
		var taskID, blockerID string
		if err := edgeRows.Scan(&taskID, &blockerID); err != nil {
			return nil, fmt.Errorf("core: scan dependency: %w", err)
		}
		if t, ok := byID[TaskID(taskID)]; ok {
			t.BlockedBy = append(t.BlockedBy, TaskID(blockerID))
		}
	}
	if err := edgeRows.Err(); err != nil {
		return nil, err
	}

	tasks := make([]*Task, 0, len(order))
	for _, id := range order {
		tasks = append(tasks, byID[id])
	}
	return tasks, nil
}

// AddDependency records that task is blocked by blocker, rejecting the
// change if it would introduce a cycle (checked against the full,
// currently-stored task set via Graph).
func (s *Store) AddDependency(ctx context.Context, task, blocker TaskID) error {
	tasks, err := s.ListTasks(ctx)
	if err != nil {
		return err
	}
	g := NewGraph(tasks)
	if err := g.AddDependency(task, blocker); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO task_dependencies (task_id, blocker_id) VALUES (?, ?)
		ON CONFLICT(task_id, blocker_id) DO NOTHING
	`, string(task), string(blocker))
	if err != nil {
		return fmt.Errorf("core: add dependency %q -> %q: %w", task, blocker, err)
	}
	return nil
}

// ReadyTasks returns the not-done tasks whose blockers are all done.
func (s *Store) ReadyTasks(ctx context.Context) ([]*Task, error) {
	tasks, err := s.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	return NewGraph(tasks).ReadyTasks(), nil
}

// SaveHabit inserts or updates a habit's own fields (not its completion log).
func (s *Store) SaveHabit(ctx context.Context, h *Habit) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO habits (id, title, frequency, interval_days)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			frequency = excluded.frequency,
			interval_days = excluded.interval_days
	`, string(h.ID), h.Title, string(h.Frequency), h.IntervalDays)
	if err != nil {
		return fmt.Errorf("core: save habit %q: %w", h.ID, err)
	}
	return nil
}

// CheckInHabit records a completion for the given day, deduplicating
// same-day check-ins at the database level.
func (s *Store) CheckInHabit(ctx context.Context, id HabitID, t time.Time) error {
	day := truncateToDay(t).Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO habit_completions (habit_id, completed_at) VALUES (?, ?)
		ON CONFLICT(habit_id, completed_at) DO NOTHING
	`, string(id), day)
	if err != nil {
		return fmt.Errorf("core: check in habit %q: %w", id, err)
	}
	return nil
}

// GetHabit loads a single habit, including its completion log.
func (s *Store) GetHabit(ctx context.Context, id HabitID) (*Habit, error) {
	habits, err := s.ListHabits(ctx)
	if err != nil {
		return nil, err
	}
	for _, h := range habits {
		if h.ID == id {
			return h, nil
		}
	}
	return nil, fmt.Errorf("core: habit %q not found", id)
}

// ListHabits loads every habit with its completion log populated.
func (s *Store) ListHabits(ctx context.Context) ([]*Habit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, frequency, interval_days FROM habits`)
	if err != nil {
		return nil, fmt.Errorf("core: list habits: %w", err)
	}
	defer rows.Close()

	byID := make(map[HabitID]*Habit)
	var order []HabitID
	for rows.Next() {
		var id, title, freq string
		var intervalDays int
		if err := rows.Scan(&id, &title, &freq, &intervalDays); err != nil {
			return nil, fmt.Errorf("core: scan habit: %w", err)
		}
		h := &Habit{
			ID:           HabitID(id),
			Title:        title,
			Frequency:    Frequency(freq),
			IntervalDays: intervalDays,
		}
		byID[h.ID] = h
		order = append(order, h.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	logRows, err := s.db.QueryContext(ctx, `SELECT habit_id, completed_at FROM habit_completions ORDER BY completed_at`)
	if err != nil {
		return nil, fmt.Errorf("core: list habit completions: %w", err)
	}
	defer logRows.Close()
	for logRows.Next() {
		var habitID, completedAt string
		if err := logRows.Scan(&habitID, &completedAt); err != nil {
			return nil, fmt.Errorf("core: scan habit completion: %w", err)
		}
		h, ok := byID[HabitID(habitID)]
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, completedAt)
		if err != nil {
			return nil, fmt.Errorf("core: parse completed_at for %q: %w", habitID, err)
		}
		h.CompletionLog = append(h.CompletionLog, t)
	}
	if err := logRows.Err(); err != nil {
		return nil, err
	}

	habits := make([]*Habit, 0, len(order))
	for _, id := range order {
		habits = append(habits, byID[id])
	}
	return habits, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
