package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// taskRow is a reusable list-row widget: checkbox, title/notes/due stack,
// and a status tag. Built as a plain composition (widget.BaseWidget +
// widget.NewSimpleRenderer over stock widgets) — nothing here needs custom
// hit-testing or freeform drawing, so no custom WidgetRenderer is warranted.
type taskRow struct {
	widget.BaseWidget

	check *widget.Check
	title *widget.Label
	notes *widget.Label
	due   *widget.Label
	tag   *widget.Label

	onToggle func(done bool)
}

func newTaskRow() *taskRow {
	r := &taskRow{
		check: widget.NewCheck("", nil),
		title: widget.NewLabel(""),
		notes: widget.NewLabel(""),
		due:   widget.NewLabel(""),
		tag:   widget.NewLabel(""),
	}
	r.tag.Alignment = fyne.TextAlignTrailing
	r.check.OnChanged = func(checked bool) {
		if r.onToggle != nil {
			r.onToggle(checked)
		}
	}
	r.ExtendBaseWidget(r)
	return r
}

func (r *taskRow) CreateRenderer() fyne.WidgetRenderer {
	center := container.NewVBox(r.title, r.notes, r.due)
	content := container.NewBorder(nil, nil, r.check, r.tag, center)
	return widget.NewSimpleRenderer(content)
}

// update binds t (with its precomputed status) into the row's widgets.
// onToggle is invoked with the new checked state when the checkbox is
// tapped; the caller is responsible for persisting it and refreshing.
func (r *taskRow) update(t *core.Task, status taskStatus, onToggle func(done bool)) {
	r.onToggle = nil // avoid firing a stale callback while we set state
	r.check.SetChecked(t.Done)
	r.onToggle = onToggle

	r.title.Text = t.Title
	if t.Done {
		r.title.Importance = widget.LowImportance
	} else {
		r.title.Importance = widget.MediumImportance
	}
	r.title.Refresh()

	if t.Notes != "" {
		r.notes.Text = t.Notes
		r.notes.Importance = widget.LowImportance
		r.notes.Show()
	} else {
		r.notes.Hide()
	}
	r.notes.Refresh()

	if t.DueAt != nil {
		overdue := t.IsOverdue(time.Now())
		prefix := "Due "
		if overdue {
			prefix = "Overdue · "
		}
		r.due.Text = prefix + t.DueAt.Format("Jan 2")
		if overdue {
			r.due.Importance = widget.DangerImportance
		} else {
			r.due.Importance = widget.LowImportance
		}
		r.due.Show()
	} else {
		r.due.Hide()
	}
	r.due.Refresh()

	r.tag.Text = status.label()
	switch status {
	case statusReady:
		r.tag.Importance = widget.HighImportance
	case statusDone:
		r.tag.Importance = widget.LowImportance
	default:
		r.tag.Importance = widget.MediumImportance
	}
	r.tag.Refresh()
}
