package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

const habitStripDays = 7
const habitStripSquare = 14

// habitCard is a reusable list-row widget for one habit: title, streak
// text, a 7-day completion strip, and a check-in button. Plain composition
// (widget.BaseWidget + widget.NewSimpleRenderer) — the strip's squares are
// static canvas.Rectangles rebuilt on refresh, no custom renderer needed.
type habitCard struct {
	widget.BaseWidget

	title   *widget.Label
	streak  *widget.Label
	squares [habitStripDays]*canvas.Rectangle
	checkIn *widget.Button

	onCheckIn func()
}

func newHabitCard() *habitCard {
	c := &habitCard{
		title:   widget.NewLabel(""),
		streak:  widget.NewLabel(""),
		checkIn: widget.NewButton("Check in", nil),
	}
	c.streak.Importance = widget.LowImportance
	for i := range c.squares {
		sq := canvas.NewRectangle(colorFaint)
		sq.SetMinSize(fyne.NewSize(habitStripSquare, habitStripSquare))
		c.squares[i] = sq
	}
	c.checkIn.OnTapped = func() {
		if c.onCheckIn != nil {
			c.onCheckIn()
		}
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *habitCard) CreateRenderer() fyne.WidgetRenderer {
	strip := container.NewHBox()
	for _, sq := range c.squares {
		strip.Add(sq)
	}
	header := container.NewBorder(nil, nil, nil, c.checkIn,
		container.NewVBox(c.title, c.streak))
	content := widget.NewCard("", "", container.NewVBox(header, strip))
	return widget.NewSimpleRenderer(content)
}

// update binds h into the card's widgets. onCheckIn is invoked when the
// check-in button is tapped; the caller persists it and refreshes.
func (c *habitCard) update(h *core.Habit, onCheckIn func()) {
	c.title.SetText(h.Title)

	streak := h.Streak(time.Now())
	c.streak.SetText(fmt.Sprintf("%d day streak · %s", streak, h.Frequency))

	log := lastNDays(h, habitStripDays)
	for i, done := range log {
		if done {
			c.squares[i].FillColor = colorDarkRed
		} else {
			c.squares[i].FillColor = colorFaint
		}
		c.squares[i].Refresh()
	}

	checkedToday := habitCheckedOnDay(h, time.Now())
	c.onCheckIn = nil
	if checkedToday {
		c.checkIn.SetText("Checked in")
		c.checkIn.Disable()
	} else {
		c.checkIn.SetText("Check in")
		c.checkIn.Enable()
	}
	c.onCheckIn = onCheckIn
}

// lastNDays returns, oldest-first, whether h was checked in on each of the
// last n calendar days (today last).
func lastNDays(h *core.Habit, n int) [habitStripDays]bool {
	var out [habitStripDays]bool
	now := time.Now()
	for i := 0; i < n; i++ {
		day := now.AddDate(0, 0, -(n - 1 - i))
		out[i] = habitCheckedOnDay(h, day)
	}
	return out
}

func habitCheckedOnDay(h *core.Habit, day time.Time) bool {
	y1, m1, d1 := day.Date()
	for _, c := range h.CompletionLog {
		y2, m2, d2 := c.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 {
			return true
		}
	}
	return false
}
