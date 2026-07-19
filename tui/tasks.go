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

// taskItem adapts *core.Task to bubbles/list.Item.
type taskItem struct{ t *core.Task }

func (i taskItem) Title() string {
	status := " "
	if i.t.Done {
		status = "x"
	}
	return fmt.Sprintf("[%s] %s", status, i.t.Title)
}

func (i taskItem) Description() string {
	if len(i.t.BlockedBy) == 0 {
		return "no blockers"
	}
	return fmt.Sprintf("blocked by %d task(s)", len(i.t.BlockedBy))
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
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	m.taskList.SetItems(toTaskItems(tasks))

	ready, err := m.store.ReadyTasks(ctx())
	if err != nil {
		m.err = err
		return
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i].CreatedAt.Before(ready[j].CreatedAt) })
	m.readyList.SetItems(toTaskItems(ready))

	// Keep an open detail view in sync with the underlying task's new state.
	if m.detailTask != nil {
		for _, t := range tasks {
			if t.ID == m.detailTask.ID {
				m.detailTask = t
			}
		}
	}
}

func toTaskItems(tasks []*core.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{t}
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

func (m *Model) promptAddDependency() {
	if m.detailTask == nil {
		return
	}
	task := m.detailTask
	m.pendingInput = &inputRequest{
		prompt: fmt.Sprintf("Block %q by task ID:", task.Title),
		onSubmit: func(m *Model, value string) {
			if err := m.store.AddDependency(ctx(), task.ID, core.TaskID(value)); err != nil {
				m.err = err
				return
			}
			m.err = nil
			m.refreshTasks()
		},
	}
	m.input.Placeholder = "e.g. 3159557c"
	m.input.Focus()
}

func (m *Model) renderTaskDetail() string {
	t := m.detailTask
	if t == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(detailLabelStyle.Render(t.Title) + "\n\n")
	fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("ID:"), t.ID)
	fmt.Fprintf(&b, "%s %v\n", detailLabelStyle.Render("Done:"), t.Done)

	notes := t.Notes
	if notes == "" {
		notes = "(none)"
	}
	fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Notes:"), notes)

	if len(t.BlockedBy) == 0 {
		fmt.Fprintf(&b, "%s (none)\n", detailLabelStyle.Render("Blocked by:"))
	} else {
		ids := make([]string, len(t.BlockedBy))
		for i, id := range t.BlockedBy {
			ids[i] = string(id)
		}
		fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Blocked by:"), strings.Join(ids, ", "))
	}

	return b.String()
}
