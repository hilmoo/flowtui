package jsonview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func maxWidth(s string) int {
	m := 0
	for _, l := range strings.Split(s, "\n") {
		if w := ansi.StringWidth(l); w > m {
			m = w
		}
	}
	return m
}

func TestWrapPlain(t *testing.T) {
	if got := Wrap("abcdefghij", 4); got != "abcd\nefgh\nij" {
		t.Errorf("got %q", got)
	}
	if got := Wrap("short\nlines", 20); got != "short\nlines" {
		t.Errorf("got %q", got)
	}
	if got := Wrap("a\r\nb\tc", 20); got != "a\nb    c" {
		t.Errorf("CR/tab handling: %q", got)
	}
	if got := Wrap("abc", 0); maxWidth(got) > 1 {
		t.Errorf("width 0 must clamp to 1: %q", got)
	}
}

func TestWrapUnicodeWidth(t *testing.T) {
	// 6 wide chars (2 cells each) wrapped at 5 cells.
	got := Wrap("日本語日本語", 5)
	if maxWidth(got) > 5 {
		t.Errorf("exceeds width: %q", got)
	}
	if strings.ReplaceAll(got, "\n", "") != "日本語日本語" {
		t.Errorf("lost characters: %q", got)
	}
	_ = Wrap("日本", 1) // must terminate
}

func TestWrapANSI(t *testing.T) {
	red := "\x1b[31m" + strings.Repeat("x", 10) + "\x1b[0m"
	lines := strings.Split(Wrap(red, 4), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %q", lines)
	}
	for i, l := range lines {
		if ansi.StringWidth(l) > 4 {
			t.Errorf("line %d too wide: %q", i, l)
		}
		if !strings.Contains(l, "\x1b[31m") {
			t.Errorf("line %d lost its color: %q", i, l)
		}
	}
	if got := ansi.Strip(strings.Join(lines, "\n")); got != "xxxx\nxxxx\nxx" {
		t.Errorf("stripped = %q", got)
	}
}

func TestStripANSI(t *testing.T) {
	if got := StripANSI("\x1b[1;32mok\x1b[0m \x1b[2K done"); got != "ok  done" {
		t.Errorf("got %q", got)
	}
}
