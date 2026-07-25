package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"

	"github.com/sweetlife999/go-my-tracker/core"
)

func ctx() context.Context { return context.Background() }

// taskStatus is the Done > Ready > Blocked classification shown on a task
// row, mirroring mobile/status.go. It is always derived from
// core.Store.ReadyTasks rather than recomputed from BlockedBy edges, so a
// task whose blockers are all done reads as ready, not blocked.
type taskStatus int

const (
	statusBlocked taskStatus = iota
	statusReady
	statusDone
)

// snapshot is the derived view of the whole task set that row rendering
// needs: which tasks core considers ready, and how many of each task's
// blockers are still open.
type snapshot struct {
	ready        map[core.TaskID]bool
	openBlockers map[core.TaskID]int
}

// newSnapshot derives readiness and open-blocker counts for tasks. ready is
// the set of IDs core.Store.ReadyTasks returned.
func newSnapshot(tasks []*core.Task, ready []*core.Task) snapshot {
	s := snapshot{
		ready:        make(map[core.TaskID]bool, len(ready)),
		openBlockers: make(map[core.TaskID]int, len(tasks)),
	}
	for _, t := range ready {
		s.ready[t.ID] = true
	}

	done := make(map[core.TaskID]bool, len(tasks))
	for _, t := range tasks {
		done[t.ID] = t.Done
	}
	for _, t := range tasks {
		open := 0
		for _, id := range t.BlockedBy {
			if !done[id] {
				open++
			}
		}
		s.openBlockers[t.ID] = open
	}
	return s
}

func (s snapshot) statusOf(t *core.Task) taskStatus {
	switch {
	case t.Done:
		return statusDone
	case s.ready[t.ID]:
		return statusReady
	default:
		return statusBlocked
	}
}

// taskItem adapts *core.Task to bubbles/list.Item, carrying the derived
// status/open-blocker count alongside it so Description never has to guess.
type taskItem struct {
	t            *core.Task
	status       taskStatus
	openBlockers int
}

func (i taskItem) Title() string {
	status := " "
	if i.t.Done {
		status = "x"
	}
	return fmt.Sprintf("[%s] %s", status, i.t.Title)
}

func (i taskItem) Description() string {
	switch i.status {
	case statusDone:
		return "done"
	case statusReady:
		if len(i.t.BlockedBy) == 0 {
			return "ready — no blockers"
		}
		return fmt.Sprintf("ready — all %d blocker(s) done", len(i.t.BlockedBy))
	default:
		return fmt.Sprintf("blocked by %d unfinished task(s)", i.openBlockers)
	}
}

func (i taskItem) FilterValue() string { return i.t.Title }

// refreshTasks reloads both the all-tasks and ready-tasks lists from the
// store. Called after any mutation so the two views stay consistent.
func (m *Model) refreshTasks() {
	tasks, err := m.store.ListTasks(ctx())
	if err != nil {
		m.err = err
		return
	}
	ready, err := m.store.ReadyTasks(ctx())
	if err != nil {
		m.err = err
		return
	}
	m.snapshot = newSnapshot(tasks, ready)

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	m.taskList.SetItems(m.toTaskItems(tasks))

	sort.Slice(ready, func(i, j int) bool { return ready[i].CreatedAt.Before(ready[j].CreatedAt) })
	m.readyList.SetItems(m.toTaskItems(ready))

	// Keep an open detail view in sync with the underlying task's new state.
	if m.detailTask != nil {
		for _, t := range tasks {
			if t.ID == m.detailTask.ID {
				m.detailTask = t
			}
		}
	}
}

// toTaskItems wraps tasks as list items, stamping each with the status
// derived from the model's current snapshot.
func (m *Model) toTaskItems(tasks []*core.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{
			t:            t,
			status:       m.snapshot.statusOf(t),
			openBlockers: m.snapshot.openBlockers[t.ID],
		}
	}
	return items
}

func (m *Model) selectedTask() *core.Task {
	var l *list.Model
	switch m.screen {
	case screenTasks:
		l = &m.taskList
	case screenReady:
		l = &m.readyList
	default:
		return nil
	}
	item, ok := l.SelectedItem().(taskItem)
	if !ok {
		return nil
	}
	return item.t
}

func (m *Model) toggleSelectedTaskDone() {
	t := m.selectedTask()
	if t == nil {
		return
	}
	if t.Done {
		t.MarkUndone()
	} else {
		t.MarkDone()
	}
	if err := m.store.SaveTask(ctx(), t); err != nil {
		m.err = err
		return
	}
	m.refreshTasks()
}

func (m *Model) promptAddTask() {
	m.pendingInput = &inputRequest{
		prompt: "New task title:",
		onSubmit: func(m *Model, value string) {
			t, err := core.NewTask(core.TaskID(core.NewID()), value)
			if err != nil {
				m.err = err
				return
			}
			if err := m.store.SaveTask(ctx(), t); err != nil {
				m.err = err
				return
			}
			m.err = nil
			m.refreshTasks()
		},
	}
	m.input.Placeholder = "Write report"
	m.input.Focus()
}

func (m *Model) renderTaskDetail() string {
	t := m.detailTask
	if t == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(detailLabelStyle.Render(t.Title) + "\n\n")
	fmt.Fprintf(&b, "%s %v\n", detailLabelStyle.Render("Done:"), t.Done)

	notes := t.Notes
	if notes == "" {
		notes = "(none)"
	}
	fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Notes:"), notes)

	if len(t.BlockedBy) == 0 {
		fmt.Fprintf(&b, "%s (none)\n", detailLabelStyle.Render("Blocked by:"))
	} else {
		titles := make([]string, len(t.BlockedBy))
		for i, id := range t.BlockedBy {
			titles[i] = m.taskTitle(id)
		}
		fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Blocked by:"), strings.Join(titles, ", "))
	}

	return b.String()
}

// taskTitle resolves a TaskID to its title using the already-loaded task
// list, falling back to the raw ID if the task can't be found there.
func (m *Model) taskTitle(id core.TaskID) string {
	for _, item := range m.taskList.Items() {
		if ti, ok := item.(taskItem); ok && ti.t.ID == id {
			return ti.t.Title
		}
	}
	return string(id)
}
