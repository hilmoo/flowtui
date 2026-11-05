package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/hilmoo/flowtui/internal/discover"
)

const previewMaxBytes = 64 * 1024

// picker is the flow list with a source preview. It is a value type.
type picker struct {
	items    []discover.Item
	filter   string
	filtered []int // indexes into items
	cursor   int   // index into filtered
	offset   int
	root     string

	previewFor string // item key the cached preview belongs to
	preview    []string
}

func newPicker(items []discover.Item, root string) picker {
	p := picker{items: items, root: root}
	p.refilter()
	return p
}

// Filter returns the indexes of items whose name contains q (case-insensitive).
func Filter(names []string, q string) []int {
	q = strings.ToLower(q)
	var out []int
	for i, n := range names {
		if q == "" || strings.Contains(strings.ToLower(n), q) {
			out = append(out, i)
		}
	}
	return out
}

func (p *picker) refilter() {
	names := make([]string, len(p.items))
	for i, it := range p.items {
		names[i] = it.Name
	}
	p.filtered = Filter(names, p.filter)
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p picker) selected() (discover.Item, bool) {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return discover.Item{}, false
	}
	return p.items[p.filtered[p.cursor]], true
}

func (p picker) move(d int) picker {
	if len(p.filtered) == 0 {
		return p
	}
	p.cursor += d
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	return p
}

func (p picker) typeText(s string) picker {
	p.filter += s
	p.cursor = 0
	p.refilter()
	return p
}

func (p picker) backspace() picker {
	if p.filter == "" {
		return p
	}
	r := []rune(p.filter)
	p.filter = string(r[:len(r)-1])
	p.cursor = 0
	p.refilter()
	return p
}

// loadPreview caches the source (or command line) of the selected item.
func (p picker) loadPreview() picker {
	it, ok := p.selected()
	if !ok {
		p.previewFor, p.preview = "", nil
		return p
	}
	key := it.Name + "|" + it.Path
	if key == p.previewFor {
		return p
	}
	p.previewFor = key
	if it.IsExtra() {
		p.preview = []string{"extra action (no file)", "", "$ " + strings.Join(it.Command, " ")}
		return p
	}
	data, err := os.ReadFile(filepath.Join(p.root, it.Path))
	if err != nil {
		p.preview = []string{"cannot read " + it.Path + ": " + err.Error()}
		return p
	}
	truncated := false
	if len(data) > previewMaxBytes {
		data, truncated = data[:previewMaxBytes], true
	}
	text := strings.ReplaceAll(string(data), "\r", "")
	text = strings.ReplaceAll(text, "\t", "    ")
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	width := len(fmt.Sprint(len(lines)))
	num := lipgloss.NewStyle().Foreground(gray)
	p.preview = make([]string, 0, len(lines)+1)
	for i, l := range lines {
		p.preview = append(p.preview, num.Render(fmt.Sprintf("%*d ", width, i+1))+l)
	}
	if truncated {
		p.preview = append(p.preview, num.Render("… file truncated"))
	}
	return p
}

// view renders the picker body (without header/footer) in a w×h area.
func (p picker) view(w, h int, running string) string {
	h = atLeast1(h)
	listH := h - 1 // first row shows the filter
	if listH < 1 {
		listH = 1
	}
	p.offset = listWindow(p.offset, p.cursor, listH, len(p.filtered))

	wide := w >= wideMin
	lw := w
	if wide {
		lw = leftWidth(w)
	}

	left := []string{ansi.Truncate(promptStyle.Render("filter› ")+p.filter+lipgloss.NewStyle().Reverse(true).Render(" "), lw, "")}
	if len(p.filtered) == 0 {
		left = append(left, dim.Render("no matching flows"))
	}
	for row := 0; row < listH && p.offset+row < len(p.filtered); row++ {
		idx := p.offset + row
		name := p.items[p.filtered[idx]].Name
		if idx == p.cursor {
			left = append(left, selStyle.Render(ansi.Truncate("▸ "+name, lw, "…")))
		} else {
			left = append(left, ansi.Truncate("  "+name, lw, "…"))
		}
	}
	if !wide {
		l := fit(left, lw, h)
		if running != "" {
			l[len(l)-1] = fit([]string{running}, lw, 1)[0]
		}
		return strings.Join(l, "\n")
	}

	rw := atLeast1(w - lw - 1)
	var right []string
	if running != "" {
		right = []string{running}
	} else {
		right = p.preview
	}
	sep := dim.Render("│")
	return joinPanes(fit(left, lw, h), fit(right, rw, h), sep)
}
