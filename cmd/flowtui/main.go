// Command flowtui lists runnable flows, runs one and shows its HTTP traffic.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/hilmoo/flowtui/internal/config"
	"github.com/hilmoo/flowtui/internal/discover"
	"github.com/hilmoo/flowtui/internal/ui"
)

// Set at release time via -ldflags "-X main.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v" || os.Args[1] == "version") {
		fmt.Printf("flowtui %s (commit %s, built %s)\n", version, commit, date)
		return
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "flowtui:", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, created, err := config.Load(root)
	if err != nil {
		return err
	}
	if created {
		fmt.Fprintf(os.Stderr, "flowtui: created default %s\n", config.FileName)
	}
	items, err := discover.Discover(cfg, root)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(ui.New(cfg, root, items)).Run()
	return err
}
