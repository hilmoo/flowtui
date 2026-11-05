package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Mouse capture stops the terminal from selecting text natively, so flowtui
// implements selection itself: drag with the left button to highlight, and the
// text is copied to the clipboard (OSC 52) on release.

type point struct{ x, y int }

type selection struct {
	active   bool // button is held down
	visible  bool // a non-empty selection is highlighted
	from, to point
	// region is the area the drag started in; the selection never leaves it.
	x0, x1, y0, y1 int // inclusive bounds
}

// regionAt returns the inclusive bounds of the pane under p: the header and
// footer rows are their own regions, and in the two-pane layout each pane is
// separate (the separator column belongs to neither).
func (m Model) regionAt(p point) (x0, x1, y0, y1 int, ok bool) {
	last := m.h - 1
	switch {
	case p.y <= 0:
		return 0, m.w - 1, 0, 0, true
	case p.y >= last:
		return 0, m.w - 1, last, last, true
	}
	y0, y1 = 1, last-1
	if m.w < wideMin {
		return 0, m.w - 1, y0, y1, true
	}
	lw := leftWidth(m.w)
	switch {
	case p.x < lw:
		return 0, lw - 1, y0, y1, true
	case p.x == lw:
		return 0, 0, 0, 0, false
	}
	return lw + 1, m.w - 1, y0, y1, true
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (s selection) clampPoint(p point) point {
	return point{clamp(p.x, s.x0, s.x1), clamp(p.y, s.y0, s.y1)}
}

// ordered returns the endpoints in reading order.
func (s selection) ordered() (a, b point) {
	a, b = s.from, s.to
	if b.y < a.y || (b.y == a.y && b.x < a.x) {
		a, b = b, a
	}
	return a, b
}

// span returns the half-open cell range [l, r) selected on row y.
func (s selection) span(y int) (l, r int, ok bool) {
	a, b := s.ordered()
	if y < a.y || y > b.y {
		return 0, 0, false
	}
	l, r = s.x0, s.x1+1
	if y == a.y {
		l = a.x
	}
	if y == b.y {
		r = b.x + 1
	}
	return l, r, l < r
}

// highlight renders rendered with the selection in reverse video.
func (s selection) highlight(rendered string) string {
	if !s.visible {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	for y, line := range lines {
		l, r, ok := s.span(y)
		if !ok {
			continue
		}
		lines[y] = ansi.Cut(line, 0, l) +
			"\x1b[7m" + ansi.Strip(ansi.Cut(line, l, r)) + "\x1b[27m" +
			ansi.Cut(line, r, ansi.StringWidth(line))
	}
	return strings.Join(lines, "\n")
}

// text extracts the selected plain text from rendered.
func (s selection) text(rendered string) string {
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines))
	for y, line := range lines {
		l, r, ok := s.span(y)
		if !ok {
			continue
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(line, l, r)), " "))
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// handleMouse processes click/drag/release. handled reports whether msg was a
// selection event.
func (m Model) handleMouse(msg tea.Msg) (_ Model, cmd tea.Cmd, handled bool) {
	switch e := msg.(type) {
	case tea.MouseClickMsg:
		if e.Button != tea.MouseLeft {
			return m, nil, true
		}
		p := point{e.X, e.Y}
		x0, x1, y0, y1, ok := m.regionAt(p)
		if !ok {
			m.sel = selection{}
			return m, nil, true
		}
		m.sel = selection{active: true, from: p, to: p, x0: x0, x1: x1, y0: y0, y1: y1}
		return m, nil, true
	case tea.MouseMotionMsg:
		if !m.sel.active {
			return m, nil, true
		}
		m.sel.to = m.sel.clampPoint(point{e.X, e.Y})
		m.sel.visible = m.sel.to != m.sel.from
		return m, nil, true
	case tea.MouseReleaseMsg:
		if !m.sel.active {
			return m, nil, true
		}
		m.sel.active = false
		m.sel.to = m.sel.clampPoint(point{e.X, e.Y})
		m.sel.visible = m.sel.to != m.sel.from
		if !m.sel.visible {
			return m, nil, true
		}
		txt := m.sel.text(m.content())
		if txt == "" {
			return m, nil, true
		}
		m.status = "copied selection to clipboard"
		return m, tea.SetClipboard(txt), true
	}
	return m, nil, false
}
