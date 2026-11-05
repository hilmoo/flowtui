package ui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up, Down, Left, Right, Tab         key.Binding
	PageUp, PageDown, HalfUp, HalfDown key.Binding
	Top, Bottom, Full, Rerun, Copy     key.Binding
	Export, Back, Run, Cancel          key.Binding
	Type                               key.Binding
}

var keys = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "list")),
	Right:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "detail")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab/←→", "pane")),
	PageUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
	PageDown: key.NewBinding(key.WithKeys("pgdown", "space"), key.WithHelp("pgdn/space", "page")),
	HalfUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("ctrl+u", "half page up")),
	HalfDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "half page down")),
	Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g/G", "top/bottom")),
	Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
	Full:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "full")),
	Rerun:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "re-run")),
	Copy:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
	Export:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
	Back:     key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
	Run:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
	Cancel:   key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel run")),
	Type:     key.NewBinding(key.WithKeys(), key.WithHelp("type", "filter")),
}

// pickerKeys, runningKeys and resultsKeys implement help.KeyMap.
type pickerKeys struct{}

func (pickerKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		keys.Type,
		key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "select")),
		keys.Run,
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}
func (p pickerKeys) FullHelp() [][]key.Binding { return [][]key.Binding{p.ShortHelp()} }

type runningKeys struct{}

func (runningKeys) ShortHelp() []key.Binding    { return []key.Binding{keys.Cancel} }
func (r runningKeys) FullHelp() [][]key.Binding { return [][]key.Binding{r.ShortHelp()} }

type resultsKeys struct{}

func (resultsKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up"), key.WithHelp("↑↓/jk", "move")),
		keys.Tab, keys.PageDown, keys.Top, keys.Full, keys.Rerun, keys.Copy, keys.Export, keys.Back,
	}
}
func (r resultsKeys) FullHelp() [][]key.Binding { return [][]key.Binding{r.ShortHelp()} }
