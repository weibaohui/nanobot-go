package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/weibaohui/nanobot-go/internal/tui/client"
)

// PageType 页面类型
type PageType int

const (
	PageDashboard PageType = iota
	PageUsers
	PageAgents
	PageChannels
	PageSessions
)

// FocusArea 焦点区域
type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
	FocusDialog
)

// App TUI 应用主结构
type App struct {
	keys        KeyMap
	help        help.Model
	client      *client.Client
	width       int
	height      int
	currentPage PageType
	focus       FocusArea
	sidebar     *Sidebar
	content     tea.Model
	dialog      tea.Model
	showHelp    bool
	message     string
	messageType string
}

// NewApp 创建新的 TUI 应用
func NewApp(apiURL string) *App {
	keys := DefaultKeyMap()
	helpModel := help.New()
	helpModel.ShowAll = false

	app := &App{
		keys:        keys,
		help:        helpModel,
		client:      client.NewClient(apiURL),
		currentPage: PageDashboard,
		focus:       FocusSidebar,
		sidebar:     NewSidebar(),
	}

	app.setPage(PageDashboard)
	return app
}

// Init 初始化应用
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.sidebar.Init(),
		a.checkConnection(),
	)
}

func (a *App) checkConnection() tea.Cmd {
	return func() tea.Msg {
		if err := a.client.HealthCheck(); err != nil {
			return ConnectionMsg{Success: false, Error: err}
		}
		return ConnectionMsg{Success: true}
	}
}

// ConnectionMsg 连接状态消息
type ConnectionMsg struct {
	Success bool
	Error   error
}

func (a *App) setPage(page PageType) tea.Cmd {
	a.currentPage = page
	a.sidebar.SetSelected(int(page))

	var initCmd tea.Cmd
	switch page {
	case PageDashboard:
		a.content = NewDashboardPage(a.client)
		initCmd = a.content.Init()
	case PageUsers:
		a.content = NewUserListPage(a.client)
		initCmd = a.content.Init()
	case PageAgents:
		a.content = NewAgentListPage(a.client)
		initCmd = a.content.Init()
	case PageChannels:
		a.content = NewChannelListPage(a.client)
		initCmd = a.content.Init()
	case PageSessions:
		a.content = NewSessionListPage(a.client)
		initCmd = a.content.Init()
	}
	return initCmd
}

// Update 处理消息
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.help.Width = msg.Width

		contentWidth := msg.Width - 20
		contentHeight := msg.Height - 2

		a.sidebar.SetSize(20, contentHeight)
		if a.content != nil {
			newContent, cmd := a.content.Update(tea.WindowSizeMsg{
				Width:  contentWidth,
				Height: contentHeight,
			})
			a.content = newContent
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "?":
			a.showHelp = !a.showHelp
			return a, nil
		case "tab":
			if a.focus == FocusSidebar {
				a.focus = FocusContent
			} else {
				a.focus = FocusSidebar
			}
			return a, nil
		case "1":
			return a, a.setPage(PageDashboard)
		case "2":
			return a, a.setPage(PageUsers)
		case "3":
			return a, a.setPage(PageAgents)
		case "4":
			return a, a.setPage(PageChannels)
		case "5":
			return a, a.setPage(PageSessions)
		case "r":
			if a.content != nil {
				newContent, cmd := a.content.Update(RefreshMsg{})
				a.content = newContent
				cmds = append(cmds, cmd)
			}
			return a, nil
		}

	case ConnectionMsg:
		if !msg.Success {
			a.message = fmt.Sprintf("连接失败: %v", msg.Error)
			a.messageType = "error"
		} else {
			a.message = "已连接"
			a.messageType = "success"
		}
		return a, nil

	case ShowMessageMsg:
		a.message = msg.Message
		a.messageType = msg.Type
		return a, nil

	case PageChangeMsg:
		return a, a.setPage(msg.Page)
	}

	switch a.focus {
	case FocusSidebar:
		newSidebar, cmd := a.sidebar.Update(msg)
		a.sidebar = newSidebar.(*Sidebar)
		cmds = append(cmds, cmd)
		if page, ok := a.sidebar.GetSelectedPage(); ok {
			pageCmd := a.setPage(page)
			cmds = append(cmds, pageCmd)
		}
	case FocusContent:
		if a.content != nil {
			newContent, cmd := a.content.Update(msg)
			a.content = newContent
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

// View 渲染视图
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	header := a.renderHeader()

	var content string
	if a.showHelp {
		content = a.help.View(a.keys)
	} else {
		content = a.renderMainContent()
	}

	statusBar := a.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Top, header, content, statusBar)
}

func (a *App) renderHeader() string {
	title := "🐈 nanobot TUI"
	pageName := fmt.Sprintf("[%s]", a.getPageName())
	helpText := "[? Help]"

	titleWidth := lipgloss.Width(title)
	pageNameWidth := lipgloss.Width(pageName)
	helpWidth := lipgloss.Width(helpText)
	remainingWidth := a.width - titleWidth - pageNameWidth - helpWidth
	if remainingWidth < 0 {
		remainingWidth = 0
	}

	return HeaderStyle.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			title,
			lipgloss.NewStyle().Width(remainingWidth).Render(""),
			pageName,
			helpText,
		),
	)
}

func (a *App) renderMainContent() string {
	contentHeight := a.height - 2
	sidebarView := a.sidebar.View()

	var contentView string
	if a.content != nil {
		contentView = a.content.View()
	}

	contentView = ContentStyle.
		Width(a.width - 20).
		Height(contentHeight).
		Render(contentView)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView)
}

func (a *App) renderStatusBar() string {
	var statusMsg string
	switch a.messageType {
	case "success":
		statusMsg = SuccessMessageStyle.Render(a.message)
	case "error":
		statusMsg = ErrorMessageStyle.Render(a.message)
	case "warning":
		statusMsg = WarningMessageStyle.Render(a.message)
	default:
		statusMsg = a.message
	}

	return StatusBarStyle.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			StatusBarKeyStyle.Render("nanobot TUI"),
			" | ↑/↓: Navigate | q: Quit | ",
			statusMsg,
		),
	)
}

func (a *App) getPageName() string {
	switch a.currentPage {
	case PageDashboard:
		return "Dashboard"
	case PageUsers:
		return "Users"
	case PageAgents:
		return "Agents"
	case PageChannels:
		return "Channels"
	case PageSessions:
		return "Sessions"
	default:
		return "Unknown"
	}
}

// 消息类型
type RefreshMsg struct{}
type ShowDialogMsg struct{ Dialog tea.Model }
type CloseDialogMsg struct{}
type ShowMessageMsg struct{ Message string; Type string }
type PageChangeMsg struct{ Page PageType }
