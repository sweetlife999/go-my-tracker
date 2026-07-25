package main

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/sweetlife999/go-my-tracker/core"
)

// newTestApplication builds an application against an in-memory store and
// a headless test window, mirroring how tui/walkthrough_test.go drives the
// real Update/View code directly instead of mocking it.
func newTestApplication(t *testing.T) (*application, *core.Store) {
	t.Helper()
	test.NewTempApp(t)

	store, err := core.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	a := newApplication(store)
	test.NewTempWindow(t, a.sheets.root)
	return a, store
}

func TestWalkthrough_TasksReadyDependenciesAndHabits(t *testing.T) {
	ctx := context.Background()
	a, store := newTestApplication(t)

	t1, err := core.NewTask(core.TaskID(core.NewID()), "Draft proposal")
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	if err := store.SaveTask(ctx, t1); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	t2, err := core.NewTask(core.TaskID(core.NewID()), "Get sign-off")
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	if err := store.SaveTask(ctx, t2); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	a.refreshAll()

	if got := len(a.ready.view.shown); got != 2 {
		t.Fatalf("expected 2 ready tasks before blocking, got %d", got)
	}

	// Block t2 by t1, mirroring the blocker-picker's AddDependency call.
	if err := store.AddDependency(ctx, t2.ID, t1.ID); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}
	a.refreshAll()
	if got := a.ready.view.shown; len(got) != 1 || got[0].ID != t1.ID {
		t.Fatalf("expected only t1 ready after blocking, got %+v", got)
	}

	// Remove the dependency via the new RemoveDependency and confirm it unblocks.
	if err := store.RemoveDependency(ctx, t2.ID, t1.ID); err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}
	a.refreshAll()
	if got := len(a.ready.view.shown); got != 2 {
		t.Fatalf("expected both tasks ready again after removing dependency, got %d", got)
	}

	// A cycle attempt should still be rejected end-to-end through the store.
	if err := store.AddDependency(ctx, t1.ID, t1.ID); err == nil {
		t.Fatal("expected self-block to be rejected")
	}

	// Add a habit and check in; confirm the Hub tab's counts update.
	h, err := core.NewHabit(core.HabitID(core.NewID()), "Meditate", core.Daily, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}
	if err := store.SaveHabit(ctx, h); err != nil {
		t.Fatalf("SaveHabit: %v", err)
	}
	if err := store.CheckInHabit(ctx, h.ID, time.Now()); err != nil {
		t.Fatalf("CheckInHabit: %v", err)
	}
	a.refreshAll()

	const wantHabits = "1 / 1 Habits closed"
	if got := a.hub.habitsLine.Text; got != wantHabits {
		t.Fatalf("hub habits line = %q, want %q", got, wantHabits)
	}
	const wantTasks = "0 / 2 Tasks done"
	if got := a.hub.tasksLine.Text; got != wantTasks {
		t.Fatalf("hub tasks line = %q, want %q", got, wantTasks)
	}

	// Mark a task done and confirm Hub picks it up too.
	t1.MarkDone()
	if err := store.SaveTask(ctx, t1); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	a.refreshAll()
	const wantTasksAfterDone = "1 / 2 Tasks done"
	if got := a.hub.tasksLine.Text; got != wantTasksAfterDone {
		t.Fatalf("hub tasks line after done = %q, want %q", got, wantTasksAfterDone)
	}
}

func TestDeriveStatus(t *testing.T) {
	ctx := context.Background()
	_, store := newTestApplication(t)

	t1, _ := core.NewTask(core.TaskID(core.NewID()), "A")
	t2, _ := core.NewTask(core.TaskID(core.NewID()), "B")
	if err := store.SaveTask(ctx, t1); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTask(ctx, t2); err != nil {
		t.Fatal(err)
	}
	if err := store.AddDependency(ctx, t2.ID, t1.ID); err != nil {
		t.Fatal(err)
	}

	ready, err := store.ReadyTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	set := readySet(ready)

	if got := deriveStatus(t1, set); got != statusReady {
		t.Fatalf("t1 status = %v, want statusReady", got)
	}
	if got := deriveStatus(t2, set); got != statusBlocked {
		t.Fatalf("t2 status = %v, want statusBlocked", got)
	}

	t1.MarkDone()
	if got := deriveStatus(t1, set); got != statusDone {
		t.Fatalf("done task status = %v, want statusDone", got)
	}
}

func TestComputeLevels(t *testing.T) {
	a := &core.Task{ID: "a"}
	b := &core.Task{ID: "b", BlockedBy: []core.TaskID{"a"}}
	c := &core.Task{ID: "c", BlockedBy: []core.TaskID{"a", "b"}}

	level := computeLevels([]*core.Task{a, b, c})
	if level["a"] != 0 {
		t.Fatalf("level[a] = %d, want 0", level["a"])
	}
	if level["b"] != 1 {
		t.Fatalf("level[b] = %d, want 1", level["b"])
	}
	if level["c"] != 2 {
		t.Fatalf("level[c] = %d, want 2 (max(level[a], level[b]) + 1)", level["c"])
	}
}
