package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/tui/client"
)

// ListPage shows the session list
type SessionListPage struct {
	client   *client.Client
	width    int
	height   int
	table    table.Model
	sessions []models.Session
	loading  bool
	err      error
}

// NewListPage creates a new session list page
func NewSessionListPage(client *client.Client) *SessionListPage {
	columns := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Session Key", Width: 25},
		{Title: "Channel", Width: 10},
		{Title: "Agent", Width: 10},
		{Title: "Last Active", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(10),
	)

	return &SessionListPage{
		client:  client,
		table:   t,
		loading: true,
	}
}

// Init initializes the page
func (p *SessionListPage) Init() tea.Cmd {
	return p.loadSessions()
}

func (p *SessionListPage) loadSessions() tea.Cmd {
	return func() tea.Msg {
		// Load sessions for user 0 (all)
		sessions, err := p.client.ListSessions(0)
		return SessionListLoadedMsg{Sessions: sessions, Error: err}
	}
}

// SessionListLoadedMsg is sent when sessions are loaded
type SessionListLoadedMsg struct {
	Sessions []models.Session
	Error    error
}

// Update handles messages
func (p *SessionListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.table.SetHeight(msg.Height - 8)

	case SessionListLoadedMsg:
		p.loading = false
		if msg.Error != nil {
			p.err = msg.Error
		} else {
			p.sessions = msg.Sessions
			p.updateTableRows()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			p.loading = true
			p.err = nil
			return p, p.loadSessions()
		}
	}

	newTable, cmd := p.table.Update(msg)
	p.table = newTable
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

func (p *SessionListPage) updateTableRows() {
	rows := make([]table.Row, len(p.sessions))
	for i, s := range p.sessions {
		agentID := "-"
		if s.AgentID != nil {
			agentID = fmt.Sprintf("%d", *s.AgentID)
		}
		lastActive := "-"
		if s.LastActiveAt != nil {
			lastActive = s.LastActiveAt.Format(time.RFC3339)[:19]
		}
		rows[i] = table.Row{
			fmt.Sprintf("%d", s.ID),
			s.SessionKey,
			fmt.Sprintf("%d", s.ChannelID),
			agentID,
			lastActive,
		}
	}
	p.table.SetRows(rows)
}

// View renders the page
func (p *SessionListPage) View() string {
	if p.loading {
		return "Loading sessions..."
	}

	if p.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347")).Render(
			fmt.Sprintf("Error: %v\n\nPress 'r' to retry", p.err),
		)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B68EE")).
		Render("💬 Sessions")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render(fmt.Sprintf("Total: %d | Enter: detail | d: delete | r: refresh", len(p.sessions)))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		"",
		p.table.View(),
		"",
		help,
	)
}
