// Package ui contains the Bubble Tea model of flowtui.
package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/hilmoo/flowtui/internal/config"
	"github.com/hilmoo/flowtui/internal/discover"
	"github.com/hilmoo/flowtui/internal/runner"
)

type screen int

const (
	screenPicker screen = iota
	screenResults
)

// runDoneMsg is delivered when a run finishes.
type runDoneMsg struct {
	id  int
	res runner.Result
}

// runState describes the run in flight.
type runState struct {
	id         int
	item       discover.Item
	cancel     context.CancelFunc
	cancelling bool
}

// Model is the root Bubble Tea model. It is a value type; sub-models
// (picker, results, spinner, help) are values too.
type Model struct {
	cfg     config.Config
	root    string
	w, h    int
	screen  screen
	picker  picker
	results results
	lastRun discover.Item
	running *runState
	nextID  int
	spin    spinner.Model
	help    help.Model
	status  string
}

// New builds the model. root is the directory flows and files are relative to.
func New(cfg config.Config, root string, items []discover.Item) Model {
	return Model{
		cfg:    cfg,
		root:   root,
		w:      100,
		h:      30,
		picker: newPicker(items, root).loadPreview(),
		spin:   spinner.New(spinner.WithSpinner(spinner.Dot)),
		help:   help.New(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

func (m Model) bodyHeight() int { return atLeast1(m.h - 2) }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = atLeast1(msg.Width), atLeast1(msg.Height)
		m.help.SetWidth(m.w)
		if m.screen == screenResults {
			m.results.resize(m.w, m.bodyHeight())
		}
		return m, nil

	case spinner.TickMsg:
		if m.running == nil {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case runDoneMsg:
		return m.finishRun(msg)
	}

	if m.running != nil {
		return m.updateRunning(msg)
	}
	switch m.screen {
	case screenResults:
		return m.updateResults(msg)
	default:
		return m.updatePicker(msg)
	}
}

func (m Model) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || !key.Matches(k, keys.Cancel) {
		return m, nil // all other input is ignored while running
	}
	if m.running.cancelling {
		return m, tea.Quit // second ctrl+c: stop waiting and exit
	}
	rs := *m.running
	rs.cancelling = true
	m.running = &rs
	rs.cancel()
	return m, nil
}

func (m Model) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.picker = m.picker.move(-1).loadPreview()
		case tea.MouseWheelDown:
			m.picker = m.picker.move(1).loadPreview()
		}
	case tea.KeyPressMsg:
		m.status = ""
		switch {
		case msg.String() == "ctrl+c", msg.String() == "esc":
			return m, tea.Quit
		case key.Matches(msg, keys.Run):
			if it, ok := m.picker.selected(); ok {
				return m.startRun(it)
			}
		case msg.String() == "up":
			m.picker = m.picker.move(-1).loadPreview()
		case msg.String() == "down":
			m.picker = m.picker.move(1).loadPreview()
		case msg.String() == "backspace":
			m.picker = m.picker.backspace().loadPreview()
		case msg.Text != "":
			m.picker = m.picker.typeText(msg.Text).loadPreview()
		}
	}
	return m, nil
}

func (m Model) startRun(it discover.Item) (tea.Model, tea.Cmd) {
	m.nextID++
	ctx, cancel := context.WithCancel(context.Background())
	m.running = &runState{id: m.nextID, item: it, cancel: cancel}
	m.lastRun = it
	m.status = ""
	spec := runner.Spec{Name: it.Name, Command: it.Command, Dir: m.root, HistoryFile: m.cfg.HistoryFile}
	id := m.nextID
	run := func() tea.Msg { return runDoneMsg{id: id, res: runner.Run(ctx, spec)} }
	return m, tea.Batch(run, m.spin.Tick)
}

func (m Model) finishRun(msg runDoneMsg) (tea.Model, tea.Cmd) {
	if m.running == nil || msg.id != m.running.id {
		return m, nil
	}
	m.running.cancel() // release the context
	m.running = nil
	if msg.res.Cancelled {
		m.screen = screenPicker
		m.status = "cancelled " + msg.res.Name
		return m, nil
	}
	m.screen = screenResults
	m.results = newResults(msg.res, m.w, m.bodyHeight())
	return m, nil
}

func (m Model) updateResults(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		m.status = ""
		if k.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	var ev resultsEvent
	m.results, ev = m.results.update(msg)
	switch ev.kind {
	case evBack:
		m.screen = screenPicker
		m.picker = m.picker.loadPreview()
	case evRerun:
		return m.startRun(m.lastRun)
	case evCopy:
		if ev.text == "" {
			m.status = "nothing to copy"
			return m, nil
		}
		m.status = fmt.Sprintf("copied %d bytes to clipboard", len(ev.text))
		return m, tea.SetClipboard(ev.text)
	case evExport:
		m.status = m.export()
	}
	return m, nil
}

func (m Model) export() string {
	data, err := marshalExport(m.results.exportValue())
	if err != nil {
		return "export failed: " + err.Error()
	}
	path := filepath.Join(m.root, "out", "last.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "export failed: " + err.Error()
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "export failed: " + err.Error()
	}
	return "wrote out/last.json"
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var header, body string
	var km help.KeyMap
	switch {
	case m.running != nil:
		header = headerStyle.Render("flowtui")
		line := m.spin.View() + " running " + m.running.item.Name + " …"
		if m.running.cancelling {
			line = m.spin.View() + " cancelling " + m.running.item.Name + " … (ctrl+c again to quit)"
		}
		body = m.picker.view(m.w, m.bodyHeight(), line)
		km = runningKeys{}
	case m.screen == screenResults:
		header = headerStyle.Render(m.results.header())
		body = m.results.view()
		km = resultsKeys{}
	default:
		header = headerStyle.Render("flowtui") + dim.Render(fmt.Sprintf(" · %d/%d flows", len(m.picker.filtered), len(m.picker.items)))
		body = m.picker.view(m.w, m.bodyHeight(), "")
		km = pickerKeys{}
	}
	footer := m.help.ShortHelpView(km.ShortHelp())
	if m.status != "" {
		footer = warnStyle.Render(m.status)
	}
	lines := []string{
		ansi.Truncate(header, m.w, "…"),
		body,
		ansi.Truncate(footer, m.w, "…"),
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// Screen reports the active screen (used by tests).
func (m Model) Screen() string {
	if m.screen == screenResults {
		return "results"
	}
	return "picker"
}
