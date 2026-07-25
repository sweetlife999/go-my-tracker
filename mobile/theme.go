package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Palette: dark red as the sole accent, plus black/white/gray — mirrors
// tui/styles.go's colorDarkRed/colorWhite/colorMuted/colorFaint so the
// mobile app and the TUI read as the same product.
var (
	colorDarkRed = color.NRGBA{R: 0x87, G: 0x00, B: 0x00, A: 0xFF} // #870000
	colorWhite   = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	colorBlack   = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	colorMuted   = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF} // dim gray
	colorFaint   = color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xFF} // near-black gray
)

// darkRedTheme is always dark (this app has no light mode, matching the
// TUI), so it ignores the requested fyne.ThemeVariant.
type darkRedTheme struct{}

func (darkRedTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameSelection, theme.ColorNameHyperlink:
		return colorDarkRed
	case theme.ColorNameBackground, theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		return colorBlack
	case theme.ColorNameForeground:
		return colorWhite
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder, theme.ColorNameDisabledButton:
		return colorMuted
	case theme.ColorNameInputBackground, theme.ColorNameButton, theme.ColorNameHeaderBackground:
		return colorFaint
	case theme.ColorNameInputBorder, theme.ColorNameSeparator:
		return colorFaint
	case theme.ColorNameError:
		// The TUI's errorStyle also uses the sole accent for errors — no
		// separate "danger" hue in this single-accent palette.
		return colorDarkRed
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (darkRedTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (darkRedTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (darkRedTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
