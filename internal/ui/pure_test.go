package ui

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/hilmoo/flowtui/internal/config"
	"github.com/hilmoo/flowtui/internal/discover"
	"github.com/hilmoo/flowtui/internal/history"
	"github.com/hilmoo/flowtui/internal/runner"
)

func TestLastSegment(t *testing.T) {
	tests := []struct{ in, want string }{
		{"http://h/api/ProjectCreate", "ProjectCreate"},
		{"http://h/api/ProjectCreate/", "ProjectCreate"},
		{"http://h/api/x?y=1#f", "x"},
		{"http://host:8080", "host:8080"},
		{"/relative/path", "path"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := LastSegment(tt.in); got != tt.want {
			t.Errorf("LastSegment(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRowLabel(t *testing.T) {
	r := history.Record{Method: "post", URL: "http://h/api/ProjectCreate", Status: 201}
	if got := RowLabel(r); got != "201 POS ProjectCreate" {
		t.Errorf("got %q", got)
	}
	if got := RowLabel(history.Record{Method: "GET", URL: "http://h/a"}); got != "--- GET a" {
		t.Errorf("no status: %q", got)
	}
	if got := ansi.StringWidth(RowStyled(r, 8)); got > 8 {
		t.Errorf("RowStyled exceeds width: %d", got)
	}
}

func TestStatusColor(t *testing.T) {
	render := func(s int) string { return StatusColor(s).Render("x") }
	want := map[int]string{200: "32m", 404: "31m", 500: "35m", 302: "33m", 0: "90m"}
	for status, code := range want {
		if !strings.Contains(render(status), code) {
			t.Errorf("status %d: %q lacks %s", status, render(status), code)
		}
	}
}

func TestPathOf(t *testing.T) {
	if got := PathOf("http://h/api/x?a=1"); got != "/api/x?a=1" {
		t.Errorf("got %q", got)
	}
	if got := PathOf("http://h"); got != "/" {
		t.Errorf("got %q", got)
	}
}

func TestLeftWidth(t *testing.T) {
	tests := []struct{ w, want int }{{80, 26}, {100, 33}, {200, 44}, {60, 24}, {10, 9}, {1, 1}}
	for _, tt := range tests {
		if got := leftWidth(tt.w); got != tt.want {
			t.Errorf("leftWidth(%d) = %d, want %d", tt.w, got, tt.want)
		}
	}
}

func TestListWindow(t *testing.T) {
	tests := []struct{ off, sel, h, total, want int }{
		{0, 0, 5, 20, 0},
		{0, 7, 5, 20, 3},
		{10, 2, 5, 20, 2},
		{18, 19, 5, 20, 15},
		{5, 0, 0, 3, 0},
		{0, 0, 5, 2, 0},
	}
	for _, tt := range tests {
		if got := listWindow(tt.off, tt.sel, tt.h, tt.total); got != tt.want {
			t.Errorf("listWindow(%+v) = %d", tt, got)
		}
	}
}

func TestFit(t *testing.T) {
	got := fit([]string{"abcdef", "日本"}, 4, 3)
	if len(got) != 3 {
		t.Fatalf("rows = %d", len(got))
	}
	for i, l := range got {
		if ansi.StringWidth(l) != 4 {
			t.Errorf("row %d width %d: %q", i, ansi.StringWidth(l), l)
		}
	}
	if len(fit(nil, 0, 0)) != 1 {
		t.Error("height must clamp to 1")
	}
}

func TestFilter(t *testing.T) {
	names := []string{"[regen]", "auth/Login", "auth/logout", "projects/create"}
	tests := []struct {
		q    string
		want []int
	}{
		{"", []int{0, 1, 2, 3}},
		{"LOG", []int{1, 2}},
		{"auth/lo", []int{1, 2}},
		{"[", []int{0}},
		{"zzz", nil},
	}
	for _, tt := range tests {
		if got := Filter(names, tt.q); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Filter(%q) = %v, want %v", tt.q, got, tt.want)
		}
	}
}

func sampleResult() runner.Result {
	long := strings.Repeat("B", 600)
	return runner.Result{
		Name: "demo",
		Requests: []history.Record{
			{Time: "2025-01-01T10:00:00Z", Method: "POST", URL: "http://h/api/ProjectCreate", Request: json.RawMessage(`{"name":"x"}`), Status: 201, DurationMs: 12, Response: json.RawMessage(`{"id":1,"shot":"` + long + `"}`)},
			{Time: "2025-01-01T10:00:01Z", Method: "GET", URL: "http://h/api/Missing", Status: 404, DurationMs: 3, Response: json.RawMessage(`"Not Found"`)},
		},
		Log:      "line one\nline two",
		Duration: 1500 * time.Millisecond,
	}
}

func TestDetailContent(t *testing.T) {
	res := sampleResult()
	plain := func(s string) string { return ansi.Strip(s) }

	got := plain(DetailContent(res, 0, false, 60))
	for _, want := range []string{"POST /api/ProjectCreate", "201 · 12ms", "── request ──", "── response ──", `"name": "x"`, "(600 chars)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, strings.Repeat("B", 300)) {
		t.Error("long string must be truncated")
	}
	full := plain(DetailContent(res, 0, true, 60))
	if strings.Count(full, "B") != 600 {
		t.Errorf("full mode must show all 600 chars, got %d", strings.Count(full, "B"))
	}
	for _, l := range strings.Split(DetailContent(res, 0, true, 40), "\n") {
		if ansi.StringWidth(l) > 40 {
			t.Fatalf("line wider than 40: %q", l)
		}
	}

	notFound := plain(DetailContent(res, 1, false, 60))
	if !strings.Contains(notFound, "Not Found") || !strings.Contains(notFound, "(none)") {
		t.Errorf("plain-text response / absent request: %s", notFound)
	}

	console := plain(DetailContent(res, 2, false, 60))
	if !strings.Contains(console, "exit code 0") || !strings.Contains(console, "line two") {
		t.Errorf("console: %s", console)
	}
}

func TestResultsInitialSelection(t *testing.T) {
	res := sampleResult()
	if r := newResults(res, 100, 20); r.sel != 1 {
		t.Errorf("success: sel = %d, want last request (1)", r.sel)
	}
	res.ExitCode = 2
	if r := newResults(res, 100, 20); r.sel != 2 {
		t.Errorf("failure: sel = %d, want console row (2)", r.sel)
	}
	res.ExitCode = 0
	res.Requests = nil
	if r := newResults(res, 100, 20); r.sel != 0 {
		t.Errorf("no requests: sel = %d", r.sel)
	}
}

func TestResultsResponsive(t *testing.T) {
	res := sampleResult()
	r := newResults(res, 100, 20)
	if !r.wide() || r.focus != listPane {
		t.Error("wide: two panes, list focused")
	}
	if got := len(strings.Split(r.view(), "\n")); got != 20 {
		t.Errorf("view rows = %d", got)
	}
	r.resize(60, 10)
	if r.wide() || r.detailWidth() != 60 {
		t.Errorf("narrow: detail must take the full width, got %d", r.detailWidth())
	}
	for _, l := range strings.Split(r.view(), "\n") {
		if ansi.StringWidth(l) > 60 {
			t.Fatalf("line wider than terminal: %q", l)
		}
	}
	// tiny sizes must not panic and clamp to >= 1 row
	r.resize(1, 0)
	_ = r.view()
	r.resize(0, 0)
	_ = r.view()
	_ = r.header()
}

func TestResultsHeaderLines(t *testing.T) {
	res := sampleResult()
	res.Log = strings.Repeat("row\n", 100)
	r := newResults(res, 100, 10)
	r.selectRow(2)
	if h := r.header(); !strings.Contains(h, "lines 1-10/") || !strings.Contains(h, "2 requests") {
		t.Errorf("header = %q", h)
	}
	r.vp.PageDown()
	if h := r.header(); !strings.Contains(h, "lines 11-20/") {
		t.Errorf("after page down: %q", h)
	}
	r.selectRow(0) // selection change resets scroll
	if h := r.header(); !strings.Contains(h, "lines 1-") {
		t.Errorf("after reselect: %q", h)
	}
}

func TestResultsTogglesFullKeepsOffsetValid(t *testing.T) {
	r := newResults(sampleResult(), 100, 8)
	r.selectRow(0)
	r.full = true
	r.resize(100, 8)
	if from, _, total := r.lineRange(); from < 1 || total < from {
		t.Errorf("range %d/%d", from, total)
	}
}

func TestResultsNavigationFlow(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	script := `printf '%s\n' '{"method":"POST","url":"http://h/api/AlphaCreate","request":{"n":1},"status":201,"durationMs":10,"response":{"id":1}}' '{"method":"GET","url":"http://h/api/AlphaGet","status":418,"durationMs":4,"response":{"teapot":true}}' >> h.jsonl`
	item := discover.Item{Name: "alpha", Command: []string{"sh", "-c", script}}
	cfg := config.Default()
	cfg.HistoryFile = "h.jsonl"
	root := t.TempDir()

	var model tea.Model = New(cfg, root, []discover.Item{item})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	rs := model.(Model).running
	if rs == nil {
		t.Fatal("enter should start a run")
	}
	// While running, other keys are ignored.
	model, _ = model.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if model.(Model).running == nil || model.(Model).picker.filter != "" {
		t.Error("input must be ignored while running")
	}
	res := runner.Run(context.Background(), runner.Spec{Name: "alpha", Command: item.Command, Dir: root, HistoryFile: "h.jsonl"})
	model, _ = model.Update(runDoneMsg{id: rs.id, res: res})
	view := func() string { return ansi.Strip(model.(Model).View().Content) }
	if model.(Model).Screen() != "results" || !strings.Contains(view(), "GET /api/AlphaGet") {
		t.Fatalf("expected results with last request selected:\n%s", view())
	}

	model, _ = model.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if !strings.Contains(view(), "POST /api/AlphaCreate") {
		t.Errorf("k should select the previous request:\n%s", view())
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if !strings.Contains(view(), "POST /api/AlphaCreate") {
		t.Errorf("j in the detail pane must scroll, not change the selection:\n%s", view())
	}
	// Narrow terminal: one pane (the focused detail) gets the whole width.
	model, _ = model.Update(tea.WindowSizeMsg{Width: 50, Height: 12})
	for _, l := range strings.Split(view(), "\n") {
		if ansi.StringWidth(l) > 50 {
			t.Fatalf("line wider than 50: %q", l)
		}
	}
	if strings.Contains(view(), "console output") {
		t.Error("narrow results must show one pane at a time")
	}
	// e writes out/last.json
	model, _ = model.Update(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if _, err := os.Stat(filepath.Join(root, "out", "last.json")); err != nil {
		t.Errorf("last.json not written: %v", err)
	}
	// y copies the response via OSC52.
	_, cmd := model.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Error("y should return a clipboard command")
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.(Model).Screen() != "picker" {
		t.Error("esc must return to the picker")
	}
}

func TestCtrlCCancelsRun(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	item := discover.Item{Name: "slow", Command: []string{"sh", "-c", "sleep 30"}}
	var model tea.Model = New(config.Default(), t.TempDir(), []discover.Item{item})
	model, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	rs := model.(Model).running
	done := make(chan tea.Msg, 1)
	go func() {
		// First batch element is the run command.
		if b, ok := cmd().(tea.BatchMsg); ok {
			done <- b[0]()
		}
	}()
	time.Sleep(300 * time.Millisecond)
	model, _ = model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	select {
	case msg := <-done:
		model, _ = model.Update(msg)
	case <-time.After(10 * time.Second):
		t.Fatal("run did not stop after ctrl+c")
	}
	m := model.(Model)
	if m.running != nil || m.Screen() != "picker" || !strings.Contains(m.status, "cancelled") {
		t.Errorf("running=%v screen=%s status=%q (rs=%v)", m.running, m.Screen(), m.status, rs != nil)
	}
}
