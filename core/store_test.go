package core

import (
	"context"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStore_SaveAndListTasks(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	a := mustTask(t, "a", "Write plan")
	due := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	a.DueAt = &due
	must(t, s.SaveTask(ctx, a))

	got, err := s.GetTask(ctx, "a")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Title != "Write plan" || got.DueAt == nil || !got.DueAt.Equal(due) {
		t.Fatalf("round-tripped task mismatch: %+v", got)
	}

	a.MarkDone()
	must(t, s.SaveTask(ctx, a))
	got, err = s.GetTask(ctx, "a")
	if err != nil {
		t.Fatalf("GetTask after update: %v", err)
	}
	if !got.Done {
		t.Fatal("expected task to be persisted as done")
	}
}

func TestStore_AddDependency_PersistsAndRejectsCycles(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	must(t, s.SaveTask(ctx, mustTask(t, "a", "A")))
	must(t, s.SaveTask(ctx, mustTask(t, "b", "B")))

	must(t, s.AddDependency(ctx, "b", "a")) // b blocked by a

	if err := s.AddDependency(ctx, "a", "b"); err == nil {
		t.Fatal("expected cycle to be rejected")
	}

	tasks, err := s.ListTasks(ctx)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	var b *Task
	for _, tk := range tasks {
		if tk.ID == "b" {
			b = tk
		}
	}
	if b == nil || len(b.BlockedBy) != 1 || b.BlockedBy[0] != "a" {
		t.Fatalf("expected b to be persisted as blocked by a, got %+v", b)
	}
}

func TestStore_ReadyTasks(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	must(t, s.SaveTask(ctx, mustTask(t, "a", "A")))
	must(t, s.SaveTask(ctx, mustTask(t, "b", "B")))
	must(t, s.AddDependency(ctx, "b", "a"))

	ready, err := s.ReadyTasks(ctx)
	if err != nil {
		t.Fatalf("ReadyTasks: %v", err)
	}
	if len(ready) != 1 || ready[0].ID != "a" {
		t.Fatalf("expected only a to be ready, got %+v", ready)
	}

	a, err := s.GetTask(ctx, "a")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	a.MarkDone()
	must(t, s.SaveTask(ctx, a))

	ready, err = s.ReadyTasks(ctx)
	if err != nil {
		t.Fatalf("ReadyTasks after completing a: %v", err)
	}
	if len(ready) != 1 || ready[0].ID != "b" {
		t.Fatalf("expected only b to be ready, got %+v", ready)
	}
}

func TestStore_HabitCheckInAndStreak(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	h, err := NewHabit("h1", "Meditate", Daily, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}
	must(t, s.SaveHabit(ctx, h))

	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	must(t, s.CheckInHabit(ctx, "h1", base))
	must(t, s.CheckInHabit(ctx, "h1", base.AddDate(0, 0, 1)))
	must(t, s.CheckInHabit(ctx, "h1", base.AddDate(0, 0, 1))) // duplicate same-day check-in

	got, err := s.GetHabit(ctx, "h1")
	if err != nil {
		t.Fatalf("GetHabit: %v", err)
	}
	if len(got.CompletionLog) != 2 {
		t.Fatalf("CompletionLog len = %d, want 2 (dedup)", len(got.CompletionLog))
	}
	if streak := got.Streak(base.AddDate(0, 0, 1)); streak != 2 {
		t.Fatalf("streak = %d, want 2", streak)
	}
}
