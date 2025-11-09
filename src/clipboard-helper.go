package src

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func CopyToClipboard(s string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(s)
		return ClipboardMsg{Success: err == nil, Err: err}
	}
}
