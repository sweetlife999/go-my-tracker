package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sweetlife999/go-my-tracker/core"
)

// screen identifies which of the three tabbed views is active.
type screen int

const (
	screenTasks screen = iota
	screenReady
	screenHabits
)

func (s screen) String() string {
	switch s {
	case screenTasks:
		return "Tasks"
	case screenReady:
		return "Ready"
	case screenHabits:
		return "Habits"
	default:
		return ""
	}
}

// inputRequest describes a pending single-line prompt (add task, add habit,
// add dependency) that the model routes keystrokes to until Enter/Esc.
type inputRequest struct {
	prompt   string
	onSubmit func(m *Model, value string)
}

// Model is the root Bubble Tea model: it owns the store connection, the
// three list views, and the small amount of modal state (task detail,
// text-input prompts) shared across them.
type Model struct {
	store  *core.Store
	screen screen

	taskList  list.Model
	readyList list.Model
	habitList list.Model

	showDetail bool
	detailTask *core.Task

	showBlockerPicker bool
	blockerTarget     *core.Task
	blockerList       list.Model

	input        textinput.Model
	pendingInput *inputRequest

	err error

	width, height int
}

func NewModel(store *core.Store) *Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.CharLimit = 200
	ti.PromptStyle = ti.PromptStyle.Foreground(colorDarkRed).Bold(true)
	ti.TextStyle = ti.TextStyle.Foreground(colorWhite)
	ti.Cursor.Style = ti.Cursor.Style.Foreground(colorDarkRed)

	m := &Model{
		store:     store,
		taskList:  newStyledList("All Tasks"),
		readyList: newStyledList("Ready to Work On"),
		habitList: newStyledList("Habits"),
		input:     ti,
	}
	m.blockerList = newStyledList("")

	m.refreshTasks()
	m.refreshHabits()
	return m
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		listHeight := msg.Height - 6
		if listHeight < 3 {
			listHeight = 3
		}
		for _, l := range []*list.Model{&m.taskList, &m.readyList, &m.habitList} {
			l.SetSize(msg.Width, listHeight)
		}
		return m, nil

	case tea.KeyMsg:
		if m.pendingInput != nil {
			return m.updatePendingInput(msg)
		}
		if m.showBlockerPicker {
			return m.updateBlockerPicker(msg)
		}
		if m.showDetail {
			return m.updateDetail(msg)
		}
		return m.updateBrowsing(msg)
	}
	return m, nil
}

func (m *Model) updatePendingInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pendingInput = nil
		m.input.Blur()
		m.input.SetValue("")
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.input.Value())
		req := m.pendingInput
		m.pendingInput = nil
		m.input.Blur()
		m.input.SetValue("")
		if value != "" {
			req.onSubmit(m, value)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.showDetail = false
		m.detailTask = nil
		return m, nil
	case "d":
		if m.detailTask.Done {
			m.detailTask.MarkUndone()
		} else {
			m.detailTask.MarkDone()
		}
		if err := m.store.SaveTask(ctx(), m.detailTask); err != nil {
			m.err = err
		}
		m.refreshTasks()
		return m, nil
	case "b":
		m.openBlockerPicker()
		return m, nil
	}
	return m, nil
}

