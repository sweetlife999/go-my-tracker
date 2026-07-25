package main

import (
	"context"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// taskListView is the shared list-of-tasks machinery behind the Tasks and
// Ready tabs: a widget.List of taskRow widgets backed by a caller-supplied
// loader (ListTasks vs ReadyTasks), an empty-state message, and an optional
// search filter. The "Ready" tag on a row is never hardcoded — it falls out
// of deriveStatus once a task is confirmed to be in the ready set, which is
// always true for every row the Ready tab shows.
type taskListView struct {
	store *core.Store
	load  func(ctx context.Context) ([]*core.Task, error)

	onOpenDetail func(*core.Task)
	onChanged    func() // notify the app so sibling screens (Hub, Graph, Ready) refresh

	query string

	all   []*core.Task
	ready map[core.TaskID]bool
	shown []*core.Task

	list       *widget.List
	emptyLabel *widget.Label
	stack      *fyne.Container
}

func newTaskListView(store *core.Store, load func(ctx context.Context) ([]*core.Task, error), emptyMessage string, onOpenDetail func(*core.Task), onChanged func()) *taskListView {
	v := &taskListView{
		store:        store,
		load:         load,
		onOpenDetail: onOpenDetail,
		onChanged:    onChanged,
	}

	v.emptyLabel = widget.NewLabel(emptyMessage)
	v.emptyLabel.Alignment = fyne.TextAlignCenter
	v.emptyLabel.Importance = widget.LowImportance

	v.list = widget.NewList(
		func() int { return len(v.shown) },
		func() fyne.CanvasObject { return newTaskRow() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(v.shown) {
				return
			}
			t := v.shown[id]
			row := obj.(*taskRow)
			status := deriveStatus(t, v.ready)
			row.update(t, status, func(bool) { v.toggleDone(t) })
		},
	)
	v.list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(v.shown) {
			return
		}
		t := v.shown[id]
		v.list.Unselect(id)
		if v.onOpenDetail != nil {
			v.onOpenDetail(t)
		}
	}

	v.stack = container.NewStack(v.list, container.NewCenter(v.emptyLabel))
	return v
}

func (v *taskListView) toggleDone(t *core.Task) {
	if t.Done {
		t.MarkUndone()
	} else {
		t.MarkDone()
	}
	_ = v.store.SaveTask(context.Background(), t)
	v.Refresh()
	if v.onChanged != nil {
		v.onChanged()
	}
}

// SetQuery applies a case-insensitive title substring filter; "" clears it.
func (v *taskListView) SetQuery(q string) {
	v.query = strings.ToLower(strings.TrimSpace(q))
	v.applyFilter()
}

// Refresh reloads tasks from the store and reapplies the current filter.
func (v *taskListView) Refresh() {
	tasks, err := v.load(context.Background())
	if err != nil {
		tasks = nil
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	v.all = tasks

	if readyTasks, err := v.store.ReadyTasks(context.Background()); err == nil {
		v.ready = readySet(readyTasks)
	}

	v.applyFilter()
}

func (v *taskListView) applyFilter() {
	if v.query == "" {
		v.shown = v.all
	} else {
		var filtered []*core.Task
		for _, t := range v.all {
			if strings.Contains(strings.ToLower(t.Title), v.query) {
				filtered = append(filtered, t)
			}
		}
		v.shown = filtered
	}

	v.list.Refresh()
	if len(v.shown) == 0 {
		v.emptyLabel.Show()
		v.list.Hide()
	} else {
		v.emptyLabel.Hide()
		v.list.Show()
	}
}
