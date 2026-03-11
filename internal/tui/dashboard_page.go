package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/weibaohui/nanobot-go/internal/tui/client"
)

// DashboardPage shows system overview
type DashboardPage struct {
	client *client.Client
	width  int
	height int
	stats  DashboardStats
	loading bool
}

// DashboardStats holds statistics
type DashboardStats struct {
	UserCount    int
	AgentCount   int
	ChannelCount int
	SessionCount int
}

// NewDashboardPage creates a new dashboard page
func NewDashboardPage(client *client.Client) *DashboardPage {
	return &DashboardPage{
		client:  client,
		loading: true,
	}
}

// Init initializes the page
func (p *DashboardPage) Init() tea.Cmd {
	return p.loadStats()
}

func (p *DashboardPage) loadStats() tea.Cmd {
	return func() tea.Msg {
		users, _ := p.client.ListUsers()
		agents, _ := p.client.ListAgents(1) // Default admin user
		channels, _ := p.client.ListChannels(1)
		sessions, _ := p.client.ListSessions(1)

		return StatsLoadedMsg{
			Stats: DashboardStats{
				UserCount:    len(users),
				AgentCount:   len(agents),
				ChannelCount: len(channels),
				SessionCount: len(sessions),
			},
		}
	}
}

// StatsLoadedMsg is sent when stats are loaded
type StatsLoadedMsg struct {
	Stats DashboardStats
}

// Update handles messages
func (p *DashboardPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case RefreshMsg:
		p.loading = true
		return p, p.loadStats()

	case StatsLoadedMsg:
		p.stats = msg.Stats
		p.loading = false
	}
	return p, nil
}

// View renders the dashboard
func (p *DashboardPage) View() string {
	if p.loading {
		return "Loading..."
	}

	title := TitleStyle.Render("📊 Dashboard")

	// Create stat boxes
	userBox := p.renderStatBox("👤 Users", fmt.Sprintf("%d", p.stats.UserCount))
	agentBox := p.renderStatBox("🤖 Agents", fmt.Sprintf("%d", p.stats.AgentCount))
	channelBox := p.renderStatBox("📡 Channels", fmt.Sprintf("%d", p.stats.ChannelCount))
	// sessionBox := p.renderStatBox("💬 Sessions", fmt.Sprintf("%d", p.stats.SessionCount))

	// Layout stats in a grid
	statsRow1 := lipgloss.JoinHorizontal(lipgloss.Top, userBox, agentBox)
	statsRow2 := lipgloss.JoinHorizontal(lipgloss.Top, channelBox)
	statsGrid := lipgloss.JoinVertical(lipgloss.Top, statsRow1, statsRow2)

	// Welcome message
	welcome := lipgloss.NewStyle().
		Foreground(lipgloss.Color(TextColor)).
		MarginTop(2).
		Render("Welcome to nanobot TUI Management System!")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color(DimColor)).
		MarginTop(1).
		Render("Press 1-5 to switch pages, ? for help, q to quit")

	return lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		statsGrid,
		welcome,
		help,
	)
}

func (p *DashboardPage) renderStatBox(icon, value string) string {
	boxStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(SidebarBg)).
		Foreground(lipgloss.Color(TextColor)).
		Width(20).
		Height(6).
		Padding(1).
		Margin(1)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(PrimaryColor)).
		Bold(true)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		icon,
		"",
		valueStyle.Render(value),
	)

	return boxStyle.Render(content)
}
