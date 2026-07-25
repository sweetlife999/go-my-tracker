package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/sweetlife999/go-my-tracker/core"
)

const (
	graphNodeW float32 = 130
	graphNodeH float32 = 36
	graphColW  float32 = 170
	graphRowH  float32 = 52
	graphPad   float32 = 16
)

type graphNode struct {
	rect *canvas.Rectangle
	text *canvas.Text
	x, y float32
}

// depGraphWidget renders the task dependency graph as boxes (positioned by
// dependency depth) connected by lines. This needs a custom WidgetRenderer
// since it requires freeform absolute positioning that no stock container
// layout expresses.
type depGraphWidget struct {
	widget.BaseWidget

	nodes []*graphNode
	edges []*canvas.Line
	size  fyne.Size
}

func newDepGraphWidget() *depGraphWidget {
	w := &depGraphWidget{size: fyne.NewSize(graphPad*2, graphPad*2)}
	w.ExtendBaseWidget(w)
	return w
}

// update rebuilds the node/edge layout from tasks. The level algorithm
// (level 0 = no blockers, level N+1 = 1+max(level of its blockers)) is
// presentation-only, mirrored from — but not part of — core/graph.go.
func (w *depGraphWidget) update(tasks []*core.Task) {
	level := computeLevels(tasks)

	byLevel := map[int][]*core.Task{}
	for _, t := range tasks {
		byLevel[level[t.ID]] = append(byLevel[level[t.ID]], t)
	}
	var levels []int
	for l := range byLevel {
		levels = append(levels, l)
	}
	sort.Ints(levels)

	pos := make(map[core.TaskID]fyne.Position)
	var nodes []*graphNode
	maxRows := 1
	for _, l := range levels {
		rowTasks := byLevel[l]
		if len(rowTasks) > maxRows {
			maxRows = len(rowTasks)
		}
		for i, t := range rowTasks {
			x := graphPad + float32(l)*graphColW
			y := graphPad + float32(i)*graphRowH
			pos[t.ID] = fyne.NewPos(x, y)

			fill := colorBlack
			stroke := colorDarkRed
			if t.Done {
				fill = colorFaint
				stroke = colorFaint
			}
			rect := canvas.NewRectangle(fill)
			rect.StrokeColor = stroke
			rect.StrokeWidth = 1.5
			rect.CornerRadius = 8

			label := t.Title
			if len(label) > 16 {
				label = label[:15] + "…"
			}
			text := canvas.NewText(label, colorWhite)
			text.TextSize = 12

			nodes = append(nodes, &graphNode{rect: rect, text: text, x: x, y: y})
		}
	}

	var edges []*canvas.Line
	for _, t := range tasks {
		to, ok := pos[t.ID]
		if !ok {
			continue
		}
		for _, bid := range t.BlockedBy {
			from, ok := pos[bid]
			if !ok {
				continue
			}
			line := canvas.NewLine(colorFaint)
			line.StrokeWidth = 1.5
			line.Position1 = fyne.NewPos(from.X+graphNodeW, from.Y+graphNodeH/2)
			line.Position2 = fyne.NewPos(to.X, to.Y+graphNodeH/2)
			edges = append(edges, line)
		}
	}

	w.nodes = nodes
	w.edges = edges

	width := graphPad*2 + graphNodeW
	if len(levels) > 0 {
		width = graphPad*2 + float32(len(levels))*graphColW
	}
	height := graphPad*2 + float32(maxRows)*graphRowH
	w.size = fyne.NewSize(width, height)

	w.Refresh()
}

func (w *depGraphWidget) CreateRenderer() fyne.WidgetRenderer {
	return &depGraphRenderer{widget: w}
}

type depGraphRenderer struct {
	widget *depGraphWidget
}

func (r *depGraphRenderer) Destroy() {}

func (r *depGraphRenderer) Layout(_ fyne.Size) {
	for _, n := range r.widget.nodes {
		n.rect.Move(fyne.NewPos(n.x, n.y))
		n.rect.Resize(fyne.NewSize(graphNodeW, graphNodeH))
		n.text.Move(fyne.NewPos(n.x+8, n.y+8))
		n.text.Resize(fyne.NewSize(graphNodeW-16, graphNodeH-16))
	}
}

func (r *depGraphRenderer) MinSize() fyne.Size {
	return r.widget.size
}

func (r *depGraphRenderer) Objects() []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, 0, len(r.widget.edges)+len(r.widget.nodes)*2)
	for _, e := range r.widget.edges {
		objs = append(objs, e)
	}
	for _, n := range r.widget.nodes {
		objs = append(objs, n.rect, n.text)
	}
	return objs
}

