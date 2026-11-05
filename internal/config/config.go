// Package config loads flowtui.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the config file looked up in the working directory.
const FileName = "flowtui.json"

// ExtraAction is a command listed above the flows that runs without a file.
type ExtraAction struct {
	Label   string   `json:"label"`
	Command []string `json:"command"`
}

// Config is the content of flowtui.json.
type Config struct {
	FlowsDir     string        `json:"flowsDir"`
	Glob         string        `json:"glob"`
	Command      []string      `json:"command"`
	HistoryFile  string        `json:"historyFile"`
	ExtraActions []ExtraAction `json:"extraActions"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{
		FlowsDir:    "flows",
		Glob:        "**/*.flow.ts",
		Command:     []string{"pnpm", "exec", "tsx", "{file}"},
		HistoryFile: "out/history.jsonl",
		ExtraActions: []ExtraAction{
			{Label: "[regenerate client]", Command: []string{"pnpm", "exec", "orval"}},
		},
	}
}

// Load reads dir/flowtui.json. When the file is missing it is created with
// the default content and the defaults are returned. created reports that.
func Load(dir string) (cfg Config, created bool, err error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg = Default()
		out, merr := json.MarshalIndent(cfg, "", "  ")
		if merr != nil {
			return cfg, false, merr
		}
		if werr := os.WriteFile(path, append(out, '\n'), 0o644); werr != nil {
			return cfg, false, fmt.Errorf("create %s: %w", path, werr)
		}
		return cfg, true, nil
	}
	if err != nil {
		return cfg, false, err
	}
	cfg = Default()
	// Fields absent from the file keep their default, except extraActions
	// which is replaced wholesale when present.
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, false, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, false, cfg.Validate()
}

// Validate checks required fields.
func (c Config) Validate() error {
	if c.FlowsDir == "" {
		return errors.New("flowsDir must not be empty")
	}
	if c.Glob == "" {
		return errors.New("glob must not be empty")
	}
	if len(c.Command) == 0 {
		return errors.New("command must not be empty")
	}
	if c.HistoryFile == "" {
		return errors.New("historyFile must not be empty")
	}
	for i, a := range c.ExtraActions {
		if len(a.Command) == 0 {
			return fmt.Errorf("extraActions[%d]: command must not be empty", i)
		}
	}
	return nil
}
