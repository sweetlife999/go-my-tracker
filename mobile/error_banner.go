package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// errorBanner is a dismissible strip pinned above the tabs, used to report a
// failed store operation. A banner rather than a modal dialog: it doesn't
// interrupt what the user was typing, and it keeps mobile/ free of the
// fyne dialog package's extra module dependency.
type errorBanner struct {
	label *widget.Label
	root  *fyne.Container
}

func newErrorBanner() *errorBanner {
	b := &errorBanner{label: widget.NewLabel("")}
	b.label.Importance = widget.DangerImportance
	b.label.Wrapping = fyne.TextWrapWord

	dismiss := widget.NewButtonWithIcon("", theme.CancelIcon(), func() { b.Clear() })
	dismiss.Importance = widget.LowImportance

	b.root = container.NewBorder(nil, nil, nil, dismiss, b.label)
	b.root.Hide()
	return b
}

// Show reveals the banner with err's message. A nil error is ignored.
func (b *errorBanner) Show(err error) {
	if err == nil {
		return
	}
	b.label.SetText(err.Error())
	b.root.Show()
}

// Clear hides the banner.
func (b *errorBanner) Clear() {
	b.label.SetText("")
	b.root.Hide()
}
