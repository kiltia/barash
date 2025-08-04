package tui

import (
	"fmt"

	"orb/runner/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
)

type ConfigPreviewModel struct {
	BaseModel
}

func NewConfigPreviewModel(config *config.Config) ConfigPreviewModel {
	model := ConfigPreviewModel{
		BaseModel: BaseModel{
			Options: configPreviewOptions,
			Config:  config,
		},
	}
	model.title = "Configuration Preview"
	return model
}

func (m ConfigPreviewModel) Init() tea.Cmd {
	return m.BaseModel.Init()
}

func (m ConfigPreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.BaseModel.Update(msg)
	if cmd != nil {
		return model, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return NewMainMenuModel(m.Config), nil
		case "up":
			m.navigateUp()
		case "down":
			m.navigateDown()
		case "enter":
			return m.handleEnter()
		}
	}
	return m, nil
}

func (m ConfigPreviewModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // Back to main menu
		NewMainMenuModel(m.Config)
	case 1: // Run application
		return m, tea.Quit
	}
	return m, nil
}

func (m ConfigPreviewModel) View() string {
	var options []string
	for _, option := range configPreviewOptions {
		options = append(options, option.Name)
	}
	baseRender := m.BaseModel.View()
	return fmt.Sprintf("%s%s", baseRender, renderConfigPreview(m, options))
}
