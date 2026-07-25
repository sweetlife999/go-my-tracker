package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// newAddSheet builds the reusable "New task"/"New habit" sheet: a title
// entry plus Cancel/Add buttons. Enter submits, matching the mockup.
func newAddSheet(title, placeholder string, onSubmit func(value string), onCancel func()) fyne.CanvasObject {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(placeholder)

	submit := func() {
		value := strings.TrimSpace(entry.Text)
		if value == "" {
			return
		}
		onSubmit(value)
	}
	entry.OnSubmitted = func(string) { submit() }

	cancelBtn := widget.NewButton("Cancel", onCancel)
	addBtn := widget.NewButton("Add", submit)
	addBtn.Importance = widget.HighImportance

	footer := container.NewHBox(cancelBtn, layout.NewSpacer(), addBtn)

	heading := widget.NewLabel(title)
	heading.TextStyle = fyne.TextStyle{Bold: true}

	return container.NewVBox(heading, entry, footer)
}
