package tui

import (
	"strings"

	"github.com/kiltia/barash/pkg/config"

	tea "github.com/charmbracelet/bubbletea"
)

// BaseModel содержит общие поля и методы для всех моделей
type BaseModel struct {
	Config  *config.Config
	Options []ConfigItem

	cursor    int
	oldCursor int
	title     string
	message   string
	err       error
}

func (m BaseModel) Init() tea.Cmd {
	return nil
}

func (m BaseModel) View() string {
	return ""
}

func (m BaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *BaseModel) navigateUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *BaseModel) navigateDown() {
	if m.cursor < m.getMaxCursor() {
		m.cursor++
	}
}

func (m BaseModel) renderInner(
	f func(*strings.Builder) *strings.Builder,
) string {
	var s strings.Builder
	if m.title != "" {
		s.WriteString(Styles.Title.Render(m.title) + "\n\n")
	}

	if m.message != "" {
		s.WriteString(Styles.Success.Render(m.message) + "\n\n")
	}

	if m.err != nil {
		s.WriteString(Styles.Error.Render(m.err.Error()) + "\n\n")
	}

	s = *f(&s)

	s.WriteString(
		"\n\n  ↑/↓ — navigation\n  Esc — back\n  Enter — select\n Ctrl+C — quit\n",
	)

	return s.String()
}

func (m BaseModel) getMaxCursor() int {
	return len(m.Options) - 1
}
