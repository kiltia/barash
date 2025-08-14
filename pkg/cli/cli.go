package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kiltia/barash/internal/tui"
	"github.com/kiltia/barash/pkg/config"
)

func Run() (*config.Config, error) {
	cfg := config.Config{}
	config.Load(&cfg)
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
		return m.Config, nil
	} else {
		return nil, fmt.Errorf("unexpected model type: %T", finalModel)
	}
}
