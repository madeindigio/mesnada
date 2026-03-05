package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all global key bindings for the TUI.
// Implements help.KeyMap interface for the bubbles/help component.
type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Left           key.Binding
	Right          key.Binding
	Top            key.Binding
	Bottom         key.Binding
	FilterStatus   key.Binding
	FilterAll      key.Binding
	FilterRunning  key.Binding
	FilterComplete key.Binding
	FilterFailed   key.Binding
	Search         key.Binding
	Quit           key.Binding
	Help           key.Binding
	Refresh        key.Binding
	ToggleSidebar  key.Binding
	Pause          key.Binding
	Resume         key.Binding
	Cancel         key.Binding
	Retry          key.Binding
	Purge          key.Binding
	ToggleLogPanel key.Binding
	ConfirmYes     key.Binding
	ConfirmNo      key.Binding
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.FilterStatus, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom},
		{k.FilterStatus, k.FilterAll, k.FilterRunning, k.FilterComplete, k.FilterFailed},
		{k.Search, k.ToggleSidebar, k.ToggleLogPanel, k.Refresh, k.Help, k.Quit},
		{k.Pause, k.Resume, k.Cancel, k.Retry, k.Purge},
		{k.ConfirmYes, k.ConfirmNo},
	}
}

// DefaultKeyMap returns the default vim-like key bindings.
func DefaultKeyMap() *KeyMap {
	return &KeyMap{
		Up:             key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:           key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:           key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left/focus list")),
		Right:          key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right/focus sidebar")),
		Top:            key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
		Bottom:         key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
		FilterStatus:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filter next")),
		FilterAll:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "filter all")),
		FilterRunning:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "filter running")),
		FilterComplete: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "filter completed")),
		FilterFailed:   key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "filter failed")),
		Search:         key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Quit:           key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:           key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh:        key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "refresh")),
		ToggleSidebar:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "sidebar")),
		Pause:          key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pause")),
		Resume:         key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "resume")),
		Cancel:         key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "cancel")),
		Retry:          key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "retry")),
		Purge:          key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "purge")),
		ToggleLogPanel: key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "logs")),
		ConfirmYes:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
		ConfirmNo:      key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "dismiss")),
	}
}
