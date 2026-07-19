package core

import "fmt"

// Graph holds a set of tasks and the BlockedBy edges between them, and
// answers dependency questions over that set (cycle detection, readiness).
type Graph struct {
	tasks map[TaskID]*Task
}

// NewGraph builds a Graph from a set of tasks. BlockedBy edges already
// present on the tasks are taken as-is; use AddDependency to add new edges
// with cycle checking.
func NewGraph(tasks []*Task) *Graph {
	g := &Graph{tasks: make(map[TaskID]*Task, len(tasks))}
	for _, t := range tasks {
		g.tasks[t.ID] = t
	}
	return g
}

// AddDependency records that task is blocked by blocker. It rejects the
// edge if it would introduce a cycle, or if either task is unknown to the
// graph.
func (g *Graph) AddDependency(task, blocker TaskID) error {
	if task == blocker {
		return fmt.Errorf("core: task %q cannot block itself", task)
	}
	t, ok := g.tasks[task]
	if !ok {
		return fmt.Errorf("core: unknown task %q", task)
	}
	if _, ok := g.tasks[blocker]; !ok {
		return fmt.Errorf("core: unknown task %q", blocker)
	}
	for _, existing := range t.BlockedBy {
		if existing == blocker {
			return nil // edge already present
		}
	}

	// Adding task -> blocker would cycle if blocker can already reach task.
	if g.reaches(blocker, task) {
		return fmt.Errorf("core: adding dependency %q -> %q would create a cycle", task, blocker)
	}

	t.BlockedBy = append(t.BlockedBy, blocker)
	return nil
}

// reaches reports whether a DFS from `from` can reach `to` via BlockedBy edges.
func (g *Graph) reaches(from, to TaskID) bool {
	visited := make(map[TaskID]bool)
	var visit func(id TaskID) bool
	visit = func(id TaskID) bool {
		if id == to {
			return true
		}
		if visited[id] {
			return false
		}
		visited[id] = true
		t, ok := g.tasks[id]
		if !ok {
			return false
		}
		for _, blocker := range t.BlockedBy {
			if visit(blocker) {
				return true
			}
		}
		return false
	}
	return visit(from)
}

// IsReady reports whether every task blocking the given task is Done.
// Unknown tasks are never ready.
func (g *Graph) IsReady(id TaskID) bool {
	t, ok := g.tasks[id]
	if !ok {
		return false
	}
	if t.Done {
		return false
	}
	for _, blockerID := range t.BlockedBy {
		blocker, ok := g.tasks[blockerID]
		if !ok || !blocker.Done {
			return false
		}
	}
	return true
}

// ReadyTasks returns the not-done tasks whose blockers are all Done, i.e.
// the tasks that can actually be worked on right now.
func (g *Graph) ReadyTasks() []*Task {
	var ready []*Task
	for id, t := range g.tasks {
		if g.IsReady(id) {
			ready = append(ready, t)
		}
	}
	return ready
}
