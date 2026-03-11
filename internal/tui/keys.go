package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the application
type KeyMap struct {
	// Global keys
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Tab     key.Binding
	BackTab key.Binding

	// Navigation keys
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Home  key.Binding
	End   key.Binding

	// Action keys
	Enter  key.Binding
	Escape key.Binding
	Save   key.Binding
	Edit   key.Binding
	Delete key.Binding
	New    key.Binding
	Search key.Binding

	// Page switch keys
	PageDashboard key.Binding
	PageUsers     key.Binding
	PageAgents    key.Binding
	PageChannels  key.Binding
	PageSessions  key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "f5"),
			key.WithHelp("r", "refresh"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
		BackTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		PageDashboard: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "dashboard"),
		),
		PageUsers: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "users"),
		),
		PageAgents: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "agents"),
		),
		PageChannels: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "channels"),
		),
		PageSessions: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "sessions"),
		),
	}
}

// ShortHelp returns the short help keybindings
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Enter, k.Quit, k.Help,
	}
}

// FullHelp returns the full help keybindings
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Home, k.End},
		{k.Enter, k.Escape, k.Tab, k.BackTab},
		{k.New, k.Edit, k.Delete, k.Save, k.Search},
		{k.Refresh, k.Help, k.Quit},
		{k.PageDashboard, k.PageUsers, k.PageAgents, k.PageChannels, k.PageSessions},
	}
}
