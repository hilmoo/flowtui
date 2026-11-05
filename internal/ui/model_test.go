package ui_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/hilmoo/flowtui/internal/config"
	"github.com/hilmoo/flowtui/internal/discover"
	"github.com/hilmoo/flowtui/internal/ui"
)

// fakeFlow builds an item whose command appends two JSONL records to the
// history file in the working directory.
func fakeFlow(name, marker string) discover.Item {
	script := `mkdir -p out
printf '%s\n' '{"time":"2025-01-01T10:00:00Z","method":"POST","url":"http://h/api/` + marker + `Create","request":{"n":1},"status":201,"durationMs":10,"response":{"id":1}}' >> out/history.jsonl
printf '%s\n' '{"time":"2025-01-01T10:00:01Z","method":"GET","url":"http://h/api/` + marker + `Get","status":418,"durationMs":4,"response":{"teapot":true}}' >> out/history.jsonl
echo ran ` + name
	return discover.Item{Name: name, Path: "", Command: []string{"sh", "-c", script}}
}

func TestPickFilterRunAndResults(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
	root := t.TempDir()
	// Pre-existing history that must not show up in the run.
	if err := os.MkdirAll(filepath.Join(root, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := `{"method":"GET","url":"http://h/api/Stale","status":200}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "out", "history.jsonl"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	items := []discover.Item{fakeFlow("alpha", "Alpha"), fakeFlow("beta", "Beta"), fakeFlow("gamma", "Gamma")}
	cfg := config.Default()
	m := ui.New(cfg, root, items)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	// Filter down to "beta".
	tm.Type("bet")
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("1/3 flows"))
	}, teatest.WithDuration(5*time.Second))

	// Run it.
	tm.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("console output"))
	}, teatest.WithDuration(10*time.Second))

	// Quit with ctrl+c and inspect the final model.
	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second)).(ui.Model)
	if !ok {
		t.Fatal("final model has unexpected type")
	}
	if final.Screen() != "results" {
		t.Fatalf("screen = %s, want results", final.Screen())
	}
	view := ansi.Strip(final.View().Content)
	for _, want := range []string{"beta", "2 requests", "201 POS BetaCreate", "418 GET BetaGet", "console output"} {
		if !strings.Contains(view, want) {
			t.Errorf("results view lacks %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Stale") {
		t.Error("stale history line leaked into the run")
	}
	// The last request is selected on success: its status is shown in the detail pane.
	if !strings.Contains(view, "GET /api/BetaGet") || !strings.Contains(view, "418") || !strings.Contains(view, `"teapot": true`) {
		t.Errorf("last request not selected/shown:\n%s", view)
	}
}
