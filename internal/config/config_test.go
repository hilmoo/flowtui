package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, created, err := Load(dir)
	if err != nil || !created {
		t.Fatalf("Load = %v, %v", created, err)
	}
	if cfg.FlowsDir != "flows" || cfg.Glob != "**/*.flow.ts" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if _, err := os.Stat(filepath.Join(dir, FileName)); err != nil {
		t.Errorf("config file not written: %v", err)
	}
	again, created, err := Load(dir)
	if err != nil || created {
		t.Fatalf("second Load = %v, %v", created, err)
	}
	if again.HistoryFile != cfg.HistoryFile {
		t.Errorf("round trip mismatch")
	}
}

func TestLoadErrors(t *testing.T) {
	tests := map[string]string{
		"bad json":      `{`,
		"empty command": `{"command": []}`,
		"empty extra":   `{"extraActions": [{"label":"x","command":[]}]}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := Load(dir); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestLoadPartialKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"flowsDir":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FlowsDir != "x" || cfg.HistoryFile != "out/history.jsonl" {
		t.Errorf("got %+v", cfg)
	}
}
