// Command mobile is a Fyne-based mobile/desktop GUI for the task/habit
// tracker, sharing core with the tui package. Both interfaces are kept —
// this is additive, not a replacement.
package main

import (
	"context"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/sweetlife999/go-my-tracker/core"
)

// application ties the store, the tab screens, and the modal-sheet
// plumbing together — the mobile counterpart of tui's root Model.
type application struct {
	store  *core.Store
	sheets *sheetHost

	tasks  *tasksScreen
	ready  *readyScreen
	hub    *hubScreen
	graph  *graphScreen
	habits *habitsScreen
}

func newApplication(store *core.Store) *application {
	a := &application{store: store}

	a.tasks = newTasksScreen(store,
		func(t *core.Task) { a.openDetail(t.ID) },
		func() { a.openAddTaskSheet() },
		a.refreshAll,
	)
	a.ready = newReadyScreen(store,
		func(t *core.Task) { a.openDetail(t.ID) },
		a.refreshAll,
	)
	a.hub = newHubScreen(store)
	a.graph = newGraphScreen(store)
	a.habits = newHabitsScreen(store,
		func() { a.openAddHabitSheet() },
		a.refreshAll,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Tasks", a.tasks.root),
		container.NewTabItem("Ready", a.ready.root),
		container.NewTabItem("Hub", a.hub.root),
		container.NewTabItem("Graph", a.graph.root),
		container.NewTabItem("Habits", a.habits.root),
	)
	tabs.SetTabLocation(container.TabLocationBottom)

	a.sheets = newSheetHost(tabs)
	a.refreshAll()
	return a
}

// refreshAll reloads every screen from the store. Called after any
// mutation (add/toggle/dependency/check-in) rather than trying to track
// exactly which screens are affected — the dataset here is small enough
// that this is simpler and safer than partial invalidation.
func (a *application) refreshAll() {
	a.tasks.Refresh()
	a.ready.Refresh()
	a.hub.Refresh()
	a.graph.Refresh()
	a.habits.Refresh()
}

func (a *application) openDetail(id core.TaskID) {
	ctx := context.Background()
	t, err := a.store.GetTask(ctx, id)
	if err != nil {
		return
	}
	allTasks, _ := a.store.ListTasks(ctx)
	readyTasks, _ := a.store.ReadyTasks(ctx)
	status := deriveStatus(t, readySet(readyTasks))

	content := newTaskDetail(a.store, t, allTasks, status,
		func(target *core.Task) { a.openBlockerPicker(target.ID) },
		func(mutated *core.Task) {
			a.refreshAll()
			a.openDetail(mutated.ID)
		},
		func() {
			a.refreshAll()
			a.sheets.Hide()
		},
	)
	a.sheets.Show(newModalPanel(content))
}

func (a *application) openBlockerPicker(id core.TaskID) {
	target, err := a.store.GetTask(context.Background(), id)
	if err != nil {
		return
	}
	content := newBlockerPicker(a.store, target,
		func() { a.openDetail(id) },
		func() { a.openDetail(id) },
	)
	a.sheets.Show(newModalPanel(content))
}

func (a *application) openAddTaskSheet() {
	content := newAddSheet("New task", "Write report", func(value string) {
		t, err := core.NewTask(core.TaskID(core.NewID()), value)
		if err != nil {
			return
		}
		if err := a.store.SaveTask(context.Background(), t); err != nil {
			return
		}
		a.sheets.Hide()
		a.refreshAll()
	}, func() { a.sheets.Hide() })
	a.sheets.Show(newModalPanel(content))
}

func (a *application) openAddHabitSheet() {
	content := newAddSheet("New habit", "Meditate", func(value string) {
		h, err := core.NewHabit(core.HabitID(core.NewID()), value, core.Daily, 0)
		if err != nil {
			return
		}
		if err := a.store.SaveHabit(context.Background(), h); err != nil {
			return
		}
		a.sheets.Hide()
		a.refreshAll()
	}, func() { a.sheets.Hide() })
	a.sheets.Show(newModalPanel(content))
}

func main() {
	fyneApp := app.NewWithID("com.sweetlife999.tasktracker")
	fyneApp.Settings().SetTheme(darkRedTheme{})

	store, err := openStore(fyneApp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open store:", err)
		os.Exit(1)
	}
	defer store.Close()

	appModel := newApplication(store)

	w := fyneApp.NewWindow("Task Tracker")
	w.SetContent(appModel.sheets.root)
	w.Resize(fyne.NewSize(400, 800))
	w.ShowAndRun()
}
