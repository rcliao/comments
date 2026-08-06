package tui

import "github.com/charmbracelet/lipgloss"

// styleSet bundles every style the TUI renders, built once from a Theme by
// newStyleSet. The Model holds a *styleSet captured at construction, so
// rendering never reads mutable package state — SetTheme in one test cannot
// race renders in another (t.Parallel-safe). A styleSet is immutable after
// construction; share it freely.
type styleSet struct {
	// theme keeps the source colors for one-off local styles
	theme Theme

	// Title and headers
	title lipgloss.Style

	// Comment markers
	commentMarker lipgloss.Style

	// Line numbers
	lineNumber lipgloss.Style

	// Help text
	help lipgloss.Style

	// Comment panel
	commentPanel lipgloss.Style

	// Selected comment
	selectedComment lipgloss.Style

	// Sidebar group header (line + thread count badge)
	groupHeader lipgloss.Style

	// Gutter marker: unresolved blocking threads on this line
	blockingMarker lipgloss.Style

	// Gutter marker: all threads on this line resolved
	resolvedMarker lipgloss.Style

	// Reply author/timestamp line in expanded sidebar threads
	replyMeta lipgloss.Style

	// Cursor (for line selection): subtle cursorline background, text left readable
	cursor lipgloss.Style

	// Cursor arrow + line number accent on the focused line
	cursorAccent lipgloss.Style

	// Accented line number, same 4-cell right-aligned box as lineNumber
	cursorLineNum lipgloss.Style

	// Range selection
	rangeMarker  lipgloss.Style
	selectedLine lipgloss.Style

	// Virtual-text line summaries (dimmed end-of-line thread digest)
	virtualText lipgloss.Style

	// NEW-activity badge: thread has replies newer than the last signoff
	newBadge lipgloss.Style

	// Dimmed round separators between replies that straddle a signoff
	roundSeparator lipgloss.Style

	// In-place markdown span styling: syntax glyphs stay visible but dimmed
	syntaxGlyph lipgloss.Style

	// Bold/italic span content (glyphs excluded, handled by syntaxGlyph)
	boldSpan   lipgloss.Style
	italicSpan lipgloss.Style

	// Inline code span content
	codeSpan lipgloss.Style

	// List bullets (-, *, +, numbered)
	bullet lipgloss.Style

	// Blockquote > bars
	quoteBar lipgloss.Style

	// Modal overlay
	modalOverlay lipgloss.Style

	// Markdown headings (whole-line; heading4 covers H4-H6)
	heading1 lipgloss.Style
	heading2 lipgloss.Style
	heading3 lipgloss.Style
	heading4 lipgloss.Style
}

// newStyleSet builds every style from the given theme.
func newStyleSet(t Theme) *styleSet {
	return &styleSet{
		theme: t,

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Title),

		commentMarker: lipgloss.NewStyle().
			Foreground(t.Marker).
			Bold(true),

		lineNumber: lipgloss.NewStyle().
			Foreground(t.LineNumber).
			Width(4).
			Align(lipgloss.Right),

		help: lipgloss.NewStyle().
			Foreground(t.HelpText),

		commentPanel: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(t.GroupHeader).
			Padding(0, 1),

		selectedComment: lipgloss.NewStyle().
			Background(t.CursorLineBg),

		groupHeader: lipgloss.NewStyle().
			Foreground(t.GroupHeader).
			Bold(true),

		blockingMarker: lipgloss.NewStyle().
			Foreground(t.Blocking).
			Bold(true),

		resolvedMarker: lipgloss.NewStyle().
			Foreground(t.Resolved),

		replyMeta: lipgloss.NewStyle().
			Foreground(t.ReplyMeta),

		cursor: lipgloss.NewStyle().
			Background(t.CursorLineBg),

		cursorAccent: lipgloss.NewStyle().
			Foreground(t.CursorAccent).
			Bold(true),

		cursorLineNum: lipgloss.NewStyle().
			Foreground(t.CursorAccent).
			Bold(true).
			Width(4).
			Align(lipgloss.Right),

		rangeMarker: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		selectedLine: lipgloss.NewStyle().
			Background(t.SelectionBg),

		virtualText: lipgloss.NewStyle().
			Foreground(t.VirtualText).
			Italic(true),

		newBadge: lipgloss.NewStyle().
			Foreground(t.New).
			Bold(true),

		roundSeparator: lipgloss.NewStyle().
			Foreground(t.DimSyntax),

		syntaxGlyph: lipgloss.NewStyle().
			Foreground(t.DimSyntax),

		boldSpan:   lipgloss.NewStyle().Bold(true),
		italicSpan: lipgloss.NewStyle().Italic(true),

		codeSpan: lipgloss.NewStyle().
			Foreground(t.Code),

		bullet: lipgloss.NewStyle().
			Foreground(t.Bullet).
			Bold(true),

		quoteBar: lipgloss.NewStyle().
			Foreground(t.Quote).
			Bold(true),

		modalOverlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Title).
			Padding(1, 2),

		heading1: lipgloss.NewStyle().Bold(true).Foreground(t.Heading1),
		heading2: lipgloss.NewStyle().Bold(true).Foreground(t.Heading2),
		heading3: lipgloss.NewStyle().Bold(true).Foreground(t.Heading3),
		heading4: lipgloss.NewStyle().Bold(true).Foreground(t.Heading4),
	}
}
