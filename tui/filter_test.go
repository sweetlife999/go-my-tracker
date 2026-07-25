package main

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sweetlife999/go-my-tracker/core"
)

// send delivers msg to the model and then drains the commands it returns,
// the way tea's runtime would. bubbles/list computes filter matches in a
// command, so without this the filter never narrows.
func send(t *testing.T, m *Model, msg tea.Msg) {
	t.Helper()
	pending := []tea.Msg{msg}
	for depth := 0; len(pending) > 0 && depth < 50; depth++ {
		next := pending[0]
		pending = pending[1:]

		_, cmd := m.Update(next)
		if cmd == nil {
			continue
		}
		switch out := runCmd(cmd).(type) {
		case nil:
		case tea.BatchMsg:
			for _, c := range out {
				if produced := runCmd(c); produced != nil {
					pending = append(pending, produced)
				}
			}
		default:
			pending = append(pending, out)
		}
	}
}

// runCmd executes cmd, giving up on anything that doesn't produce a message
// promptly. Timer-based commands (cursor blink) would otherwise sleep on
// every keystroke and never settle; the messages this drain cares about
// (list.FilterMatchesMsg) are produced synchronously.
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func typeFilter(t *testing.T, m *Model, s string) {
	t.Helper()
	for _, r := range s {
		send(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// newFilterTestModel returns a model with two tasks whose titles contain the
// letters bound to browsing shortcuts ("a" add, "d" toggle done, "q" quit),
// which is exactly what made the filter unusable.
func newFilterTestModel(t *testing.T) *Model {
	t.Helper()
	store, err := core.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	m := NewModel(store)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	key(m, "a")
	typeAndEnter(m, "Quarterly audit")
	key(m, "a")
	typeAndEnter(m, "Ship release")
	return m
}

func TestFilter_TypingGoesToTheFilterNotTheShortcuts(t *testing.T) {
	m := newFilterTestModel(t)

	key(m, "/")
	if got := m.taskList.FilterState(); got != list.Filtering {
		t.Fatalf("FilterState after '/' = %v, want Filtering", got)
	}

	typeFilter(t, m, "audit")
	if m.pendingInput != nil {
		t.Fatal("'a' while filtering opened the add-task prompt instead of filtering")
	}
	if got := m.taskList.FilterInput.Value(); got != "audit" {
		t.Fatalf("filter input = %q, want %q", got, "audit")
	}

	// Enter applies the filter and narrows the visible items to the match.
	send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.showDetail {
		t.Fatal("enter while filtering opened the detail view instead of applying the filter")
	}
	if got := len(m.taskList.VisibleItems()); got != 1 {
		t.Fatalf("visible items after applying filter = %d, want 1", got)
	}
	if got := m.taskList.VisibleItems()[0].(taskItem).t.Title; got != "Quarterly audit" {
		t.Fatalf("filtered to %q, want %q", got, "Quarterly audit")
	}

	// Esc clears the filter and restores the full list.
	send(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if got := len(m.taskList.VisibleItems()); got != 2 {
		t.Fatalf("visible items after clearing filter = %d, want 2", got)
	}
}

func TestFilter_QDoesNotQuitWhileFiltering(t *testing.T) {
	m := newFilterTestModel(t)

	key(m, "/")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Fatal("typing 'q' into the filter quit the program")
		}
	}
	if got := m.taskList.FilterInput.Value(); got != "q" {
		t.Fatalf("filter input = %q, want %q", got, "q")
	}

	// ctrl+c must still quit, even mid-filter.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c while filtering did not quit")
	}
	if _, isQuit := cmd().(tea.QuitMsg); !isQuit {
		t.Fatal("ctrl+c while filtering did not produce QuitMsg")
	}
}

func TestFilter_ShortcutsStillWorkOnceFilterIsApplied(t *testing.T) {
	m := newFilterTestModel(t)

	key(m, "/")
	typeFilter(t, m, "audit")
	send(t, m, tea.KeyMsg{Type: tea.KeyEnter}) // apply

	// With the filter applied (not being typed into), "d" is a shortcut again.
	key(m, "d")
	if m.err != nil {
		t.Fatalf("unexpected error toggling done on the filtered task: %v", m.err)
	}
	tasks, err := m.store.ListTasks(ctx())
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	for _, task := range tasks {
		wantDone := task.Title == "Quarterly audit"
		if task.Done != wantDone {
			t.Fatalf("task %q Done = %v, want %v", task.Title, task.Done, wantDone)
		}
	}
}
