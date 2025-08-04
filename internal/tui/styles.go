package tui

import "github.com/charmbracelet/lipgloss"

var Styles = struct {
	Title       lipgloss.Style
	Selected    lipgloss.Style
	Normal      lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	ConfigVar   lipgloss.Style
	ConfigValue lipgloss.Style
}{
	Title: lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")),
	Selected: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#F25D94")),
	Normal:  lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
	Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")),
	Success: lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")),
	ConfigVar: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true),
	ConfigValue: lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")),
}
