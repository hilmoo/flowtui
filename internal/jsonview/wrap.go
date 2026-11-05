package jsonview

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// StripANSI removes all escape sequences.
func StripANSI(s string) string { return ansi.Strip(s) }

// Wrap hard-wraps every line of s to the given visible width. It is aware of
// ANSI sequences and wide characters, and re-opens the active SGR style at
// the start of each continuation line so scrolling never loses colors.
// Carriage returns are dropped and tabs expanded to four spaces.
func Wrap(s string, width int) string {
	if width < 1 {
		width = 1
	}
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if ansi.StringWidth(line) <= width {
			out = append(out, line)
			continue
		}
		active := ""
		for _, part := range strings.Split(ansi.Hardwrap(line, width, true), "\n") {
			p := active + part
			active = trackSGR(active, part)
			if active != "" {
				p += "\x1b[0m"
			}
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n")
}

// trackSGR returns the SGR sequences still active after s, given the ones
// active before it.
func trackSGR(active, s string) string {
	var state byte
	for len(s) > 0 {
		seq, _, n, ns := ansi.DecodeSequence(s, state, nil)
		state = ns
		s = s[n:]
		if len(seq) > 2 && seq[0] == 0x1b && seq[1] == '[' && seq[len(seq)-1] == 'm' {
			params := seq[2 : len(seq)-1]
			if params == "" || params == "0" {
				active = ""
			} else {
				active += seq
			}
		}
	}
	return active
}
