package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/kiltia/runner/internal"
	"github.com/kiltia/runner/internal/tui"
	"github.com/kiltia/runner/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.Config{}
	model := tui.NewMainMenuModel(&cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}

	// Check if we should run the application after exiting TUI
	if m, ok := finalModel.(tui.ConfigPreviewModel); ok {
		// Clear the screen and reset terminal
		fmt.Println("starting application")

		// Run the application in a way that doesn't block
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("application panicked: %v\n", r)
				fmt.Println("stack trace:")
				fmt.Println(string(debug.Stack()))
			}
		}()

		// Run the application
		internal.RunApplication(m.Config)
	}
}
