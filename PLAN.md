# Task/Habit Tracker: Go TUI + Mobile GUI

## Context

Building a personal task tracker with three core features: tasks, habit tracking, and
task dependencies (tasks blocked by other tasks, i.e. a DAG). Two interfaces now ship
side by side, sharing `core`: a Linux terminal UI (TUI) and a Fyne-based mobile/desktop
GUI. There is still no CLI.

**Status:** `core` (domain model + SQLite persistence) is implemented and unit-tested,
the `tui` (Bubble Tea) is implemented and verified, and the `mobile` (Fyne) GUI is
implemented and verified. The project earlier had a cobra CLI and a first attempt at a
Fyne mobile GUI; both were dropped when the target narrowed to "one good terminal
interface." The Fyne GUI has since been revived as `mobile/`, reusing `core` completely
unchanged — exactly what keeping `core` interface-agnostic through that earlier pivot
was for.

## Architecture Decision: Bubble Tea

**Bubble Tea** (github.com/charmbracelet/bubbletea) is the TUI toolkit: an
Elm-architecture (Model-Update-View) framework, paired with `bubbles` (list, table,
viewport, textinput components) and `lipgloss` (styling). It runs directly in the
terminal — no GUI toolkit, no cgo, no packaging/SDK concerns.

## Architecture Decision: Fyne

**Fyne** (fyne.io/fyne/v2) is the mobile/desktop GUI toolkit: a pure-Go, cross-platform
widget toolkit that runs unmodified as a desktop window (`go run ./mobile`, used as the
dev loop in this environment) and compiles to Android/iOS via `fyne package`. It shares
`core`/`Store` directly with no service/API layer in between.

- **Theme**: `mobile/theme.go`'s `darkRedTheme` mirrors `tui/styles.go`'s palette
  (`colorDarkRed` #870000 as the sole accent, black background, white/gray neutrals) so
  the two interfaces read as the same product, rather than adopting the source design
  mockup's purple "Nocturne" accent.
- **DB path**: `core.DefaultDBPath()` is Linux-desktop-specific
  (`~/.local/share/tasktracker/tasks.db`), which doesn't hold on Android/iOS sandboxes.
  `mobile/store.go`'s `openStore` instead resolves an app-private path via
  `fyne.App.Storage().RootURI()` (requires `app.NewWithID`, not `app.New()`) and opens it
  through the same `core.Open(path string)` — no `core` change needed.
- **Navigation**: a 5-tab `container.NewAppTabs` (Tasks/Ready/Hub/Graph/Habits) at the
  bottom, plus a `sheetHost` (`mobile/nav.go`) that overlays task detail, the blocker
  picker, and add-task/habit sheets on top of the tabs via `container.NewStack`.
- **Dependency graph**: `mobile/graph_view.go`'s `depGraphWidget` is a custom
  `fyne.WidgetRenderer` — nodes positioned by dependency depth (a presentation-only
  level computation, not a `core` addition) with `canvas.Line` edges — plus a month
  calendar grid marking due dates.

## Domain Model (`core` package)

- **Task**: ID, Title, Notes, Done, CreatedAt, DueAt (optional), BlockedBy ([]TaskID)
- **Habit**: ID, Title, Frequency (daily/weekly/custom), CompletionLog ([]time.Time);
  streak is computed from the log, not stored
- **Dependency graph**: tasks form a DAG via BlockedBy edges
  - `AddDependency` rejects cycles (DFS-based cycle check)
  - `RemoveDependency` removes a blocker edge (no-op if absent)
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
    graph.go                # dependency graph, cycle detection, ReadyTasks(), RemoveDependency()
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
  mobile/
    main.go                 # Fyne app entrypoint: app.NewWithID, window, theme, nav
    store.go                  # mobile DB path via fyne.App.Storage().RootURI()
    theme.go                   # darkRedTheme (mirrors tui/styles.go's palette)
    status.go                   # deriveStatus: Done > Ready > Blocked labeling
    nav.go                       # sheetHost: modal sheet overlay plumbing
    tasks_screen.go               # Tasks tab: search + list + add
    ready_screen.go                 # Ready tab: filtered list
    hub_screen.go                    # Hub tab: today's stats card
    habits_screen.go                  # Habits tab: habit cards + check-in
    task_row.go                        # reusable task-row widget
    habit_card.go                       # reusable habit-card widget (7-day strip)
    task_list_view.go                    # shared list/search/empty-state machinery
    task_detail.go                        # detail sheet: done/due/notes/blockers
    blocker_picker.go                      # blocker-search sheet
    add_sheet.go                            # reusable add-task/add-habit sheet
    graph_view.go                            # dependency graph (custom renderer) + calendar
    mobile_test.go                           # fyne/v2/test-driver scripted walkthrough
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
6. ~~Add `core.Graph.RemoveDependency`/`core.Store.RemoveDependency`, with tests~~ — done
7. ~~Scaffold the Fyne `mobile/` GUI importing `core`, add the dark-red theme, and add
   `fyne.io/fyne/v2` to go.mod~~ — done
8. ~~Wire up the 5-tab nav (Tasks/Ready/Hub/Graph/Habits) and the modal sheets (task
   detail, blocker picker, add task/habit)~~ — done
9. ~~Build the dependency-graph custom widget and calendar grid~~ — done
10. ~~Manually verify the mobile GUI end-to-end~~ — done (see Verification)
11. Reference `samber/cc-skills-golang` (github.com/samber/cc-skills-golang) for Go
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
- `mobile/mobile_test.go` uses `fyne.io/fyne/v2/test`'s headless driver
  (`test.NewTempApp`/`test.NewTempWindow`) to script the same kind of walkthrough as the
  TUI's test, driving the real store + screens directly: add two tasks, block one by the
  other, confirm the Ready tab narrows, remove the dependency via the new
  `RemoveDependency` and confirm it unblocks, reject a self-block cycle, add a habit,
  check in, and confirm the Hub tab's counts update — done, passing
- `go run ./mobile` (or `make run-mobile`) launches a normal desktop window via Fyne's
  desktop driver — no Android/iOS SDK needed for this dev loop; real device packaging
  (`fyne package -os android`/`-os ios`) needs the Android SDK/NDK or macOS+Xcode
  respectively and is a separate follow-up, not part of this verification path
