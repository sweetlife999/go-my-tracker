package core

import (
	"context"
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

// TestStore_CheckInHabit_DedupsAcrossTimezoneOffsets covers the offset bug:
// the same calendar day submitted from two different zones (e.g. a user who
// travelled, or a DST change between check-ins) must be one completion, not
// two. Storing an offset-bearing timestamp made these distinct primary keys
// and so inflated the streak.
func TestStore_CheckInHabit_DedupsAcrossTimezoneOffsets(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	h, err := NewHabit("h1", "Meditate", Daily, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}
	must(t, s.SaveHabit(ctx, h))

	berlin := time.FixedZone("CEST", 2*60*60)
	losAngeles := time.FixedZone("PDT", -7*60*60)

	must(t, s.CheckInHabit(ctx, "h1", time.Date(2026, 7, 20, 9, 0, 0, 0, berlin)))
	must(t, s.CheckInHabit(ctx, "h1", time.Date(2026, 7, 20, 21, 0, 0, 0, losAngeles)))

	got, err := s.GetHabit(ctx, "h1")
	if err != nil {
		t.Fatalf("GetHabit: %v", err)
	}
	if len(got.CompletionLog) != 1 {
		t.Fatalf("CompletionLog len = %d, want 1 (same calendar day from two offsets)", len(got.CompletionLog))
	}
	if y, m, d := got.CompletionLog[0].Date(); y != 2026 || m != time.July || d != 20 {
		t.Fatalf("stored completion date = %04d-%02d-%02d, want 2026-07-20", y, m, d)
	}
	if streak := got.Streak(time.Date(2026, 7, 20, 23, 0, 0, 0, berlin)); streak != 1 {
		t.Fatalf("streak = %d, want 1", streak)
	}
}

// TestHabit_Streak_WeeklyGapAcrossDSTBoundary covers the arithmetic bug:
// daysBetween used to divide a wall-clock duration by 24h, so a gap spanning
// a spring-forward transition measured 6 days and 23 hours -> 6, masking a
// weekly streak that had actually just been kept (or broken) on the boundary.
func TestHabit_Streak_WeeklyGapAcrossDSTBoundary(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}

	h, err := NewHabit("h1", "Review week", Weekly, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}

	// 2026-03-29 is the spring-forward date in Europe/Berlin.
	before := time.Date(2026, 3, 25, 10, 0, 0, 0, berlin)
	h.CheckIn(before)
	h.CheckIn(before.AddDate(0, 0, 7)) // 2026-04-01, 6d23h later in wall-clock terms

	if got := daysBetween(truncateToDay(before), truncateToDay(before.AddDate(0, 0, 7))); got != 7 {
		t.Fatalf("daysBetween across spring-forward = %d, want 7", got)
	}
	if got := h.Streak(before.AddDate(0, 0, 7)); got != 2 {
		t.Fatalf("weekly streak across DST boundary = %d, want 2", got)
	}

	// One day past the interval must still break the streak.
	h2, err := NewHabit("h2", "Review week", Weekly, 0)
	if err != nil {
		t.Fatalf("NewHabit: %v", err)
	}
	h2.CheckIn(before)
	h2.CheckIn(before.AddDate(0, 0, 8))
	if got := h2.Streak(before.AddDate(0, 0, 8)); got != 1 {
		t.Fatalf("weekly streak with an 8-day gap = %d, want 1 (streak broken)", got)
	}
}
