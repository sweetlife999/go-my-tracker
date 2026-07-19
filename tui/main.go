// Command tui is a terminal UI for the task/habit tracker, built with
// Bubble Tea.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sweetlife999/go-my-tracker/core"
)

func main() {
	path := core.DefaultDBPath()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "create db directory:", err)
			os.Exit(1)
		}
	}

	store, err := core.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open store:", err)
		os.Exit(1)
	}
	defer store.Close()

	if _, err := tea.NewProgram(NewModel(store), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
