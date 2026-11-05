package ui

import (
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/hilmoo/flowtui/internal/history"
)

var (
	green   = lipgloss.Color("2")
	red     = lipgloss.Color("1")
	yellow  = lipgloss.Color("3")
	magenta = lipgloss.Color("5")
	gray    = lipgloss.Color("8")
)

// StatusColor maps an HTTP status to its display color.
func StatusColor(status int) lipgloss.Style {
	st := lipgloss.NewStyle().Bold(true)
	switch {
	case status >= 200 && status < 300:
		return st.Foreground(green)
	case status >= 400 && status < 500:
		return st.Foreground(red)
	case status >= 500:
		return st.Foreground(magenta)
	case status == 0:
		return st.Foreground(gray)
	default:
		return st.Foreground(yellow)
	}
}

// StatusText is the status code, or --- when none was recorded.
func StatusText(status int) string {
	if status == 0 {
		return "---"
	}
	return strconv.Itoa(status)
}

// MethodAbbrev returns the first three characters of the method, upper-cased.
func MethodAbbrev(method string) string {
	m := strings.ToUpper(method)
	if utf8.RuneCountInString(m) > 3 {
		m = string([]rune(m)[:3])
	}
	return m
}

// LastSegment returns the last non-empty path segment of a URL
// ("http://h/api/ProjectCreate?x=1" -> "ProjectCreate"). It falls back to
// the host, then to the raw string.
func LastSegment(raw string) string {
	u, err := url.Parse(raw)
	if err == nil {
		segs := strings.Split(strings.Trim(u.Path, "/"), "/")
		if last := segs[len(segs)-1]; last != "" {
			return last
		}
		if u.Host != "" {
			return u.Host
		}
	}
	return raw
}

// PathOf returns the path (plus query) of a URL, for the detail title.
func PathOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Path == "" && u.Host == "") {
		return raw
	}
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	return p
}

// RowLabel builds the plain (uncolored) text of a request row.
func RowLabel(r history.Record) string {
	return StatusText(r.Status) + " " + MethodAbbrev(r.Method) + " " + LastSegment(r.URL)
}

// RowStyled renders a request row with a colored status, fitted to width.
func RowStyled(r history.Record, width int) string {
	s := StatusColor(r.Status).Render(StatusText(r.Status)) + " " + MethodAbbrev(r.Method) + " " + LastSegment(r.URL)
	return ansi.Truncate(s, width, "…")
}
