package core

import "testing"

func mustTask(t *testing.T, id TaskID, title string) *Task {
	t.Helper()
	tk, err := NewTask(id, title)
	if err != nil {
		t.Fatalf("NewTask(%q): %v", id, err)
	}
	return tk
}

func TestAddDependency_RejectsDirectCycle(t *testing.T) {
	a := mustTask(t, "a", "A")
	b := mustTask(t, "b", "B")
	g := NewGraph([]*Task{a, b})

	if err := g.AddDependency("a", "b"); err != nil {
		t.Fatalf("a blocked by b: unexpected error: %v", err)
	}
	if err := g.AddDependency("b", "a"); err == nil {
		t.Fatal("b blocked by a: expected cycle error, got nil")
	}
}

func TestAddDependency_RejectsIndirectCycle(t *testing.T) {
	a := mustTask(t, "a", "A")
	b := mustTask(t, "b", "B")
	c := mustTask(t, "c", "C")
	g := NewGraph([]*Task{a, b, c})

	// a <- b <- c (a blocked by b, b blocked by c)
	must(t, g.AddDependency("a", "b"))
	must(t, g.AddDependency("b", "c"))

	// c <- a would close the loop a -> b -> c -> a
	if err := g.AddDependency("c", "a"); err == nil {
		t.Fatal("expected indirect cycle to be rejected, got nil")
	}
}

func TestAddDependency_RejectsSelfBlock(t *testing.T) {
	a := mustTask(t, "a", "A")
	g := NewGraph([]*Task{a})
	if err := g.AddDependency("a", "a"); err == nil {
		t.Fatal("expected self-block to be rejected, got nil")
	}
}

func TestIsReady(t *testing.T) {
	a := mustTask(t, "a", "A")
	b := mustTask(t, "b", "B")
	g := NewGraph([]*Task{a, b})
	must(t, g.AddDependency("b", "a"))

	if g.IsReady("b") {
		t.Fatal("b should not be ready while a is not done")
	}
	if !g.IsReady("a") {
		t.Fatal("a should be ready: no blockers")
	}

	a.MarkDone()
	if !g.IsReady("b") {
		t.Fatal("b should be ready once a is done")
	}
	if g.IsReady("a") {
		t.Fatal("a is done, so it should no longer be 'ready to work on'")
	}
}

func TestRemoveDependency_RemovesEdge(t *testing.T) {
	a := mustTask(t, "a", "A")
	b := mustTask(t, "b", "B")
	g := NewGraph([]*Task{a, b})
	must(t, g.AddDependency("b", "a")) // b blocked by a

	g.RemoveDependency("b", "a")

	if len(b.BlockedBy) != 0 {
		t.Fatalf("expected b to have no blockers after removal, got %v", b.BlockedBy)
	}
	if !g.IsReady("b") {
		t.Fatal("b should be ready once its only blocker is removed")
	}
}

func TestRemoveDependency_NoopWhenAbsent(t *testing.T) {
	a := mustTask(t, "a", "A")
	b := mustTask(t, "b", "B")
	g := NewGraph([]*Task{a, b})

	g.RemoveDependency("b", "a") // no edge exists
	g.RemoveDependency("missing", "a")

	if len(b.BlockedBy) != 0 {
		t.Fatalf("expected no blockers, got %v", b.BlockedBy)
	}
}

func TestReadyTasks(t *testing.T) {
	a := mustTask(t, "a", "A")
	b := mustTask(t, "b", "B")
	c := mustTask(t, "c", "C")
	g := NewGraph([]*Task{a, b, c})
	must(t, g.AddDependency("b", "a")) // b blocked by a
	// c has no blockers

	ready := readyIDs(g)
	assertSameSet(t, ready, []TaskID{"a", "c"})

	a.MarkDone()
	ready = readyIDs(g)
	assertSameSet(t, ready, []TaskID{"b", "c"})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func readyIDs(g *Graph) []TaskID {
	var ids []TaskID
	for _, t := range g.ReadyTasks() {
		ids = append(ids, t.ID)
	}
	return ids
}

func assertSameSet(t *testing.T, got, want []TaskID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	set := make(map[TaskID]bool, len(got))
	for _, id := range got {
		set[id] = true
	}
	for _, id := range want {
		if !set[id] {
			t.Fatalf("got %v, want %v (missing %q)", got, want, id)
		}
	}
}
