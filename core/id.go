package core

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns a short, sufficiently-unique hex identifier suitable for a
// TaskID or HabitID. Callers assign it explicitly (see NewTask/NewHabit)
// rather than core generating IDs implicitly.
func NewID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand failing means the system entropy source is broken
	}
	return hex.EncodeToString(b[:])
}
