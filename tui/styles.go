package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Palette: dark red as the sole accent, plus black/white/gray. Grays are
// tints of black/white for de-emphasized text, not a separate hue.
var (
	colorDarkRed = lipgloss.Color("88") // #870000
	colorWhite   = lipgloss.Color("255")
	colorBlack   = lipgloss.Color("0")
	colorMuted   = lipgloss.Color("245") // dim gray, for de-emphasized text
	colorFaint   = lipgloss.Color("237") // near-black gray, for the least important text
)

var (
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 2).
			Foreground(colorWhite).
			Background(colorDarkRed)

	tabInactiveStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(colorMuted).
				Background(colorBlack)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(1, 0, 0, 0)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorDarkRed).
			Padding(0, 1)

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDarkRed)

	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorDarkRed).
			Padding(0, 1)
)

// newStyledList builds a list.Model with the dark red/white/black palette
// applied throughout — title bar, selection highlight, pagination dots,
// filter prompt — replacing bubbles' default purple accent.
func newStyledList(title string) list.Model {
	delegate := list.NewDefaultDelegate()

	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorDarkRed).
		Foreground(colorWhite).
		Background(colorDarkRed).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorDarkRed).
		Foreground(colorWhite).
		Background(colorDarkRed)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(colorWhite)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(colorMuted)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.Foreground(colorFaint)
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.Foreground(colorFaint)
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.Foreground(colorDarkRed).Bold(true)

	l := list.New(nil, delegate, 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.Styles.Title = l.Styles.Title.Background(colorDarkRed).Foreground(colorWhite)
	l.Styles.StatusBar = l.Styles.StatusBar.Foreground(colorMuted)
	l.Styles.StatusEmpty = l.Styles.StatusEmpty.Foreground(colorMuted)
	l.Styles.StatusBarActiveFilter = l.Styles.StatusBarActiveFilter.Foreground(colorWhite)
	l.Styles.NoItems = l.Styles.NoItems.Foreground(colorMuted)
	l.Styles.FilterPrompt = l.Styles.FilterPrompt.Foreground(colorDarkRed)
	l.Styles.FilterCursor = l.Styles.FilterCursor.Foreground(colorDarkRed)
	l.Styles.ActivePaginationDot = l.Styles.ActivePaginationDot.Foreground(colorDarkRed)
	l.Styles.InactivePaginationDot = l.Styles.InactivePaginationDot.Foreground(colorFaint)

	return l
}
