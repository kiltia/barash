package tui

import (
	"fmt"
	"strings"
)

func renderConfigEdit(
	s *strings.Builder,
	m ConfigEditorModel,
) *strings.Builder {
	if len(m.Path) > 0 {
		fmt.Fprintf(s, "Path: %s\n\n", strings.Join(m.Path, " → "))
	}

	for i, item := range m.Options {
		display := item.Name
		if item.IsStruct {
			display += " →"
		} else {
			display += fmt.Sprintf(": %s", FormatValue(item.Value))
		}

		if i == m.cursor {
			fmt.Fprintf(s,
				"→ %s\n", Styles.Selected.Render(display))
		} else {
			fmt.Fprintf(s, "  %s\n", Styles.Normal.Render(display))
		}
	}

	return s
}

func renderConfigPreview(m ConfigPreviewModel, options []string) string {
	var s strings.Builder

	// Show current config as env format
	envOutput := ConfigToEnv(m.Config)
	if envOutput == "" {
		s.WriteString(Styles.Normal.Render("No configuration loaded") + "\n\n")
	} else {
		s.WriteString(Styles.Normal.Render("Current configuration (env format):") + "\n\n")

		// Format env variables with colors
		lines := strings.SplitSeq(strings.TrimSpace(envOutput), "\n")
		for line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := Styles.ConfigVar.Render(parts[0])
				value := Styles.ConfigValue.Render(parts[1])
				s.WriteString(fmt.Sprintf("%s=%s\n", varName, value))
			} else {
				s.WriteString(line + "\n")
			}
		}
	}

	s.WriteString("\n")

	// Options
	for i, option := range options {
		if i == m.cursor {
			s.WriteString(fmt.Sprintf("→ %s\n", Styles.Selected.Render(option)))
		} else {
			s.WriteString(fmt.Sprintf("  %s\n", Styles.Normal.Render(option)))
		}
	}

	return s.String()
}

func renderMenu(
	s *strings.Builder,
	cursor int,
	options []string,
) *strings.Builder {
	for i, option := range options {
		if i == cursor {
			fmt.Fprintf(s, "→ %s\n", Styles.Selected.Render(option))
		} else {
			fmt.Fprintf(s, "  %s\n", Styles.Normal.Render(option))
		}
	}

	return s
}
