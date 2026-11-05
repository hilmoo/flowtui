package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// wideMin is the width from which two panes are shown side by side.
const wideMin = 80

// leftWidth returns the list pane width for a terminal width: ~33%, clamped
// to [24, 44], and never more than w-1.
func leftWidth(w int) int {
	lw := w * 33 / 100
	if lw < 24 {
		lw = 24
	}
	if lw > 44 {
		lw = 44
	}
	if lw > w-1 {
		lw = w - 1
	}
	if lw < 1 {
		lw = 1
	}
	return lw
}

func atLeast1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// fit forces lines to exactly w columns and h rows (truncating or padding).
func fit(lines []string, w, h int) []string {
	w, h = atLeast1(w), atLeast1(h)
	out := make([]string, h)
	for i := 0; i < h; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		if ansi.StringWidth(l) > w {
			l = ansi.Truncate(l, w, "")
		}
		if pad := w - ansi.StringWidth(l); pad > 0 {
			l += strings.Repeat(" ", pad)
		}
		out[i] = l
	}
	return out
}

// joinPanes places left and right side by side with a separator column.
func joinPanes(left, right []string, sep string) string {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	rows := make([]string, n)
	for i := range rows {
		rows[i] = left[i] + sep + right[i]
	}
	return strings.Join(rows, "\n")
}

// listWindow returns the first visible index so that sel stays in view.
func listWindow(offset, sel, height, total int) int {
	if height < 1 {
		height = 1
	}
	if sel < offset {
		offset = sel
	}
	if sel >= offset+height {
		offset = sel - height + 1
	}
	if max := total - height; offset > max {
		offset = max
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
