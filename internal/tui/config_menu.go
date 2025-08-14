package tui

import (
	"os"
	"strings"

	"github.com/kiltia/runner/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
)

type ConfigMenuModel struct {
	BaseModel
}

func NewConfigMenuModel(cfg *config.Config) ConfigMenuModel {
	model := ConfigMenuModel{
		BaseModel: BaseModel{
			Options: configMenuOptions,
			Config:  cfg,
		},
	}
	model.title = "Config Menu"
	return model
}

func (m ConfigMenuModel) Init() tea.Cmd {
	return m.BaseModel.Init()
}

func (m ConfigMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		case "esc":
			return NewMainMenuModel(m.Config), nil
		case "down":
			m.navigateDown()
		case "up":
			m.navigateUp()
		}
	case FileSelectedMsg:
		return m.HandleFileSelected(msg)
	}
	return m, nil
}

func (m ConfigMenuModel) View() string {
	var options []string
	for _, item := range m.Options {
		options = append(options, item.Name)
	}
	return m.renderInner(func(s *strings.Builder) *strings.Builder {
		return renderMenu(s, m.cursor, options)
	})
}

func (m ConfigMenuModel) HandleFileSelected(
	msg FileSelectedMsg,
) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.err = msg.Error
		m.message = ""
		return m, nil
	}

	if msg.Path == "" {
		return m, nil
	}

	switch msg.Action {
	case ActionLoad:
		os.Unsetenv("CONFIG_FILE")
		m.Config = &config.Config{}
		if err := LoadConfig(m.Config, msg.Path); err != nil {
			m.err = err
		} else {
			m.err = nil
			m.message = "Config loaded"
		}
	case ActionSave:
		if err := SaveConfig(m.Config, msg.Path); err != nil {
			m.err = nil
			m.err = err
		} else {
			m.message = "Config saved"
		}
	}
	return m, nil
}

func (m ConfigMenuModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // Manual change
		return NewConfigEditorModel(m.Config, m.BaseModel), nil
	case 1: // Load file
		return SelectFile(m, ActionLoad)
	case 2: // Save file
		return SelectFile(m, ActionSave)
	}
	return m, nil
}
