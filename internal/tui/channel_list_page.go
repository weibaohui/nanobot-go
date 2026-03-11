package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/tui/client"
)

// ListPage shows the channel list
type ChannelListPage struct {
	client   *client.Client
	width    int
	height   int
	table    table.Model
	channels []models.Channel
	loading  bool
	err      error
}

// NewListPage creates a new channel list page
func NewChannelListPage(client *client.Client) *ChannelListPage {
	columns := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 20},
		{Title: "Type", Width: 12},
		{Title: "User", Width: 8},
		{Title: "Agent", Width: 8},
		{Title: "Active", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(10),
	)

	return &ChannelListPage{
		client:  client,
		table:   t,
		loading: true,
	}
}

// Init initializes the page
func (p *ChannelListPage) Init() tea.Cmd {
	return p.loadChannels()
}

func (p *ChannelListPage) loadChannels() tea.Cmd {
	return func() tea.Msg {
		channels, err := p.client.ListChannels()
		return ChannelListLoadedMsg{Channels: channels, Error: err}
	}
}

// ChannelListLoadedMsg is sent when channels are loaded
type ChannelListLoadedMsg struct {
	Channels []models.Channel
	Error    error
}

// Update handles messages
func (p *ChannelListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.table.SetHeight(msg.Height - 8)

	case ChannelListLoadedMsg:
		p.loading = false
		if msg.Error != nil {
			p.err = msg.Error
		} else {
			p.channels = msg.Channels
			p.updateTableRows()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			p.loading = true
			p.err = nil
			return p, p.loadChannels()
		}
	}

	newTable, cmd := p.table.Update(msg)
	p.table = newTable
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

func (p *ChannelListPage) updateTableRows() {
	rows := make([]table.Row, len(p.channels))
	for i, c := range p.channels {
		agentID := "-"
		if c.AgentID != nil {
			agentID = fmt.Sprintf("%d", *c.AgentID)
		}
		isActive := "✗"
		if c.IsActive {
			isActive = "✓"
		}
		rows[i] = table.Row{
			fmt.Sprintf("%d", c.ID),
			c.Name,
			string(c.Type),
			fmt.Sprintf("%d", c.UserID),
			agentID,
			isActive,
		}
	}
	p.table.SetRows(rows)
}

// View renders the page
func (p *ChannelListPage) View() string {
	if p.loading {
		return "Loading channels..."
	}

	if p.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347")).Render(
			fmt.Sprintf("Error: %v\n\nPress 'r' to retry", p.err),
		)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B68EE")).
		Render("📡 Channels")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render(fmt.Sprintf("Total: %d | n: new | e: edit | d: delete | b: bind | r: refresh", len(p.channels)))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		"",
		p.table.View(),
		"",
		help,
	)
}
