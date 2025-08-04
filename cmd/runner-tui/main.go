package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"orb/runner/internal"
	"orb/runner/internal/tui"
	"orb/runner/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.Config{}
	model := tui.NewMainMenuModel(&cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}

	// Check if we should run the application after exiting TUI
	if m, ok := finalModel.(tui.ConfigPreviewModel); ok {
		// Clear the screen and reset terminal
		fmt.Println("Starting application...")

		// Run the application in a way that doesn't block
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Application panicked: %v\n", r)
				fmt.Println("Stack trace:")
				fmt.Println(string(debug.Stack()))
			}
		}()

		// Run the application
		internal.RunApplication(m.Config)
	}
}
