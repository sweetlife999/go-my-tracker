package main

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// newTaskDetail builds the task-detail sheet: done toggle, status tag, an
// editable due date (widget.DateEntry — Task.DueAt/SaveTask already
// round-trip it, this is new UI only), an editable notes field
// (widget.NewMultiLineEntry — same, new UI only), removable blocker chips,
// and an "Add dependency" button.
//
// onMutated is called after the Done toggle or a blocker removal — both
// change the task's status/blocker list, so the caller re-derives status
// from the store and rebuilds the whole sheet rather than patching it in
// place. Due-date and notes edits don't affect status, so they save inline
// without rebuilding.
func newTaskDetail(store *core.Store, t *core.Task, allTasks []*core.Task, status taskStatus,
	onOpenBlockerPicker func(target *core.Task),
	onMutated func(t *core.Task),
	onClose func()) fyne.CanvasObject {

	ctx := context.Background()

	titleLabel := widget.NewLabel(t.Title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	tag := widget.NewLabel(status.label())
	switch status {
	case statusReady:
		tag.Importance = widget.HighImportance
	case statusDone:
		tag.Importance = widget.LowImportance
	default:
		tag.Importance = widget.MediumImportance
	}

	doneCheck := widget.NewCheck("Done", func(checked bool) {
		if checked {
			t.MarkDone()
		} else {
			t.MarkUndone()
		}
		_ = store.SaveTask(ctx, t)
		onMutated(t)
	})
	doneCheck.SetChecked(t.Done)

	dueEntry := widget.NewDateEntry()
	dueEntry.SetDate(t.DueAt)
	dueEntry.OnChanged = func(d *time.Time) {
		t.DueAt = d
		_ = store.SaveTask(ctx, t)
	}
	clearDue := widget.NewButton("Clear", func() {
		t.DueAt = nil
		dueEntry.SetDate(nil)
		_ = store.SaveTask(ctx, t)
	})
	clearDue.Importance = widget.LowImportance

	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetPlaceHolder("Add notes…")
	notesEntry.SetText(t.Notes)
	notesEntry.OnChanged = func(v string) {
		t.Notes = v
		_ = store.SaveTask(ctx, t)
	}

	titleByID := make(map[core.TaskID]string, len(allTasks))
	for _, other := range allTasks {
		titleByID[other.ID] = other.Title
	}

	blockersBox := container.NewVBox()
	if len(t.BlockedBy) == 0 {
		none := widget.NewLabel("Nothing blocking this task.")
		none.Importance = widget.LowImportance
		blockersBox.Add(none)
	} else {
		for _, bid := range t.BlockedBy {
			bid := bid
			label := titleByID[bid]
			if label == "" {
				label = string(bid)
			}
			removeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
				_ = store.RemoveDependency(ctx, t.ID, bid)
				onMutated(t)
			})
			removeBtn.Importance = widget.LowImportance
			blockersBox.Add(container.NewBorder(nil, nil, nil, removeBtn, widget.NewLabel(label)))
		}
	}

	addDepBtn := widget.NewButton("Add dependency", func() { onOpenBlockerPicker(t) })
	closeBtn := widget.NewButton("Done", onClose)
	closeBtn.Importance = widget.HighImportance

	body := container.NewVBox(
		container.NewBorder(nil, nil, nil, doneCheck, titleLabel),
		tag,
		widget.NewSeparator(),
		widget.NewLabel("Due date"),
		container.NewBorder(nil, nil, nil, clearDue, dueEntry),
		widget.NewLabel("Notes"),
		notesEntry,
		widget.NewLabel("Blocked by"),
		blockersBox,
		addDepBtn,
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), closeBtn),
	)

	return container.NewVScroll(body)
}
