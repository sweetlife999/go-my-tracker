package main

import (
	"context"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// notesFlushDelay is how long typing must pause before pending notes are
// written. widget.Entry offers no focus-lost callback, so a debounce is the
// only way to avoid a database write per keystroke; every navigation path
// out of the sheet flushes synchronously so an edit can't be lost.
var notesFlushDelay = 750 * time.Millisecond

// taskDetail is the task-detail sheet: done toggle, status tag, an editable
// due date (widget.DateEntry), an editable notes field, removable blocker
// chips, and an "Add dependency" button.
type taskDetail struct {
	root fyne.CanvasObject

	notesEntry *widget.Entry

	mu      sync.Mutex
	timer   *time.Timer
	pending bool // notes differ from what's on disk
	saveFn  func()
}

// newTaskDetail builds the detail sheet for t.
//
// onMutated is called after the Done toggle or a blocker removal — both
// change the task's status/blocker list, so the caller re-derives status
// from the store and rebuilds the whole sheet rather than patching it in
// place. Due-date edits don't affect status, so they save inline without
// rebuilding; notes edits are debounced (see notesFlushDelay).
//
// onError receives any failed store write; nothing here discards one.
func newTaskDetail(store *core.Store, t *core.Task, allTasks []*core.Task, status taskStatus,
	onOpenBlockerPicker func(target *core.Task),
	onMutated func(t *core.Task),
	onClose func(),
	onError func(error)) *taskDetail {

	ctx := context.Background()
	d := &taskDetail{}

	save := func() {
		if err := store.SaveTask(ctx, t); err != nil {
			onError(err)
		}
	}
	d.saveFn = save

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
		// Clear any pending notes edit, then persist title/done/notes/due
		// in the single write this toggle already needs.
		d.discardPending()
		save()
		onMutated(t)
	})
	doneCheck.SetChecked(t.Done)

	dueEntry := widget.NewDateEntry()
	dueEntry.SetDate(t.DueAt)
	dueEntry.OnChanged = func(due *time.Time) {
		t.DueAt = due
		d.discardPending()
		save()
	}
	clearDue := widget.NewButton("Clear", func() {
		t.DueAt = nil
		dueEntry.SetDate(nil)
		d.discardPending()
		save()
	})
	clearDue.Importance = widget.LowImportance

	d.notesEntry = widget.NewMultiLineEntry()
	d.notesEntry.SetPlaceHolder("Add notes…")
	d.notesEntry.SetText(t.Notes)
	d.notesEntry.OnChanged = func(v string) {
		t.Notes = v
		d.scheduleFlush()
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
				d.flush()
				if err := store.RemoveDependency(ctx, t.ID, bid); err != nil {
					onError(err)
					return
				}
				onMutated(t)
			})
			removeBtn.Importance = widget.LowImportance
			blockersBox.Add(container.NewBorder(nil, nil, nil, removeBtn, widget.NewLabel(label)))
		}
	}

	addDepBtn := widget.NewButton("Add dependency", func() {
		d.flush()
		onOpenBlockerPicker(t)
	})
	closeBtn := widget.NewButton("Done", func() {
		d.flush()
		onClose()
	})
	closeBtn.Importance = widget.HighImportance

	body := container.NewVBox(
		container.NewBorder(nil, nil, nil, doneCheck, titleLabel),
		tag,
		widget.NewSeparator(),
		widget.NewLabel("Due date"),
		container.NewBorder(nil, nil, nil, clearDue, dueEntry),
		widget.NewLabel("Notes"),
		d.notesEntry,
		widget.NewLabel("Blocked by"),
		blockersBox,
		addDepBtn,
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), closeBtn),
	)

	d.root = container.NewVScroll(body)
	return d
}

// scheduleFlush (re)arms the debounce timer for a pending notes edit.
func (d *taskDetail) scheduleFlush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.pending = true
	if d.timer != nil {
		d.timer.Stop()
	}
	// The timer fires on its own goroutine, so hop back onto Fyne's thread
	// before running anything that touches the store or widgets.
	d.timer = time.AfterFunc(notesFlushDelay, func() { fyne.Do(d.flush) })
}

// flush writes a pending notes edit immediately and cancels the timer. Safe
// to call when nothing is pending. Every path that navigates away from the
// sheet calls this so a debounced edit is never dropped.
func (d *taskDetail) flush() {
	if d.discardPending() {
		d.saveFn()
	}
}

// discardPending stops the debounce timer and reports whether an unsaved
// notes edit was outstanding.
func (d *taskDetail) discardPending() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	pending := d.pending
	d.pending = false
	return pending
}
