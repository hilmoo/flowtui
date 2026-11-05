package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hilmoo/flowtui/internal/history"
	"github.com/hilmoo/flowtui/internal/jsonview"
	"github.com/hilmoo/flowtui/internal/runner"
)

// FormatDuration renders a millisecond duration compactly.
func FormatDuration(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.2fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}

func formatTime(s string) string {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.Local().Format("15:04:05")
	}
	return s
}

func section(title string) string {
	return dim.Render("── " + title + " ──")
}

// requestDetail builds the unwrapped detail text of one request.
func requestDetail(r history.Record, full bool) string {
	o := jsonview.Options{Color: true}
	if !full {
		o.MaxString = jsonview.DefaultMaxString
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(strings.ToUpper(r.Method)+" "+PathOf(r.URL)) + "\n")
	b.WriteString(StatusColor(r.Status).Render(StatusText(r.Status)) + dim.Render(" · "+FormatDuration(r.DurationMs)+" · "+formatTime(r.Time)) + "\n")
	b.WriteString(dim.Render(r.URL) + "\n\n")
	b.WriteString(section("request") + "\n")
	if s, ok := jsonview.Body(r.Request, o); ok {
		b.WriteString(s + "\n")
	} else {
		b.WriteString(dim.Render("(none)") + "\n")
	}
	b.WriteString("\n" + section("response") + "\n")
	if s, ok := jsonview.Body(r.Response, o); ok {
		b.WriteString(s)
	} else {
		b.WriteString(dim.Render("(none)"))
	}
	return b.String()
}

// consoleDetail builds the detail text of the "console output" row.
func consoleDetail(res runner.Result) string {
	var b strings.Builder
	code := fmt.Sprintf("exit code %d", res.ExitCode)
	if res.ExitCode == 0 {
		b.WriteString(StatusColor(200).Render(code))
	} else {
		b.WriteString(errStyle.Render(code))
	}
	b.WriteString(dim.Render(" · " + res.Duration.Round(time.Millisecond).String()))
	if res.Cancelled {
		b.WriteString(warnStyle.Render(" · cancelled"))
	}
	b.WriteString("\n")
	for _, w := range res.Warnings {
		b.WriteString(warnStyle.Render("⚠ "+w) + "\n")
	}
	b.WriteString("\n")
	if strings.TrimSpace(res.Log) == "" {
		b.WriteString(dim.Render("(no output)"))
	} else {
		b.WriteString(res.Log)
	}
	return b.String()
}

// DetailContent returns the wrapped content for row sel (len(Requests) is the
// console row).
func DetailContent(res runner.Result, sel int, full bool, width int) string {
	var s string
	if sel >= 0 && sel < len(res.Requests) {
		s = requestDetail(res.Requests[sel], full)
	} else {
		s = consoleDetail(res)
	}
	return jsonview.Wrap(s, width)
}

// plainJSON returns the full, uncolored, indented value of a raw JSON field.
func plainJSON(raw json.RawMessage) string {
	s, ok := jsonview.Body(raw, jsonview.Options{})
	if !ok {
		return ""
	}
	return s
}
