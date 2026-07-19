package core

import (
	"testing"
	"time"
)

func day(offset int) time.Time {
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, offset)
}

func TestHabit_Streak_DailyIncrements(t *testing.T) {
	h, err := NewHabit("h1", "Meditate", Daily, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}

	h.CheckIn(day(0))
	if got := h.Streak(day(0)); got != 1 {
		t.Fatalf("streak after day 0 = %d, want 1", got)
	}

	h.CheckIn(day(1))
	h.CheckIn(day(2))
	if got := h.Streak(day(2)); got != 3 {
		t.Fatalf("streak after 3 consecutive days = %d, want 3", got)
	}
}

func TestHabit_Streak_DailyResetsOnMissedDay(t *testing.T) {
	h, _ := NewHabit("h1", "Meditate", Daily, 0)
	h.CheckIn(day(0))
	h.CheckIn(day(1))
	// day 2 missed entirely
	h.CheckIn(day(3))

	if got := h.Streak(day(3)); got != 1 {
		t.Fatalf("streak after missed day = %d, want 1 (reset)", got)
	}
}

func TestHabit_Streak_DailyToleratesCheckingTodayLater(t *testing.T) {
	h, _ := NewHabit("h1", "Meditate", Daily, 0)
	h.CheckIn(day(0))
	h.CheckIn(day(1))

	// Asking "as of day 2" (haven't checked in yet today) should not have
	// broken the streak yet, since day1->day2 gap is exactly one interval.
	if got := h.Streak(day(2)); got != 2 {
		t.Fatalf("streak as of day 2 (not yet checked in) = %d, want 2", got)
	}
}

func TestHabit_Streak_WeeklyToleratesWeekLongGap(t *testing.T) {
	h, err := NewHabit("h1", "Deep clean", Weekly, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}
	h.CheckIn(day(0))
	h.CheckIn(day(7))
	h.CheckIn(day(14))

	if got := h.Streak(day(14)); got != 3 {
		t.Fatalf("weekly streak = %d, want 3", got)
	}

	// A 10-day gap exceeds the 7-day weekly interval: streak resets.
	h.CheckIn(day(24))
	if got := h.Streak(day(24)); got != 1 {
		t.Fatalf("weekly streak after >7 day gap = %d, want 1 (reset)", got)
	}
}

func TestHabit_Streak_CustomInterval(t *testing.T) {
	h, err := NewHabit("h1", "Water plants", Custom, 3)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}
	h.CheckIn(day(0))
	h.CheckIn(day(3))
	h.CheckIn(day(6))

	if got := h.Streak(day(6)); got != 3 {
		t.Fatalf("custom streak = %d, want 3", got)
	}
}

func TestHabit_Streak_NoCompletions(t *testing.T) {
	h, _ := NewHabit("h1", "Meditate", Daily, 0)
	if got := h.Streak(day(0)); got != 0 {
		t.Fatalf("streak with no completions = %d, want 0", got)
	}
}

func TestHabit_CheckIn_DedupesSameDay(t *testing.T) {
	h, _ := NewHabit("h1", "Meditate", Daily, 0)
	h.CheckIn(day(0))
	h.CheckIn(day(0).Add(3 * time.Hour))
	if len(h.CompletionLog) != 1 {
		t.Fatalf("CompletionLog len = %d, want 1 (same-day dedup)", len(h.CompletionLog))
	}
}

func TestNewHabit_CustomRequiresPositiveInterval(t *testing.T) {
	if _, err := NewHabit("h1", "X", Custom, 0); err == nil {
		t.Fatal("expected error for Custom frequency with IntervalDays 0")
	}
}

func TestNewHabit_RejectsEmptyTitle(t *testing.T) {
	if _, err := NewHabit("h1", "", Daily, 0); err == nil {
		t.Fatal("expected error for empty title")
	}
}
