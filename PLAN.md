# Task/Habit Tracker: Go TUI

## Context

Building a personal task tracker with three core features: tasks, habit tracking, and
task dependencies (tasks blocked by other tasks, i.e. a DAG). Target platform: a Linux
terminal UI (TUI). There is no CLI and no mobile app — the TUI is the only interface.

**Status:** `core` (domain model + SQLite persistence) is implemented and unit-tested,
and the `tui` (Bubble Tea) is implemented and verified. The project originally also had
a cobra CLI and, before that, a Fyne mobile GUI; both were built and then dropped as the
target narrowed to "one good terminal interface." `core` was written to be
interface-agnostic throughout, so neither pivot required touching it.

## Architecture Decision: Bubble Tea

**Bubble Tea** (github.com/charmbracelet/bubbletea) is the TUI toolkit: an
Elm-architecture (Model-Update-View) framework, paired with `bubbles` (list, table,
viewport, textinput components) and `lipgloss` (styling). It runs directly in the
terminal — no GUI toolkit, no cgo, no packaging/SDK concerns.

## Domain Model (`core` package)

- **Task**: ID, Title, Notes, Done, CreatedAt, DueAt (optional), BlockedBy ([]TaskID)
- **Habit**: ID, Title, Frequency (daily/weekly/custom), CompletionLog ([]time.Time);
  streak is computed from the log, not stored
- **Dependency graph**: tasks form a DAG via BlockedBy edges
  - `AddDependency` rejects cycles (DFS-based cycle check)
  - `IsReady(task)` — true when all blockers are Done
  - `ReadyTasks()` — the "what can I actually work on" view, the TUI's default/second tab
    given branching logic is a headline feature

(Implemented and unit-tested.)

## Storage

SQLite via `modernc.org/sqlite` — pure Go, no cgo. One `store.go` schema/repository.
Default location: `core.DefaultDBPath()` → `~/.local/share/tasktracker/tasks.db`.
(Implemented.)

## Repo Layout

```
tasktracker/
  go.mod                  # single module
  core/
    task.go                # Task model + CRUD
    habit.go                # Habit model + streak calculation
    graph.go                # dependency graph, cycle detection, ReadyTasks()
    store.go                # sqlite repository + schema migration + DefaultDBPath()
    id.go                    # ID generator
    *_test.go               # unit tests per file, in-memory sqlite
  tui/
    main.go                 # Bubble Tea program entry point
    model.go                 # root model: view/tab switching, key bindings
    tasks.go                  # task list view + task detail/dependency view
    habits.go                  # habit tracker view
    styles.go                   # lipgloss styles
    walkthrough_test.go          # scripted end-to-end interaction test
  Makefile                  # build/test/lint targets
```

## Build Steps

1. ~~`git init`, `go mod init`, set up module structure~~ — done
2. ~~Implement `core`: Task, Habit, graph/dependency logic with unit tests~~ — done
3. ~~Add SQLite persistence (`store.go`), wire into `core`, with tests~~ — done
4. ~~Scaffold the Bubble Tea TUI importing `core`~~ — done: task list view with
   ready/blocked indicators, task detail view with dependency add, habit tracker view
   with streaks and check-in, key bindings (`tab` switch view, `a` add, `enter` detail/
   check-in, `d` toggle done, `b` add dependency, `q`/`ctrl+c` quit)
5. ~~Manually verify the TUI end-to-end~~ — done (see Verification)
6. Reference `samber/cc-skills-golang` (github.com/samber/cc-skills-golang) for Go
   conventions going forward

## Skills

Install/reference `samber/cc-skills-golang` for Go conventions (project layout, error
handling). No dedicated Bubble Tea Claude Code skill is known to exist — proceed using
Bubble Tea/bubbles/lipgloss's own docs and examples for that part.

## Verification

- Unit tests for `core` (graph cycle detection, ready-task computation, streak math) run
  via `go test ./...` — done
- `tui/walkthrough_test.go` scripts a full interaction end-to-end by driving the real
  `Update`/`View` code with the same keystrokes a user would type: add two tasks, block
  one by the other, confirm the Ready tab narrows to one task, mark the blocker done,
  confirm the other becomes ready, attempt a cycle and confirm the error surfaces in the
  UI, add a habit, check in, confirm the streak, check in again same day and confirm
  dedup — done, passing
- `go run ./tui` (or `make run-tui`) for interactive manual testing in a real terminal
