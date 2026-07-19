package core

import (
	"errors"
	"strings"
	"time"
)

// TaskID uniquely identifies a Task.
type TaskID string

// Task is a unit of work, optionally blocked by other tasks.
type Task struct {
	ID        TaskID
	Title     string
	Notes     string
	Done      bool
	CreatedAt time.Time
	DueAt     *time.Time
	BlockedBy []TaskID
}

var ErrEmptyTitle = errors.New("core: title must not be empty")

// NewTask constructs a Task with a fresh ID and CreatedAt set to now.
func NewTask(id TaskID, title string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}
	return &Task{
		ID:        id,
		Title:     title,
		CreatedAt: time.Now(),
	}, nil
}

// MarkDone marks the task as complete.
func (t *Task) MarkDone() {
	t.Done = true
}

// MarkUndone reopens a completed task.
func (t *Task) MarkUndone() {
	t.Done = false
}

// IsOverdue reports whether the task has a due date in the past and is not done.
func (t *Task) IsOverdue(asOf time.Time) bool {
	if t.Done || t.DueAt == nil {
		return false
	}
	return t.DueAt.Before(asOf)
}
