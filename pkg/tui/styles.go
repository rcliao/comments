package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Title and headers
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	// Comment markers
	commentMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	// Line numbers
	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(4).
			Align(lipgloss.Right)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Comment panel
	commentPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(1)

	// Selected comment
	selectedCommentStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237"))

	// Cursor (for line selection)
	cursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))

	// Modal overlay
	modalOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("170")).
				Padding(1, 2)

	// Input field
	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)
