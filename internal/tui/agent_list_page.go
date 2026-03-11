package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/weibaohui/nanobot-go/internal/models"
	"github.com/weibaohui/nanobot-go/internal/tui/client"
)

// ListPage shows the agent list
type AgentListPage struct {
	client  *client.Client
	width   int
	height  int
	table   table.Model
	agents  []models.Agent
	loading bool
	err     error
}

// NewListPage creates a new agent list page
func NewAgentListPage(client *client.Client) *AgentListPage {
	columns := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Name", Width: 20},
		{Title: "User", Width: 10},
		{Title: "Model", Width: 15},
		{Title: "Default", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(10),
	)

	return &AgentListPage{
		client:  client,
		table:   t,
		loading: true,
	}
}

// Init initializes the page
func (p *AgentListPage) Init() tea.Cmd {
	return p.loadAgents()
}

func (p *AgentListPage) loadAgents() tea.Cmd {
	return func() tea.Msg {
		agents, err := p.client.ListAgents(1) // Default admin user
		return AgentListLoadedMsg{Agents: agents, Error: err}
	}
}

// AgentListLoadedMsg is sent when agents are loaded
type AgentListLoadedMsg struct {
	Agents []models.Agent
	Error  error
}

// Update handles messages
func (p *AgentListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.table.SetHeight(msg.Height - 8)

	case AgentListLoadedMsg:
		p.loading = false
		if msg.Error != nil {
			p.err = msg.Error
		} else {
			p.agents = msg.Agents
			p.updateTableRows()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			p.loading = true
			p.err = nil
			return p, p.loadAgents()
		}
	}

	newTable, cmd := p.table.Update(msg)
	p.table = newTable
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

func (p *AgentListPage) updateTableRows() {
	rows := make([]table.Row, len(p.agents))
	for i, a := range p.agents {
		isDefault := ""
		if a.IsDefault {
			isDefault = "✓"
		}
		rows[i] = table.Row{
			fmt.Sprintf("%d", a.ID),
			a.Name,
			fmt.Sprintf("%d", a.UserID),
			a.Model,
			isDefault,
		}
	}
	p.table.SetRows(rows)
}

// View renders the page
func (p *AgentListPage) View() string {
	if p.loading {
		return "Loading agents..."
	}

	if p.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347")).Render(
			fmt.Sprintf("Error: %v\n\nPress 'r' to retry", p.err),
		)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B68EE")).
		Render("🤖 Agents")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render(fmt.Sprintf("Total: %d | n: new | e: edit | d: delete | c: config | r: refresh", len(p.agents)))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		"",
		p.table.View(),
		"",
		help,
	)
}
