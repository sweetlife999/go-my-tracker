package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// readyScreen is the "Ready" tab: tasks with no open blockers, i.e. what
// can actually be worked on right now. No search box, no add button.
type readyScreen struct {
	view *taskListView
	root fyne.CanvasObject
}

func newReadyScreen(store *core.Store, onOpenDetail func(*core.Task), onChanged func(), onError func(error)) *readyScreen {
	s := &readyScreen{}
	s.view = newTaskListView(store, store.ReadyTasks, "Nothing is unblocked right now.", onOpenDetail, onChanged, onError)

	caption := widget.NewLabel("Tasks with no open blockers — what you can actually work on right now.")
	caption.Wrapping = fyne.TextWrapWord
	caption.Importance = widget.LowImportance

	s.root = container.NewBorder(caption, nil, nil, nil, s.view.stack)
	return s
}

func (s *readyScreen) Refresh() { s.view.Refresh() }
