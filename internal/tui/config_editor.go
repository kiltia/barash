package tui

import (
	"strings"

	"github.com/kiltia/barash/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
)

type ConfigEditorModel struct {
	BaseModel
	Path []string
}

func NewConfigEditorModel(
	config *config.Config,
	old BaseModel,
) ConfigEditorModel {
	model := ConfigEditorModel{
		BaseModel: BaseModel{
			Options:   BuildNavigationForStruct(*config),
			Config:    config,
			oldCursor: old.cursor,
		},
		Path: []string{},
	}
	model.title = "Config Editor"
	return model
}

func (m ConfigEditorModel) FromFieldEditor(
	model FieldEditorModel,
) ConfigEditorModel {
	var options []ConfigItem
	if len(model.Path) == 0 {
		// Корневой уровень - пересобираем из Config
		options = BuildNavigationForStruct(*model.Config)
	} else {
		// Внутри структуры - получаем значение по пути и пересобираем
		value := GetValueByPath(model.Config, model.Path)
		options = BuildNavigationForStruct(value)
	}
	newModel := ConfigEditorModel{
		Path:      model.Path,
		BaseModel: model.BaseModel,
	}
	newModel.cursor = model.oldCursor
	newModel.Options = options
	newModel.title = "Config Editor"
	return newModel
}

func (m ConfigEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m.goBack(), nil
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

func (m ConfigEditorModel) View() string {
	return m.renderInner(func(s *strings.Builder) *strings.Builder {
		return renderConfigEdit(s, m)
	})
}

func (m ConfigEditorModel) goBack() tea.Model {
	if len(m.Path) > 0 {
		m.Path = m.Path[:len(m.Path)-1]
		if len(m.Path) == 0 {
			m.Options = BuildNavigationForStruct(*m.Config)
		} else {
			value := GetValueByPath(m.Config, m.Path)
			m.Options = BuildNavigationForStruct(value)
		}
	} else {
		return NewConfigMenuModel(m.Config)
	}

	m.cursor = m.oldCursor
	m.err = nil
	m.message = ""
	return m
}

func (m ConfigEditorModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.Options) {
		return m, nil
	}

	item := &m.Options[m.cursor]
	if item.IsStruct {
		m.Path = append(m.Path, item.Name)
		m.Options = BuildNavigationForStruct(item.Value)
		m.oldCursor = m.cursor
		m.cursor = 0
	} else {
		return NewFieldEditorModel(m.Config, m.Path, m.Options, m.cursor), nil
	}
	return m, nil
}
