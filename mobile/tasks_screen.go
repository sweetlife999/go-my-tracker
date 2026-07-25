package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// tasksScreen is the "Tasks" tab: a search box over the full task list,
// with a "+" button to add a new task.
type tasksScreen struct {
	view *taskListView
	root fyne.CanvasObject
}

func newTasksScreen(store *core.Store, onOpenDetail func(*core.Task), onAdd func(), onChanged func()) *tasksScreen {
	s := &tasksScreen{}
	s.view = newTaskListView(store, store.ListTasks, "No tasks yet — tap + to add one.", onOpenDetail, onChanged)

	search := widget.NewEntry()
	search.SetPlaceHolder("Search tasks")
	search.OnChanged = s.view.SetQuery

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), onAdd)

	top := container.NewBorder(nil, nil, nil, addBtn, search)
	s.root = container.NewBorder(top, nil, nil, nil, s.view.stack)
	return s
}

func (s *tasksScreen) Refresh() { s.view.Refresh() }