// updateBlockerPicker routes keys to the blocker-search list. Typing
// filters candidate tasks by title (bubbles/list's fuzzy filter); Enter
// while not actively filtering confirms the highlighted task as the new
// blocker; Esc closes the picker once any active filter has been cleared.
func (m *Model) updateBlockerPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.blockerList.FilterState() == list.Unfiltered {
			m.showBlockerPicker = false
			m.blockerTarget = nil
			return m, nil
		}
	case "enter":
		if m.blockerList.FilterState() != list.Filtering {
			if item, ok := m.blockerList.SelectedItem().(taskItem); ok && m.blockerTarget != nil {
				if err := m.store.AddDependency(ctx(), m.blockerTarget.ID, item.t.ID); err != nil {
					m.err = fmt.Errorf("can't block %q by %q: it would create a dependency cycle", m.blockerTarget.Title, item.t.Title)
				} else {
					m.err = nil
					m.refreshTasks()
					m.showBlockerPicker = false
					m.blockerTarget = nil
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.blockerList, cmd = m.blockerList.Update(msg)
	return m, cmd
}

// openBlockerPicker opens a searchable list of every other task, so the
// user can find a blocker by name instead of typing its raw ID.
func (m *Model) openBlockerPicker() {
	if m.detailTask == nil {
		return
	}
	target := m.detailTask

	tasks, err := m.store.ListTasks(ctx())
	if err != nil {
		m.err = err
		return
	}
	var candidates []*core.Task
	for _, t := range tasks {
		if t.ID != target.ID {
			candidates = append(candidates, t)
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CreatedAt.Before(candidates[j].CreatedAt) })

	m.blockerTarget = target
	m.blockerList = newStyledList(fmt.Sprintf("Block %q by… (type to search)", target.Title))
	m.blockerList.SetItems(toTaskItems(candidates))

	listHeight := m.height - 6
	if listHeight < 3 {
		listHeight = 3
	}
	m.blockerList.SetSize(m.width, listHeight)
	m.blockerList.SetFilterState(list.Filtering)

	m.showBlockerPicker = true
}

func (m *Model) updateBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.screen = (m.screen + 1) % 3
		return m, nil
	case "shift+tab":
		m.screen = (m.screen + 2) % 3
		return m, nil
	case "a":
		switch m.screen {
		case screenTasks, screenReady:
			m.promptAddTask()
		case screenHabits:
			m.promptAddHabit()
		}
		return m, nil
	case "enter":
		switch m.screen {
		case screenTasks, screenReady:
			if t := m.selectedTask(); t != nil {
				m.showDetail = true
				m.detailTask = t
			}
		case screenHabits:
			m.checkInSelectedHabit()
		}
		return m, nil
	case "c":
		if m.screen == screenHabits {
			m.checkInSelectedHabit()
		}
		return m, nil
	case "d":
		if m.screen == screenTasks || m.screen == screenReady {
			m.toggleSelectedTaskDone()
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenTasks:
		m.taskList, cmd = m.taskList.Update(msg)
	case screenReady:
		m.readyList, cmd = m.readyList.Update(msg)
	case screenHabits:
		m.habitList, cmd = m.habitList.Update(msg)
	}
	return m, cmd
}

func (m *Model) View() string {
	var b strings.Builder

	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("error: "+m.err.Error()) + "\n\n")
	}

	switch {
	case m.pendingInput != nil:
		b.WriteString(promptStyle.Render(m.pendingInput.prompt) + "\n")
		b.WriteString(m.input.View() + "\n")
	case m.showBlockerPicker:
		b.WriteString(m.blockerList.View())
	case m.showDetail:
		b.WriteString(m.renderTaskDetail())
	default:
		switch m.screen {
		case screenTasks:
			b.WriteString(m.taskList.View())
		case screenReady:
			b.WriteString(m.readyList.View())
		case screenHabits:
			b.WriteString(m.habitList.View())
		}
	}

	b.WriteString(helpStyle.Render(m.helpText()))
	return b.String()
}

func (m *Model) renderTabs() string {
	tabs := []string{screenTasks.String(), screenReady.String(), screenHabits.String()}
	rendered := make([]string, len(tabs))
	for i, t := range tabs {
		if screen(i) == m.screen {
			rendered[i] = tabActiveStyle.Render(t)
		} else {
			rendered[i] = tabInactiveStyle.Render(t)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m *Model) helpText() string {
	switch {
	case m.pendingInput != nil:
		return "enter: submit  esc: cancel"
	case m.showBlockerPicker:
		return "type to search by title  enter: confirm  esc: cancel"
	case m.showDetail:
		return "d: toggle done  b: add dependency (blocked by)  esc: back  q: quit"
	case m.screen == screenHabits:
		return "tab: switch view  a: add habit  enter/c: check in  q: quit"
	default:
		return "tab: switch view  a: add task  enter: detail  d: toggle done  q: quit"
	}
}
