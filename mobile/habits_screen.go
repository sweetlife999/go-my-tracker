package main

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// habitsScreen is the "Habits" tab: a list of habit cards with streaks,
// 7-day strips, and check-in buttons, plus a "+" button to add a habit.
type habitsScreen struct {
	store  *core.Store
	habits []*core.Habit

	onChanged func()

	list *widget.List
	root fyne.CanvasObject
}

func newHabitsScreen(store *core.Store, onAdd func(), onChanged func()) *habitsScreen {
	s := &habitsScreen{store: store, onChanged: onChanged}

	s.list = widget.NewList(
		func() int { return len(s.habits) },
		func() fyne.CanvasObject { return newHabitCard() },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(s.habits) {
				return
			}
			h := s.habits[id]
			card := obj.(*habitCard)
			card.update(h, func() { s.checkIn(h) })
		},
	)

	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), onAdd)
	top := container.NewBorder(nil, nil, nil, addBtn, widget.NewLabel("Habits"))
	s.root = container.NewBorder(top, nil, nil, nil, s.list)
	return s
}

func (s *habitsScreen) checkIn(h *core.Habit) {
	if err := s.store.CheckInHabit(context.Background(), h.ID, time.Now()); err != nil {
		return
	}
	s.Refresh()
	if s.onChanged != nil {
		s.onChanged()
	}
}

func (s *habitsScreen) Refresh() {
	habits, err := s.store.ListHabits(context.Background())
	if err != nil {
		habits = nil
	}
	s.habits = habits
	s.list.Refresh()
}
