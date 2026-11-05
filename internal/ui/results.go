package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/hilmoo/flowtui/internal/runner"
)

type pane int

const (
	listPane pane = iota
	detailPane
)

type eventKind int

const (
	evNone eventKind = iota
	evBack
	evRerun
	evCopy
	evExport
)

// resultsEvent asks the root model to do something with side effects.
type resultsEvent struct {
	kind eventKind
	text string // clipboard text for evCopy
}

// results shows the requests of one finished run. It is a value type.
type results struct {
	res    runner.Result
	sel    int
	offset int
	focus  pane
	full   bool
	vp     viewport.Model
	w, h   int // body area (excluding header/footer)
}

func newResults(res runner.Result, w, h int) results {
	r := results{res: res}
	// Land on the last request on success, on the console output on failure.
	r.sel = len(res.Requests)
	if res.ExitCode == 0 && len(res.Requests) > 0 {
		r.sel = len(res.Requests) - 1
	}
	r.vp = viewport.New()
	r.resize(w, h)
	if w < wideMin {
		r.focus = detailPane
	}
	return r
}

func (r results) rows() int { return len(r.res.Requests) + 1 }

func (r results) wide() bool { return r.w >= wideMin }

func (r results) detailWidth() int {
	if r.wide() {
		return atLeast1(r.w - leftWidth(r.w) - 1)
	}
	return atLeast1(r.w)
}

// resize applies a new body size and rebuilds the wrapped content.
func (r *results) resize(w, h int) {
	r.w, r.h = atLeast1(w), atLeast1(h)
	r.vp.SetWidth(r.detailWidth())
	r.vp.SetHeight(r.h)
	y := r.vp.YOffset()
	r.rebuild()
	r.vp.SetYOffset(y)
}

func (r *results) rebuild() {
	r.vp.SetContent(DetailContent(r.res, r.sel, r.full, r.detailWidth()))
}

func (r *results) selectRow(i int) {
	if i < 0 {
		i = 0
	}
	if i >= r.rows() {
		i = r.rows() - 1
	}
	if i == r.sel {
		return
	}
	r.sel = i
	r.rebuild()
	r.vp.GotoTop()
}

// lineRange returns the 1-based visible line range and the total.
func (r results) lineRange() (from, to, total int) {
	total = r.vp.TotalLineCount()
	if total == 0 {
		return 0, 0, 0
	}
	from = r.vp.YOffset() + 1
	to = from + r.vp.VisibleLineCount() - 1
	if to > total {
		to = total
	}
	return from, to, total
}

func (r results) update(msg tea.Msg) (results, resultsEvent) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			r.vp.ScrollUp(3)
		case tea.MouseWheelDown:
			r.vp.ScrollDown(3)
		}
	case tea.KeyPressMsg:
		return r.updateKey(msg)
	}
	return r, resultsEvent{}
}

func (r results) updateKey(msg tea.KeyPressMsg) (results, resultsEvent) {
	switch {
	case key.Matches(msg, keys.Back):
		return r, resultsEvent{kind: evBack}
	case key.Matches(msg, keys.Rerun):
		return r, resultsEvent{kind: evRerun}
	case key.Matches(msg, keys.Copy):
		return r, resultsEvent{kind: evCopy, text: r.copyText()}
	case key.Matches(msg, keys.Export):
		return r, resultsEvent{kind: evExport}
	case key.Matches(msg, keys.Full):
		r.full = !r.full
		y := r.vp.YOffset()
		r.rebuild()
		r.vp.SetYOffset(y)
	case key.Matches(msg, keys.Tab):
		r.focus = 1 - r.focus
	case key.Matches(msg, keys.Left):
		r.focus = listPane
	case key.Matches(msg, keys.Right):
		r.focus = detailPane
	case key.Matches(msg, keys.Up):
		if r.focus == listPane {
			r.selectRow(r.sel - 1)
		} else {
			r.vp.ScrollUp(1)
		}
	case key.Matches(msg, keys.Down):
		if r.focus == listPane {
			r.selectRow(r.sel + 1)
		} else {
			r.vp.ScrollDown(1)
		}
	case key.Matches(msg, keys.PageDown):
		r.vp.PageDown()
	case key.Matches(msg, keys.PageUp):
		r.vp.PageUp()
	case key.Matches(msg, keys.HalfDown):
		r.vp.HalfPageDown()
	case key.Matches(msg, keys.HalfUp):
		r.vp.HalfPageUp()
	case key.Matches(msg, keys.Top):
		r.vp.GotoTop()
	case key.Matches(msg, keys.Bottom):
		r.vp.GotoBottom()
	}
	return r, resultsEvent{}
}

// copyText is what `y` puts on the clipboard: the selected response as full
// pretty JSON, or the console log for the console row.
func (r results) copyText() string {
	if r.sel < len(r.res.Requests) {
		return plainJSON(r.res.Requests[r.sel].Response)
	}
	return r.res.Log
}

// exportValue is what `e` writes to out/last.json.
func (r results) exportValue() any {
	if r.sel < len(r.res.Requests) {
		return r.res.Requests[r.sel]
	}
	return struct {
		Flow       string `json:"flow"`
		ExitCode   int    `json:"exitCode"`
		DurationMs int64  `json:"durationMs"`
		Log        string `json:"log"`
	}{r.res.Name, r.res.ExitCode, r.res.Duration.Milliseconds(), r.res.Log}
}

func marshalExport(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func (r results) listLines(w, h int) []string {
	r.offset = listWindow(r.offset, r.sel, h, r.rows())
	var lines []string
	for row := 0; row < h && r.offset+row < r.rows(); row++ {
		i := r.offset + row
		var label string
		if i < len(r.res.Requests) {
			label = RowStyled(r.res.Requests[i], w-2)
		} else {
			label = "console output"
			if r.res.ExitCode != 0 {
				label = errStyle.Render(label)
			}
		}
		switch {
		case i == r.sel && r.focus == listPane:
			lines = append(lines, selStyle.Render(ansi.Truncate("▸ "+ansi.Strip(label), w, "…")))
		case i == r.sel:
			lines = append(lines, selDimStyle.Render("▸ ")+ansi.Truncate(label, w-2, "…"))
		default:
			lines = append(lines, "  "+ansi.Truncate(label, w-2, "…"))
		}
	}
	return lines
}

func (r results) view() string {
	detail := strings.Split(r.vp.View(), "\n")
	if !r.wide() {
		if r.focus == listPane {
			return strings.Join(fit(r.listLines(r.w, r.h), r.w, r.h), "\n")
		}
		return strings.Join(fit(detail, r.w, r.h), "\n")
	}
	lw := leftWidth(r.w)
	sep := dim.Render("│")
	if r.focus == detailPane {
		sep = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("│")
	}
	return joinPanes(fit(r.listLines(lw, r.h), lw, r.h), fit(detail, r.detailWidth(), r.h), sep)
}

func (r results) header() string {
	from, to, total := r.lineRange()
	s := fmt.Sprintf("%s · %d requests · lines %d-%d/%d", r.res.Name, len(r.res.Requests), from, to, total)
	if len(r.res.Warnings) > 0 {
		s += warnStyle.Render(fmt.Sprintf(" · ⚠ %d warning(s)", len(r.res.Warnings)))
	}
	return s
}
