package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func SelectFile(m tea.Model, action Action) (tea.Model, tea.Cmd) {
	prompt := "Select file: "
	if action == ActionSave {
		prompt = "Save to: "
	}

	tmpFile, err := os.CreateTemp("", "fzf_output_*")
	if err != nil {
		return m, func() tea.Msg {
			return FileSelectedMsg{Path: "", Action: action}
		}
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	cmd := exec.Command("sh", "-c", "fzf --prompt='"+prompt+"' > "+tmpPath)

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer func() {
			_ = os.Remove(tmpPath)
		}()

		if err != nil {
			return FileSelectedMsg{Path: "", Action: action}
		}

		output, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return FileSelectedMsg{Path: "", Action: action}
		}

		path := strings.TrimSpace(string(output))
		return FileSelectedMsg{Path: path, Action: action}
	})
}
