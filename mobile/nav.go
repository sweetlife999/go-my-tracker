package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// sheetHost layers a single modal sheet (task detail, blocker picker, add
// task/habit) over the main tab content, using a plain NewStack overlay
// rather than dialog.NewCustom so the sheet can own its full layout
// (scrollable body + footer buttons) instead of a fixed dialog frame.
type sheetHost struct {
	root    *fyne.Container // NewStack(main, overlay)
	overlay *fyne.Container // holds the current sheet content, or is empty
}

func newSheetHost(main fyne.CanvasObject) *sheetHost {
	overlay := container.NewStack()
	overlay.Hide()
	return &sheetHost{
		root:    container.NewStack(main, overlay),
		overlay: overlay,
	}
}

// Show replaces any current sheet with content and reveals the overlay.
func (h *sheetHost) Show(content fyne.CanvasObject) {
	h.overlay.RemoveAll()
	h.overlay.Add(content)
	h.overlay.Show()
}

// Hide clears and hides the overlay, returning to the plain tab content.
func (h *sheetHost) Hide() {
	h.overlay.Hide()
	h.overlay.RemoveAll()
}

// newModalPanel wraps sheet content in a dim backdrop plus a centered,
// padded panel, approximating the mockup's dialog-over-backdrop look.
func newModalPanel(content fyne.CanvasObject) fyne.CanvasObject {
	backdrop := canvas.NewRectangle(color.NRGBA{A: 160})
	panelBG := canvas.NewRectangle(colorFaint)
	panelBG.CornerRadius = 12
	panel := container.NewStack(panelBG, container.NewPadded(content))
	return container.NewStack(backdrop, container.NewPadded(panel))
}
