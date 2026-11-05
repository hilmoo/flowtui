package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func needSh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
}

func TestRunCollectsNewLinesOnly(t *testing.T) {
	needSh(t)
	dir := t.TempDir()
	hist := filepath.Join(dir, "h.jsonl")
	old := `{"method":"GET","url":"http://h/old","status":200}` + "\n"
	if err := os.WriteFile(hist, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	script := `echo '{"method":"GET","url":"http://h/new","status":201}' >> h.jsonl; echo '{bad' >> h.jsonl; printf '\033[31mred\033[0m out\n'; echo err >&2`
	res := Run(context.Background(), Spec{Name: "t", Command: []string{"sh", "-c", script}, Dir: dir, HistoryFile: "h.jsonl"})
	if res.ExitCode != 0 || res.StartErr != nil {
		t.Fatalf("res = %+v", res)
	}
	if len(res.Requests) != 1 || res.Requests[0].URL != "http://h/new" {
		t.Errorf("requests = %+v", res.Requests)
	}
	if len(res.Warnings) != 1 {
		t.Errorf("warnings = %v", res.Warnings)
	}
	if res.Log != "red out\nerr" {
		t.Errorf("log = %q (ANSI must be stripped, stderr merged)", res.Log)
	}
}

func TestRunExitCodeAndMissingHistory(t *testing.T) {
	needSh(t)
	res := Run(context.Background(), Spec{Command: []string{"sh", "-c", "echo bye; exit 7"}, Dir: t.TempDir(), HistoryFile: "none.jsonl"})
	if res.ExitCode != 7 || len(res.Requests) != 0 || len(res.Warnings) != 0 {
		t.Errorf("res = %+v", res)
	}
}

func TestRunStartFailure(t *testing.T) {
	res := Run(context.Background(), Spec{Command: []string{"definitely-not-a-command-xyz"}, Dir: t.TempDir(), HistoryFile: "h"})
	if res.StartErr == nil || res.ExitCode != -1 || res.Log == "" {
		t.Errorf("res = %+v", res)
	}
	if got := Run(context.Background(), Spec{}); got.StartErr == nil {
		t.Error("empty command must fail")
	}
}

func TestRunCancelKillsTree(t *testing.T) {
	needSh(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Result, 1)
	go func() {
		done <- Run(ctx, Spec{Command: []string{"sh", "-c", "sleep 30 & sleep 30; echo unreachable"}, Dir: t.TempDir(), HistoryFile: "h"})
	}()
	time.Sleep(500 * time.Millisecond)
	cancel()
	select {
	case res := <-done:
		if !res.Cancelled {
			t.Errorf("res = %+v", res)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
