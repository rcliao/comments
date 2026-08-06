package tui

import "github.com/charmbracelet/lipgloss"

// All package-level styles are built from the active theme by applyTheme
// (called from theme.go's init and SetTheme). Call sites keep using these
// vars; only the colors change per theme.
var (
	// Title and headers
	titleStyle lipgloss.Style

	// Comment markers
	commentMarkerStyle lipgloss.Style

	// Line numbers
	lineNumberStyle lipgloss.Style

	// Help text
	helpStyle lipgloss.Style

	// Comment panel
	commentPanelStyle lipgloss.Style

	// Selected comment
	selectedCommentStyle lipgloss.Style

	// Sidebar group header (line + thread count badge)
	groupHeaderStyle lipgloss.Style

	// Gutter marker: unresolved blocking threads on this line
	blockingMarkerStyle lipgloss.Style

	// Gutter marker: all threads on this line resolved
	resolvedMarkerStyle lipgloss.Style

	// Reply author/timestamp line in expanded sidebar threads
	replyMetaStyle lipgloss.Style

	// Cursor (for line selection): subtle cursorline background, text left readable
	cursorStyle lipgloss.Style

	// Cursor arrow + line number accent on the focused line
	cursorAccentStyle lipgloss.Style

	// Accented line number, same 4-cell right-aligned box as lineNumberStyle
	cursorLineNumStyle lipgloss.Style

	// Range selection
	rangeMarkerStyle  lipgloss.Style
	selectedLineStyle lipgloss.Style

	// Virtual-text line summaries (dimmed end-of-line thread digest)
	virtualTextStyle lipgloss.Style

	// NEW-activity badge: thread has replies newer than the last signoff
	newBadgeStyle lipgloss.Style

	// Dimmed round separators between replies that straddle a signoff
	roundSeparatorStyle lipgloss.Style

	// In-place markdown span styling: syntax glyphs stay visible but dimmed
	syntaxGlyphStyle lipgloss.Style

	// Bold/italic span content (glyphs excluded, handled by syntaxGlyphStyle)
	boldSpanStyle   lipgloss.Style
	italicSpanStyle lipgloss.Style

	// Inline code span content
	codeSpanStyle lipgloss.Style

	// List bullets (-, *, +, numbered)
	bulletStyle lipgloss.Style

	// Blockquote > bars
	quoteBarStyle lipgloss.Style

	// Modal overlay
	modalOverlayStyle lipgloss.Style

	// Markdown headings (whole-line; heading4Style covers H4-H6)
	heading1Style lipgloss.Style
	heading2Style lipgloss.Style
	heading3Style lipgloss.Style
	heading4Style lipgloss.Style
)

// applyTheme rebuilds every package-level style from the given theme and
// records it as the active theme.
func applyTheme(t Theme) {
	activeTheme = t

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Title)

	commentMarkerStyle = lipgloss.NewStyle().
		Foreground(t.Marker).
		Bold(true)

	lineNumberStyle = lipgloss.NewStyle().
		Foreground(t.LineNumber).
		Width(4).
		Align(lipgloss.Right)

	helpStyle = lipgloss.NewStyle().
		Foreground(t.HelpText)

	commentPanelStyle = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.GroupHeader).
		Padding(0, 1)

	selectedCommentStyle = lipgloss.NewStyle().
		Background(t.CursorLineBg)

	groupHeaderStyle = lipgloss.NewStyle().
		Foreground(t.GroupHeader).
		Bold(true)

	blockingMarkerStyle = lipgloss.NewStyle().
		Foreground(t.Blocking).
		Bold(true)

	resolvedMarkerStyle = lipgloss.NewStyle().
		Foreground(t.Resolved)

	replyMetaStyle = lipgloss.NewStyle().
		Foreground(t.ReplyMeta)

	cursorStyle = lipgloss.NewStyle().
		Background(t.CursorLineBg)

	cursorAccentStyle = lipgloss.NewStyle().
		Foreground(t.CursorAccent).
		Bold(true)

	cursorLineNumStyle = lipgloss.NewStyle().
		Foreground(t.CursorAccent).
		Bold(true).
		Width(4).
		Align(lipgloss.Right)

	rangeMarkerStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	selectedLineStyle = lipgloss.NewStyle().
		Background(t.SelectionBg)

	virtualTextStyle = lipgloss.NewStyle().
		Foreground(t.VirtualText).
		Italic(true)

	newBadgeStyle = lipgloss.NewStyle().
		Foreground(t.New).
		Bold(true)

	roundSeparatorStyle = lipgloss.NewStyle().
		Foreground(t.DimSyntax)

	syntaxGlyphStyle = lipgloss.NewStyle().
		Foreground(t.DimSyntax)

	boldSpanStyle = lipgloss.NewStyle().Bold(true)
	italicSpanStyle = lipgloss.NewStyle().Italic(true)

	codeSpanStyle = lipgloss.NewStyle().
		Foreground(t.Code)

	bulletStyle = lipgloss.NewStyle().
		Foreground(t.Bullet).
		Bold(true)

	quoteBarStyle = lipgloss.NewStyle().
		Foreground(t.Quote).
		Bold(true)

	modalOverlayStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Title).
		Padding(1, 2)

	heading1Style = lipgloss.NewStyle().Bold(true).Foreground(t.Heading1)
	heading2Style = lipgloss.NewStyle().Bold(true).Foreground(t.Heading2)
	heading3Style = lipgloss.NewStyle().Bold(true).Foreground(t.Heading3)
	heading4Style = lipgloss.NewStyle().Bold(true).Foreground(t.Heading4)
}
