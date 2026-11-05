// Package jsonview pretty-prints JSON with syntax colors and wraps text by
// visible width.
package jsonview

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// MaxDepth bounds nesting so hostile input cannot exhaust the stack.
	MaxDepth = 500
	// DefaultMaxString is the length above which strings are truncated.
	DefaultMaxString = 200
	previewRunes     = 60
)

// Options control formatting.
type Options struct {
	// MaxString truncates strings longer than this many runes; 0 disables.
	MaxString int
	// Color enables ANSI colors.
	Color bool
}

var (
	keyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	strStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	numStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	litStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	moreStyle = lipgloss.NewStyle().Faint(true)
)

type printer struct {
	b   strings.Builder
	o   Options
	dec *json.Decoder
}

func (p *printer) paint(st lipgloss.Style, s string) string {
	if !p.o.Color {
		return s
	}
	return st.Render(s)
}

// Format pretty-prints one JSON document. It returns an error when data is
// not valid JSON, in which case callers should show the text as-is.
func Format(data []byte, o Options) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	p := &printer{o: o, dec: dec}
	if err := p.value(0); err != nil {
		return "", err
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", errors.New("trailing data after JSON value")
	}
	return p.b.String(), nil
}

func (p *printer) value(depth int) error {
	if depth > MaxDepth {
		return errors.New("json nested too deeply")
	}
	tok, err := p.dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case json.Delim:
		return p.container(t, depth)
	case string:
		p.b.WriteString(p.str(t))
	case json.Number:
		p.b.WriteString(p.paint(numStyle, t.String()))
	case bool:
		p.b.WriteString(p.paint(litStyle, fmt.Sprint(t)))
	case nil:
		p.b.WriteString(p.paint(litStyle, "null"))
	default:
		return fmt.Errorf("unexpected token %v", tok)
	}
	return nil
}

func (p *printer) container(open json.Delim, depth int) error {
	closer, isObj := "]", false
	if open == '{' {
		closer, isObj = "}", true
	}
	if !p.dec.More() {
		_, err := p.dec.Token()
		p.b.WriteString(string(open) + closer)
		return err
	}
	p.b.WriteString(string(open) + "\n")
	pad := strings.Repeat("  ", depth+1)
	for first := true; p.dec.More(); first = false {
		if !first {
			p.b.WriteString(",\n")
		}
		p.b.WriteString(pad)
		if isObj {
			kt, err := p.dec.Token()
			if err != nil {
				return err
			}
			k, _ := kt.(string)
			p.b.WriteString(p.paint(keyStyle, quote(k)) + ": ")
		}
		if err := p.value(depth + 1); err != nil {
			return err
		}
	}
	if _, err := p.dec.Token(); err != nil {
		return err
	}
	p.b.WriteString("\n" + strings.Repeat("  ", depth) + closer)
	return nil
}

func (p *printer) str(s string) string {
	n := runeCount(s)
	if p.o.MaxString > 0 && n > p.o.MaxString {
		prev := string([]rune(s)[:previewRunes])
		q := quote(prev)
		return p.paint(strStyle, q[:len(q)-1]) + p.paint(moreStyle, fmt.Sprintf("… (%d chars)", n)) + p.paint(strStyle, `"`)
	}
	return p.paint(strStyle, quote(s))
}

func runeCount(s string) int { return len([]rune(s)) }

func quote(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimRight(buf.String(), "\n")
}

// Body renders a recorded request/response value. Absent/null values give
// an empty string and ok=false. A JSON string is shown as plain text (or as
// pretty JSON when its content is itself JSON, as some servers return).
// Anything unparsable is shown verbatim.
func Body(raw []byte, o Options) (out string, ok bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			t := strings.TrimSpace(s)
			if len(t) > 0 && (t[0] == '{' || t[0] == '[') {
				if f, err := Format([]byte(t), o); err == nil {
					return f, true
				}
			}
			return truncText(s, o), true
		}
	}
	if f, err := Format(raw, o); err == nil {
		return f, true
	}
	return truncText(string(raw), o), true
}

func truncText(s string, o Options) string {
	n := runeCount(s)
	if o.MaxString > 0 && n > o.MaxString {
		return string([]rune(s)[:previewRunes]) + fmt.Sprintf("… (%d chars)", n)
	}
	return s
}
