package tui

import (
	"strings"

	"github.com/kiltia/barash/config"

	tea "github.com/charmbracelet/bubbletea"
)

type MainMenuModel struct {
	BaseModel
}

func NewMainMenuModel(config *config.Config) MainMenuModel {
	return MainMenuModel{
		BaseModel: BaseModel{
			Options: mainModelOptions,
			title:   "Main Menu",
			Config:  config,
		},
	}
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		case "down":
			m.navigateDown()
		case "up":
			m.navigateUp()
		}
	}
	return m, nil
}

func (m MainMenuModel) View() string {
	var options []string
	for _, item := range m.Options {
		options = append(options, item.Name)
	}
	rendered := m.renderInner(func(s *strings.Builder) *strings.Builder {
		return renderMenu(s, m.cursor, options)
	})
	return rendered
}

func (m MainMenuModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // Edit config
		return NewConfigMenuModel(m.Config), nil
	case 1: // Preview and Run
		return NewConfigPreviewModel(m.Config), nil
	}
	return NewConfigMenuModel(m.Config), nil
}
