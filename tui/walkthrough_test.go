package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sweetlife999/go-my-tracker/core"
)

// type simulates a user typing a string followed by Enter.
func typeAndEnter(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func key(m *Model, s string) {
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

// pickBlockerByName simulates searching the blocker picker by title (the
// thing this test exists to cover): type the name to filter, Enter to
// apply the filter, Enter again to confirm the highlighted match.
func pickBlockerByName(m *Model, name string) {
	for _, r := range name {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // apply filter
	m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm selection
}

func TestWalkthrough_TasksDependencyAndHabit(t *testing.T) {
	store, err := core.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	m := NewModel(store)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})

	// Add task A and task B via the "a" prompt, same as a user would.
	key(m, "a")
	if m.pendingInput == nil {
		t.Fatal("expected 'a' to open the add-task prompt")
	}
	typeAndEnter(m, "Write plan")

	key(m, "a")
	typeAndEnter(m, "Build core")

	if len(m.taskList.Items()) != 2 {
		t.Fatalf("expected 2 tasks after adding, got %d", len(m.taskList.Items()))
	}

	// Ready view should show both tasks (no dependencies yet).
	m.screen = screenReady
	if len(m.readyList.Items()) != 2 {
		t.Fatalf("expected 2 ready tasks before blocking, got %d", len(m.readyList.Items()))
	}

	// Find task IDs the way a user would: read them off the list.
	items := m.taskList.Items()
	a := items[0].(taskItem).t
	b := items[1].(taskItem).t

	// Open detail on B, then block it by A ("enter" then "b").
	m.screen = screenTasks
	m.taskList.Select(1) // select B
	key(m, "enter")
	if !m.showDetail || m.detailTask.ID != b.ID {
		t.Fatalf("expected detail view open on B, got showDetail=%v task=%v", m.showDetail, m.detailTask)
	}
	key(m, "b")
	if !m.showBlockerPicker {
		t.Fatal("expected 'b' to open the blocker picker")
	}
	pickBlockerByName(m, a.Title)
	if m.err != nil {
		t.Fatalf("unexpected error after adding dependency by searching for %q: %v", a.Title, m.err)
	}
	if m.showBlockerPicker {
		t.Fatal("expected picker to close after a successful selection")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // back out of detail

	// Ready view should now show only A.
	m.screen = screenReady
	m.refreshTasks()
	if got := len(m.readyList.Items()); got != 1 {
		t.Fatalf("expected 1 ready task after blocking B by A, got %d", got)
	}
	if readyTitle := m.readyList.Items()[0].(taskItem).t.ID; readyTitle != a.ID {
		t.Fatalf("expected A to be the only ready task, got %v", readyTitle)
	}

	// Mark A done via the list-view "d" shortcut; B should become ready.
	m.screen = screenTasks
	m.taskList.Select(0) // select A
	key(m, "d")
	if m.err != nil {
		t.Fatalf("unexpected error marking A done: %v", m.err)
	}

	m.screen = screenReady
	if got := len(m.readyList.Items()); got != 1 {
		t.Fatalf("expected 1 ready task after A done, got %d", got)
	}
	if readyID := m.readyList.Items()[0].(taskItem).t.ID; readyID != b.ID {
		t.Fatalf("expected B to be the only ready task now, got %v", readyID)
	}

	// Attempting to block A by B now would be a cycle; confirm the UI surfaces the error.
	m.screen = screenTasks
	m.taskList.Select(0)
	key(m, "enter")
	key(m, "b")
	pickBlockerByName(m, b.Title)
	if m.err == nil {
		t.Fatal("expected cycle error to be surfaced in m.err")
	}
	if !strings.Contains(m.err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got: %v", m.err)
	}
	// The picker should stay open on error so the user can try another task.
	if !m.showBlockerPicker {
		t.Fatal("expected picker to remain open after a rejected (cyclic) selection")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // clear the picker's filter
	m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // close the picker
	if m.showBlockerPicker {
		t.Fatal("expected picker to be closed after clearing filter then esc again")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // back out of detail
	if m.showDetail {
		t.Fatal("expected detail view to be closed")
	}

	// --- Habit walkthrough ---
	m.screen = screenHabits
	key(m, "a")
	typeAndEnter(m, "Meditate")
	if len(m.habitList.Items()) != 1 {
		t.Fatalf("expected 1 habit after adding, got %d", len(m.habitList.Items()))
	}

	m.habitList.Select(0)
	key(m, "c")
	if m.err != nil {
		t.Fatalf("unexpected error checking in: %v", m.err)
	}
	h := m.habitList.Items()[0].(habitItem).h
	if streak := h.Streak(time.Now()); streak != 1 {
		t.Fatalf("expected streak 1 after check-in, got %d", streak)
	}

	// Checking in again the same day should not double the streak (dedup).
	key(m, "c")
	h = m.habitList.Items()[0].(habitItem).h
	if streak := h.Streak(time.Now()); streak != 1 {
		t.Fatalf("expected streak to stay 1 after duplicate same-day check-in, got %d", streak)
	}

	t.Logf("final view render:\n%s", m.View())
}
