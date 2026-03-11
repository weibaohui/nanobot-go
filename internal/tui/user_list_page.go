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

// ListPage shows the user list
type UserListPage struct {
	client  *client.Client
	width   int
	height  int
	table   table.Model
	users   []models.User
	loading bool
	err     error
}

// NewListPage creates a new user list page
func NewUserListPage(client *client.Client) *UserListPage {
	columns := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "Username", Width: 20},
		{Title: "API Key", Width: 25},
		{Title: "Created At", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithHeight(10),
	)

	return &UserListPage{
		client:  client,
		table:   t,
		loading: true,
	}
}

// Init initializes the page
func (p *UserListPage) Init() tea.Cmd {
	return p.loadUsers()
}

func (p *UserListPage) loadUsers() tea.Cmd {
	return func() tea.Msg {
		users, err := p.client.ListUsers()
		return UsersLoadedMsg{Users: users, Error: err}
	}
}

// UsersLoadedMsg is sent when users are loaded
type UsersLoadedMsg struct {
	Users []models.User
	Error error
}

// Update handles messages
func (p *UserListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.table.SetHeight(msg.Height - 8)

	case UsersLoadedMsg:
		p.loading = false
		if msg.Error != nil {
			p.err = msg.Error
		} else {
			p.users = msg.Users
			p.updateTableRows()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			p.loading = true
			p.err = nil
			return p, p.loadUsers()
		case "n":
			// TODO: Open create form
			return p, nil
		case "d":
			// TODO: Delete selected user
			return p, nil
		case "e":
			// TODO: Edit selected user
			return p, nil
		}
	}

	newTable, cmd := p.table.Update(msg)
	p.table = newTable
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

func (p *UserListPage) updateTableRows() {
	rows := make([]table.Row, len(p.users))
	for i, u := range p.users {
		rows[i] = table.Row{
			fmt.Sprintf("%d", u.ID),
			u.Username,
			"hidden",
			u.CreatedAt.Format(time.RFC3339)[:19],
		}
	}
	p.table.SetRows(rows)
}

// View renders the page
func (p *UserListPage) View() string {
	if p.loading {
		return "Loading users..."
	}

	if p.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347")).Render(
			fmt.Sprintf("Error: %v\n\nPress 'r' to retry", p.err),
		)
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B68EE")).
		Render("👤 Users")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render(fmt.Sprintf("Total: %d | n: new | e: edit | d: delete | r: refresh", len(p.users)))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		"",
		p.table.View(),
		"",
		help,
	)
}
