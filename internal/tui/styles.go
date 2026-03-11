package tui

import "github.com/charmbracelet/lipgloss"

// Colors
const (
	PrimaryColor   = "#7B68EE" // 紫色
	SecondaryColor = "#00BFFF" // 蓝色
	SuccessColor   = "#32CD32" // 绿色
	WarningColor   = "#FFA500" // 橙色
	ErrorColor     = "#FF6347" // 红色
	TextColor      = "#E0E0E0" // 浅灰
	DimColor       = "#808080" // 深灰
	BgColor        = "#1E1E1E" // 背景
	SidebarBg      = "#252525" // 侧边栏背景
)

// Styles
var (
	// App styles
	AppStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(BgColor))

	// Header styles
	HeaderStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(PrimaryColor)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	HeaderInfoStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(PrimaryColor)).
		Foreground(lipgloss.Color("#E0E0E0")).
		Padding(0, 1)

	// Sidebar styles
	SidebarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(SidebarBg)).
		Width(20).
		Padding(1, 0)

	SidebarItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(TextColor)).
		Padding(0, 1)

	SidebarSelectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(PrimaryColor)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	SidebarShortcutStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(SecondaryColor)).
		Bold(true)

	// Content styles
	ContentStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(BgColor)).
		Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(PrimaryColor)).
		Bold(true).
		MarginBottom(1)

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color(TextColor)).
		Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(SecondaryColor)).
		Bold(true)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(PrimaryColor)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1)

	TableRowStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(TextColor)).
		Padding(0, 1)

	TableRowSelectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#3A3A3A")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	// Form styles
	FormLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(SecondaryColor)).
		Bold(true).
		MarginRight(1)

	FormInputStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#2A2A2A")).
		Foreground(lipgloss.Color(TextColor)).
		Padding(0, 1)

	FormFocusedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(PrimaryColor)).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	// Dialog styles
	DialogBoxStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#2A2A2A")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		Padding(2)

	DialogTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(PrimaryColor)).
		Bold(true).
		MarginBottom(1)

	// Message styles
	SuccessMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(SuccessColor))

	ErrorMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(ErrorColor))

	WarningMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(WarningColor))
)
