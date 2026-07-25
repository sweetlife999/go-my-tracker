package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/sweetlife999/go-my-tracker/core"
)

// openStore resolves an app-private database path via Fyne's storage API
// (stable and correct across desktop/Android/iOS sandboxes, unlike
// core.DefaultDBPath's Linux-specific ~/.local/share assumption) and opens
// it. Requires the app to have been created with app.NewWithID so
// a.Storage().RootURI() is non-empty.
func openStore(a fyne.App) (*core.Store, error) {
	root := a.Storage().RootURI()
	dbURI, err := storage.Child(root, "tasks.db")
	if err != nil {
		return nil, fmt.Errorf("mobile: resolve db path: %w", err)
	}

	path := dbURI.Path()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mobile: create db directory: %w", err)
		}
	}

	return core.Open(path)
}
