package tui

import (
	"fmt"
	"strings"

	"github.com/kiltia/runner/pkg/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type FieldEditorModel struct {
	BaseModel
	textInput  textinput.Model
	EditField  *ConfigItem
	Path       []string
	Navigation []ConfigItem
}

func NewFieldEditorModel(
	config *config.Config,
	path []string,
	navigation []ConfigItem,
) FieldEditorModel {
	ti := textinput.New()
	ti.Placeholder = "Enter new value"
	ti.Width = 50
	ti.Prompt = ">"
	ti.Focus() // ВАЖНО: устанавливаем фокус
	return FieldEditorModel{
		BaseModel: BaseModel{
			Options: navigation,
			Config:  config,
		},
		textInput:  ti,
		EditField:  &navigation[0],
		Path:       path,
		Navigation: navigation,
	}
}

func (m FieldEditorModel) Init() tea.Cmd {
	return m.BaseModel.Init()
}

func (m FieldEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.textInput, cmd = m.textInput.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m.goBack(), nil
		case "enter":
			return m.handleEnter()
		}
	}
	return m, cmd
}

func (m FieldEditorModel) View() string {
	var s strings.Builder

	// Header
	s.WriteString(Styles.Title.Render(" Field Editor ") + "\n\n")

	// Field name
	s.WriteString(
		fmt.Sprintf("Field: %s\n", Styles.Selected.Render(m.EditField.Name)),
	)

	// Current value
	s.WriteString(fmt.Sprintf("Current value: %s\n\n",
		Styles.Normal.Render(FormatValue(m.EditField.Value))))

	// Input field with label
	s.WriteString("New value:\n")
	s.WriteString(m.textInput.View() + "\n\n")

	// Instructions
	s.WriteString(Styles.Normal.Render("Press ") +
		Styles.Selected.Render("Enter") +
		Styles.Normal.Render(" to save, ") +
		Styles.Selected.Render("Esc") +
		Styles.Normal.Render(" to cancel"))

	return s.String()
}

func (m FieldEditorModel) goBack() tea.Model {
	m.cursor = 0
	m.textInput.Blur()

	var model ConfigEditorModel
	return model.FromFieldEditor(m)
}

func (m FieldEditorModel) handleEnter() (tea.Model, tea.Cmd) {
	if err := UpdateField(m); err != nil {
		m.err = err
	} else {
		m.message = "Field updated"
		m.textInput.Blur()
	}
	var model ConfigEditorModel
	return model.FromFieldEditor(m), nil
}
