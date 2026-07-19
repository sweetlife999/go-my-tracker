package core

import (
	"errors"
	"sort"
	"time"
)

// HabitID uniquely identifies a Habit.
type HabitID string

// Frequency is how often a habit is expected to be checked in on.
type Frequency string

const (
	Daily  Frequency = "daily"
	Weekly Frequency = "weekly"
	Custom Frequency = "custom"
)

// Habit tracks recurring check-ins. Streak is derived from CompletionLog,
// never stored directly.
type Habit struct {
	ID            HabitID
	Title         string
	Frequency     Frequency
	IntervalDays  int // only used when Frequency == Custom
	CompletionLog []time.Time
}

var (
	ErrHabitEmptyTitle     = errors.New("core: habit title must not be empty")
	ErrInvalidFrequency    = errors.New("core: invalid frequency")
	ErrCustomNeedsInterval = errors.New("core: custom frequency requires IntervalDays > 0")
)

// NewHabit constructs a Habit. For Custom frequency, intervalDays must be > 0
// and is ignored otherwise.
func NewHabit(id HabitID, title string, freq Frequency, intervalDays int) (*Habit, error) {
	if title == "" {
		return nil, ErrHabitEmptyTitle
	}
	switch freq {
	case Daily, Weekly:
		intervalDays = 0
	case Custom:
		if intervalDays <= 0 {
			return nil, ErrCustomNeedsInterval
		}
	default:
		return nil, ErrInvalidFrequency
	}
	return &Habit{
		ID:           id,
		Title:        title,
		Frequency:    freq,
		IntervalDays: intervalDays,
	}, nil
}

// intervalDays returns the number of days that may elapse between two
// consecutive check-ins without breaking the streak.
func (h *Habit) intervalDays() int {
	switch h.Frequency {
	case Weekly:
		return 7
	case Custom:
		return h.IntervalDays
	default: // Daily
		return 1
	}
}

// CheckIn records a completion at t. Multiple check-ins on the same
// calendar day are deduplicated.
func (h *Habit) CheckIn(t time.Time) {
	day := truncateToDay(t)
	for _, existing := range h.CompletionLog {
		if truncateToDay(existing).Equal(day) {
			return
		}
	}
	h.CompletionLog = append(h.CompletionLog, day)
}

// Streak computes the current streak as of asOf: the number of consecutive
// completions, spaced no more than the habit's interval apart, ending at
// the most recent completion. If the most recent completion is further
// than one interval in the past relative to asOf, the streak is broken and
// Streak returns 0.
func (h *Habit) Streak(asOf time.Time) int {
	if len(h.CompletionLog) == 0 {
		return 0
	}

	days := make([]time.Time, len(h.CompletionLog))
	for i, t := range h.CompletionLog {
		days[i] = truncateToDay(t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].After(days[j]) }) // descending

	interval := h.intervalDays()
	today := truncateToDay(asOf)

	if daysBetween(days[0], today) > interval {
		return 0
	}

	streak := 1
	for i := 1; i < len(days); i++ {
		if daysBetween(days[i], days[i-1]) > interval {
			break
		}
		streak++
	}
	return streak
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func daysBetween(earlier, later time.Time) int {
	return int(later.Sub(earlier).Hours() / 24)
}
