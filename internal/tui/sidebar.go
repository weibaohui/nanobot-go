package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SidebarItem represents a menu item
type SidebarItem struct {
	Icon  string
	Label string
	Page  PageType
}

// Sidebar is the navigation sidebar
type Sidebar struct {
	items    []SidebarItem
	selected int
	width    int
	height   int
}

// NewSidebar creates a new sidebar
func NewSidebar() *Sidebar {
	return &Sidebar{
		items: []SidebarItem{
			{Icon: "🏠", Label: "Dashboard", Page: PageDashboard},
			{Icon: "👤", Label: "Users", Page: PageUsers},
			{Icon: "🤖", Label: "Agents", Page: PageAgents},
			{Icon: "📡", Label: "Channels", Page: PageChannels},
			{Icon: "💬", Label: "Sessions", Page: PageSessions},
		},
		selected: 0,
	}
}

// SetSize sets the sidebar dimensions
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetSelected sets the selected item
func (s *Sidebar) SetSelected(index int) {
	if index >= 0 && index < len(s.items) {
		s.selected = index
	}
}

// GetSelectedPage returns the currently selected page
func (s *Sidebar) GetSelectedPage() (PageType, bool) {
	if s.selected >= 0 && s.selected < len(s.items) {
		return s.items[s.selected].Page, true
	}
	return PageDashboard, false
}

// Init initializes the sidebar
func (s *Sidebar) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (s *Sidebar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.selected > 0 {
				s.selected--
				return s, func() tea.Msg {
					return PageChangeMsg{Page: s.items[s.selected].Page}
				}
			}
		case "down", "j":
			if s.selected < len(s.items)-1 {
				s.selected++
				return s, func() tea.Msg {
					return PageChangeMsg{Page: s.items[s.selected].Page}
				}
			}
		}
	}
	return s, nil
}

// View renders the sidebar
func (s *Sidebar) View() string {
	var items []string

	for i, item := range s.items {
		var style lipgloss.Style
		if i == s.selected {
			style = SidebarSelectedStyle
		} else {
			style = SidebarItemStyle
		}
		content := fmt.Sprintf(" %s %s", item.Icon, item.Label)
		items = append(items, style.Width(s.width).Render(content))
	}

	// Fill remaining space
	for i := len(items); i < s.height-2; i++ {
		items = append(items, "")
	}

	return SidebarStyle.Width(s.width).Height(s.height).Render(
		lipgloss.JoinVertical(lipgloss.Left, items...),
	)
}
