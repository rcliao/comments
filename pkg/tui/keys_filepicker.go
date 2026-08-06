package tui

// File-picker mode: choose a markdown file to review.

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleFilePickerKeys handles keys in file picker mode
func (m Model) handleFilePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	// Check if a file was selected
	if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
		return m.loadFile(path)
	}

	return m, cmd
}

// viewFilePicker renders the file picker view
func (m Model) viewFilePicker() string {
	title := m.styles.title.Render("comments - Select a markdown file")
	help := m.styles.help.Render("↑/↓: navigate • Enter: select • q: quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.filePicker.View(),
		"",
		help,
	)
}
