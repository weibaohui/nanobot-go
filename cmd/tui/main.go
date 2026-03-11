package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/weibaohui/nanobot-go/internal/tui"
)

func main() {
	apiURL := os.Getenv("NANOBOT_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8081"
	}

	app := tui.NewApp(apiURL)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
