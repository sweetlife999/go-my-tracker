package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"

	"github.com/sweetlife999/go-my-tracker/core"
)

// habitItem adapts *core.Habit to bubbles/list.Item.
type habitItem struct{ h *core.Habit }

func (i habitItem) Title() string {
	return fmt.Sprintf("%s  (streak: %d)", i.h.Title, i.h.Streak(time.Now()))
}

func (i habitItem) Description() string {
	if i.h.Frequency == core.Custom {
		return fmt.Sprintf("every %d days", i.h.IntervalDays)
	}
	return string(i.h.Frequency)
}

func (i habitItem) FilterValue() string { return i.h.Title }

func (m *Model) refreshHabits() {
	habits, err := m.store.ListHabits(ctx())
	if err != nil {
		m.err = err
		return
	}
	items := make([]list.Item, len(habits))
	for i, h := range habits {
		items[i] = habitItem{h}
	}
	m.habitList.SetItems(items)
}

func (m *Model) selectedHabit() *core.Habit {
	item, ok := m.habitList.SelectedItem().(habitItem)
	if !ok {
		return nil
	}
	return item.h
}

func (m *Model) checkInSelectedHabit() {
	h := m.selectedHabit()
	if h == nil {
		return
	}
	if err := m.store.CheckInHabit(ctx(), h.ID, time.Now()); err != nil {
		m.err = err
		return
	}
	m.err = nil
	m.refreshHabits()
}

func (m *Model) promptAddHabit() {
	m.pendingInput = &inputRequest{
		prompt: "New habit title (daily):",
		onSubmit: func(m *Model, value string) {
			h, err := core.NewHabit(core.HabitID(core.NewID()), value, core.Daily, 0)
			if err != nil {
				m.err = err
				return
			}
			if err := m.store.SaveHabit(ctx(), h); err != nil {
				m.err = err
				return
			}
			m.err = nil
			m.refreshHabits()
		},
	}
	m.input.Placeholder = "Meditate"
	m.input.Focus()
}
