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
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(0, 1)

	// Selected comment
	selectedCommentStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237"))

	// Sidebar group header (line + thread count badge)
	groupHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("63")).
				Bold(true)

	// Gutter marker: unresolved blocking threads on this line
	blockingMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	// Gutter marker: all threads on this line resolved
	resolvedMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("28"))

	// Reply author/timestamp line in expanded sidebar threads
	replyMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	// Cursor (for line selection)
	cursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230"))

	// Range selection
	rangeMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)

	selectedLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235"))

	// Virtual-text line summaries (dimmed end-of-line thread digest)
	virtualTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)

	// NEW-activity badge: thread has replies newer than the last signoff
	newBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	// Dimmed round separators between replies that straddle a signoff
	roundSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	// In-place markdown span styling: syntax glyphs stay visible but dimmed
	syntaxGlyphStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	// Bold/italic span content (glyphs excluded, handled by syntaxGlyphStyle)
	boldSpanStyle   = lipgloss.NewStyle().Bold(true)
	italicSpanStyle = lipgloss.NewStyle().Italic(true)

	// Inline code span content
	codeSpanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("173"))

	// List bullets (-, *, +, numbered)
	bulletStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	// Blockquote > bars
	quoteBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("108")).
			Bold(true)

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
