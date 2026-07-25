package main

import (
	"context"
	"errors"
	"testing"
	"time"
	"unicode/utf8"

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
	test.NewTempWindow(t, a.root)
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

// TestTruncateRunes covers the graph-label bug: the node label used to be
// byte-sliced, which cuts multibyte characters in half and renders mojibake.
func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
	}{
		{"short ascii is untouched", "Ship it", 16},
		{"exact length is untouched", "0123456789abcdef", 16},
		{"long ascii is cut with an ellipsis", "Write the quarterly report", 16},
		{"cyrillic is cut on a rune boundary", "Написать план на квартал", 16},
		{"emoji is cut on a rune boundary", "🚀🚀🚀🚀🚀", 3},
		{"a one-character budget is just the ellipsis", "Ship it", 1},
		{"a zero budget does not panic", "Ship it", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateRunes(tc.in, tc.max)
			if !utf8.ValidString(got) {
				t.Fatalf("truncateRunes(%q, %d) = %q, which is not valid UTF-8", tc.in, tc.max, got)
			}
			if n := utf8.RuneCountInString(got); n > tc.max {
				t.Fatalf("truncateRunes(%q, %d) = %q, %d runes (over the limit)", tc.in, tc.max, got, n)
			}
		})
	}

	// The specific regression: a Cyrillic title must not be corrupted.
	const title = "Написать план на квартал"
	got := truncateRunes(title, graphLabelRunes)
	if !utf8.ValidString(got) {
		t.Fatalf("Cyrillic title truncated to invalid UTF-8: %q", got)
	}
	if want := "Написать план н…"; got != want {
		t.Fatalf("truncateRunes(%q) = %q, want %q", title, got, want)
	}
}

// newDetailForTest builds a task-detail sheet over a stored task, returning
// it alongside the store and a recorder of any reported errors.
func newDetailForTest(t *testing.T, task *core.Task) (*taskDetail, *core.Store, *[]error) {
	t.Helper()
	a, store := newTestApplication(t)

	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	a.refreshAll()

	var reported []error
	d := newTaskDetail(store, task, []*core.Task{task}, statusReady,
		func(*core.Task) {},
		func(*core.Task) {},
		func() {},
		func(err error) { reported = append(reported, err) },
	)
	return d, store, &reported
}

func storedNotes(t *testing.T, store *core.Store, id core.TaskID) string {
	t.Helper()
	got, err := store.GetTask(context.Background(), id)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	return got.Notes
}

// TestTaskDetail_NotesAreNotWrittenPerKeystroke covers the write-amplification
// bug: notes used to call SaveTask from OnChanged, one database write per
// character typed.
func TestTaskDetail_NotesAreNotWrittenPerKeystroke(t *testing.T) {
	task, err := core.NewTask(core.TaskID(core.NewID()), "Draft proposal")
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	d, store, reported := newDetailForTest(t, task)

	// Simulate typing: OnChanged fires per keystroke.
	for _, prefix := range []string{"C", "Ca", "Cal", "Call", "Call ", "Call Bo", "Call Bob"} {
		d.notesEntry.SetText(prefix)
	}

	if got := storedNotes(t, store, task.ID); got != "" {
		t.Fatalf("notes were written mid-typing: stored %q, want %q", got, "")
	}

	// Navigating away flushes exactly once.
	d.flush()
	if got := storedNotes(t, store, task.ID); got != "Call Bob" {
		t.Fatalf("notes after flush = %q, want %q", got, "Call Bob")
	}
	if len(*reported) != 0 {
		t.Fatalf("unexpected errors reported: %v", *reported)
	}

	// A second flush with nothing pending must not write again.
	d.flush()
	if got := storedNotes(t, store, task.ID); got != "Call Bob" {
		t.Fatalf("notes after redundant flush = %q, want unchanged", got)
	}
}

// TestTaskDetail_DebounceTimerFlushes confirms the pending edit lands on its
// own once typing stops, not only when the user navigates away.
func TestTaskDetail_DebounceTimerFlushes(t *testing.T) {
	original := notesFlushDelay
	notesFlushDelay = 10 * time.Millisecond
	t.Cleanup(func() { notesFlushDelay = original })

	task, err := core.NewTask(core.TaskID(core.NewID()), "Draft proposal")
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	d, store, _ := newDetailForTest(t, task)

	d.notesEntry.SetText("Ship it")

	deadline := time.Now().Add(2 * time.Second)
	for storedNotes(t, store, task.ID) != "Ship it" {
		if time.Now().After(deadline) {
			t.Fatalf("debounced notes never reached the store; stored %q", storedNotes(t, store, task.ID))
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestErrorBanner_ShowsAndClears covers the swallowed-error bug: a failed
// store call must be visible instead of silently discarded.
func TestErrorBanner_ShowsAndClears(t *testing.T) {
	a, _ := newTestApplication(t)

	if a.errBar.root.Visible() {
		t.Fatal("error banner should start hidden")
	}

	a.reportError(nil) // a nil error is not an error
	if a.errBar.root.Visible() {
		t.Fatal("reporting nil should not show the banner")
	}

	a.reportError(errors.New("disk on fire"))
	if !a.errBar.root.Visible() {
		t.Fatal("reported error did not show the banner")
	}
	if got := a.errBar.label.Text; got != "disk on fire" {
		t.Fatalf("banner text = %q, want %q", got, "disk on fire")
	}

	a.errBar.Clear()
	if a.errBar.root.Visible() {
		t.Fatal("banner should be hidden after Clear")
	}
}

// TestToggleDone_ReportsStoreFailure checks the toggle path surfaces a write
// failure rather than showing a checkbox change that never persisted.
func TestToggleDone_ReportsStoreFailure(t *testing.T) {
	a, store := newTestApplication(t)

	task, err := core.NewTask(core.TaskID(core.NewID()), "Draft proposal")
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	a.refreshAll()

	// Closing the store makes every subsequent write fail.
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	a.tasks.view.toggleDone(task)

	if !a.errBar.root.Visible() {
		t.Fatal("a failed SaveTask was swallowed instead of shown to the user")
	}
}
