package core

import (
	"testing"
	"time"
)

func TestNewTask_RejectsEmptyTitle(t *testing.T) {
	if _, err := NewTask("t1", "   "); err == nil {
		t.Fatal("expected error for blank title")
	}
}

func TestTask_MarkDoneUndone(t *testing.T) {
	tk := mustTask(t, "t1", "Write plan")
	if tk.Done {
		t.Fatal("new task should not be done")
	}
	tk.MarkDone()
	if !tk.Done {
		t.Fatal("expected task to be done")
	}
	tk.MarkUndone()
	if tk.Done {
		t.Fatal("expected task to be reopened")
	}
}

func TestTask_IsOverdue(t *testing.T) {
	tk := mustTask(t, "t1", "Ship it")
	now := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC)

	if tk.IsOverdue(now) {
		t.Fatal("task with no due date should never be overdue")
	}

	past := now.AddDate(0, 0, -1)
	tk.DueAt = &past
	if !tk.IsOverdue(now) {
		t.Fatal("task with past due date should be overdue")
	}

	tk.MarkDone()
	if tk.IsOverdue(now) {
		t.Fatal("done tasks are never overdue")
	}
}
