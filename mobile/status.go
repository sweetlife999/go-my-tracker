package main

import "github.com/sweetlife999/go-my-tracker/core"

// taskStatus is the Done > Ready > Blocked label shown on a task row/detail,
// derived from core's existing readiness logic (never recomputed here).
type taskStatus int

const (
	statusBlocked taskStatus = iota
	statusReady
	statusDone
)

// deriveStatus classifies t using the set of currently-ready task IDs
// (as returned by core.Store.ReadyTasks), so the ready/blocked distinction
// always matches core.Graph.IsReady rather than a UI-side reimplementation.
func deriveStatus(t *core.Task, ready map[core.TaskID]bool) taskStatus {
	switch {
	case t.Done:
		return statusDone
	case ready[t.ID]:
		return statusReady
	default:
		return statusBlocked
	}
}

func (s taskStatus) label() string {
	switch s {
	case statusDone:
		return "Done"
	case statusReady:
		return "Ready"
	default:
		return "Blocked"
	}
}

func readySet(ready []*core.Task) map[core.TaskID]bool {
	set := make(map[core.TaskID]bool, len(ready))
	for _, t := range ready {
		set[t.ID] = true
	}
	return set
}
