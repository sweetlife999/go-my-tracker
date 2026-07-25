package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// hubScreen is the "Hub" tab: a single stats card summarizing today's
// progress. Purely derived/read-only — no new core capability needed.
type hubScreen struct {
	store      *core.Store
	habitsLine *widget.Label
	tasksLine  *widget.Label
	root       fyne.CanvasObject
}

func newHubScreen(store *core.Store) *hubScreen {
	s := &hubScreen{store: store}
	s.habitsLine = widget.NewLabel("")
	s.tasksLine = widget.NewLabel("")
	card := widget.NewCard("Today", "", container.NewVBox(s.habitsLine, s.tasksLine))
	s.root = container.NewVBox(card)
	return s
}

func (s *hubScreen) Refresh() {
	ctx := context.Background()

	habits, err := s.store.ListHabits(ctx)
	if err != nil {
		habits = nil
	}
	checked := 0
	now := time.Now()
	for _, h := range habits {
		if habitCheckedOnDay(h, now) {
			checked++
		}
	}
	s.habitsLine.SetText(fmt.Sprintf("%d / %d Habits closed", checked, len(habits)))

	tasks, err := s.store.ListTasks(ctx)
	if err != nil {
		tasks = nil
	}
	done := 0
	for _, t := range tasks {
		if t.Done {
			done++
		}
	}
	s.tasksLine.SetText(fmt.Sprintf("%d / %d Tasks done", done, len(tasks)))
}
