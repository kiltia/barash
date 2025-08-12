package tui

import (
	"errors"
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
			return FileSelectedMsg{Path: "", Action: action, Error: err}
		}
	}
	tmpPath := tmpFile.Name()
	err = tmpFile.Close()
	if err != nil {
		return m, func() tea.Msg {
			return FileSelectedMsg{Path: "", Action: action, Error: err}
		}
	}
	cmd := exec.Command("sh", "-c", "fzf --print-query --prompt='"+prompt+"' > "+tmpPath)

	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer func() {
			_ = os.Remove(tmpPath)
		}()

		var exitError *exec.ExitError
		if err != nil && errors.As(err, &exitError) && (action == ActionLoad || exitError.ExitCode() == 2) {
			return FileSelectedMsg{Path: "", Action: action, Error: err}
		}

		output, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return FileSelectedMsg{Path: "", Action: action, Error: readErr}
		}

		out := strings.TrimSpace(string(output))
		lines := strings.Split(out, "\n")
		var path string
		if len(lines) > 1 {
			if lines[1] != "" {
				path = lines[1]
			} else {
				path = lines[0]
			}
		} else {
			path = lines[0]
		}
		return FileSelectedMsg{Path: path, Action: action}
	})
}
