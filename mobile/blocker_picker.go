package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

// newBlockerPicker builds the "Block by…" search sheet, mirroring
// tui/model.go's openBlockerPicker: search-filtered list of every other
// task not already a blocker, tapping one calls the existing
// store.AddDependency (a cycle rejection is shown inline, same message
// style as the TUI). onDone is called after a successful add; onBack
// returns to the detail sheet without picking anything; onError receives a
// failed store read.
func newBlockerPicker(store *core.Store, target *core.Task, onDone func(), onBack func(), onError func(error)) fyne.CanvasObject {
	ctx := context.Background()

	all, err := store.ListTasks(ctx)
	if err != nil {
		onError(err)
		all = nil
	}
	blockedSet := make(map[core.TaskID]bool, len(target.BlockedBy))
	for _, id := range target.BlockedBy {
		blockedSet[id] = true
	}
	var candidates []*core.Task
	for _, t := range all {
		if t.ID != target.ID && !blockedSet[t.ID] {
			candidates = append(candidates, t)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CreatedAt.Before(candidates[j].CreatedAt) })

	shown := candidates

	errLabel := widget.NewLabel("")
	errLabel.Importance = widget.DangerImportance
	errLabel.Wrapping = fyne.TextWrapWord
	errLabel.Hide()

	emptyLabel := widget.NewLabel("No matching tasks.")
	emptyLabel.Alignment = fyne.TextAlignCenter
	emptyLabel.Importance = widget.LowImportance

	var list *widget.List
	list = widget.NewList(
		func() int { return len(shown) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(shown) {
				return
			}
			obj.(*widget.Label).SetText(shown[id].Title)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(shown) {
			return
		}
		candidate := shown[id]
		if err := store.AddDependency(ctx, target.ID, candidate.ID); err != nil {
			errLabel.SetText(fmt.Sprintf("Can't block %q by %q: it would create a dependency cycle.", target.Title, candidate.Title))
			errLabel.Show()
			list.Unselect(id)
			return
		}
		onDone()
	}

	listStack := container.NewStack(list, container.NewCenter(emptyLabel))
	updateEmpty := func() {
		if len(shown) == 0 {
			emptyLabel.Show()
			list.Hide()
		} else {
			emptyLabel.Hide()
			list.Show()
		}
	}
	updateEmpty()

	search := widget.NewEntry()
	search.SetPlaceHolder("Search tasks")
	search.OnChanged = func(q string) {
		q = strings.ToLower(strings.TrimSpace(q))
		if q == "" {
			shown = candidates
		} else {
			var filtered []*core.Task
			for _, t := range candidates {
				if strings.Contains(strings.ToLower(t.Title), q) {
					filtered = append(filtered, t)
				}
			}
			shown = filtered
		}
		list.Refresh()
		updateEmpty()
	}

	heading := widget.NewLabel(fmt.Sprintf("Block %q by…", target.Title))
	heading.TextStyle = fyne.TextStyle{Bold: true}

	backBtn := widget.NewButton("Back", onBack)

	return container.NewBorder(
		container.NewVBox(heading, search, errLabel), backBtn, nil, nil,
		listStack,
	)
}