func (r *depGraphRenderer) Refresh() {
	r.Layout(r.widget.size)
	canvas.Refresh(r.widget)
}

// computeLevels assigns each task a dependency-depth level. Pure
// presentation logic — core has no notion of "depth", only IsReady/DAG.
func computeLevels(tasks []*core.Task) map[core.TaskID]int {
	byID := make(map[core.TaskID]*core.Task, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
	}
	level := make(map[core.TaskID]int, len(tasks))

	var resolve func(id core.TaskID, seen map[core.TaskID]bool) int
	resolve = func(id core.TaskID, seen map[core.TaskID]bool) int {
		if l, ok := level[id]; ok {
			return l
		}
		if seen[id] {
			return 0 // defensive cycle guard; core already rejects real cycles
		}
		seen[id] = true
		t, ok := byID[id]
		if !ok || len(t.BlockedBy) == 0 {
			level[id] = 0
			return 0
		}
		max := 0
		for _, b := range t.BlockedBy {
			if l := resolve(b, seen); l > max {
				max = l
			}
		}
		level[id] = max + 1
		return level[id]
	}

	for _, t := range tasks {
		resolve(t.ID, map[core.TaskID]bool{})
	}
	return level
}

// newCalendarGrid builds a month calendar highlighting today and marking
// days with a due (not-done) task with a small accent dot.
func newCalendarGrid(tasks []*core.Task) (fyne.CanvasObject, string) {
	now := time.Now()
	year, month, today := now.Date()
	loc := now.Location()
	first := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	startOffset := int(first.Weekday())
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()

	dueDays := map[int]bool{}
	for _, t := range tasks {
		if t.DueAt == nil || t.Done {
			continue
		}
		d := *t.DueAt
		if d.Year() == year && d.Month() == month {
			dueDays[d.Day()] = true
		}
	}

	grid := container.NewGridWithColumns(7)
	for _, wd := range [...]string{"S", "M", "T", "W", "T", "F", "S"} {
		lbl := widget.NewLabel(wd)
		lbl.Alignment = fyne.TextAlignCenter
		lbl.Importance = widget.LowImportance
		grid.Add(lbl)
	}
	for i := 0; i < startOffset; i++ {
		grid.Add(widget.NewLabel(""))
	}
	for d := 1; d <= daysInMonth; d++ {
		dayLabel := widget.NewLabel(fmt.Sprintf("%d", d))
		dayLabel.Alignment = fyne.TextAlignCenter
		if d == today {
			dayLabel.Importance = widget.HighImportance
		}
		if dueDays[d] {
			dot := canvas.NewRectangle(colorDarkRed)
			dot.SetMinSize(fyne.NewSize(6, 6))
			grid.Add(container.NewVBox(dayLabel, container.NewCenter(dot)))
		} else {
			grid.Add(container.NewVBox(dayLabel))
		}
	}

	return grid, first.Format("January 2006")
}

// graphScreen is the "Graph" tab: the dependency diagram above a month
// calendar of due dates.
type graphScreen struct {
	store      *core.Store
	graph      *depGraphWidget
	calBox     *fyne.Container
	monthLabel *widget.Label
	root       fyne.CanvasObject
}

func newGraphScreen(store *core.Store) *graphScreen {
	s := &graphScreen{store: store}
	s.graph = newDepGraphWidget()
	s.monthLabel = widget.NewLabel("")
	s.monthLabel.Importance = widget.LowImportance
	s.calBox = container.NewVBox()

	heading := widget.NewLabel("Task graph")
	heading.TextStyle = fyne.TextStyle{Bold: true}
	sub := widget.NewLabel("What blocks what")
	sub.Importance = widget.LowImportance

	calHeading := widget.NewLabel("Calendar")
	calHeading.TextStyle = fyne.TextStyle{Bold: true}

	body := container.NewVBox(
		heading, sub,
		container.NewHScroll(s.graph),
		widget.NewSeparator(),
		calHeading, s.monthLabel,
		s.calBox,
	)
	s.root = container.NewVScroll(body)
	return s
}

func (s *graphScreen) Refresh() {
	tasks, err := s.store.ListTasks(context.Background())
	if err != nil {
		tasks = nil
	}
	s.graph.update(tasks)

	cal, month := newCalendarGrid(tasks)
	s.monthLabel.SetText(month)
	s.calBox.RemoveAll()
	s.calBox.Add(cal)
}
