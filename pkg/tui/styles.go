package tui

import lipgloss "charm.land/lipgloss/v2"

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

	// Resolvable file references (citations): link-styled, followable with f
	refLink lipgloss.Style

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
	strikeSpan lipgloss.Style

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
			Foreground(t.Title.Color()),

		commentMarker: lipgloss.NewStyle().
			Foreground(t.Marker.Color()).
			Bold(true),

		lineNumber: lipgloss.NewStyle().
			Foreground(t.LineNumber.Color()).
			Width(4).
			Align(lipgloss.Right),

		help: lipgloss.NewStyle().
			Foreground(t.HelpText.Color()),

		commentPanel: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(t.GroupHeader.Color()).
			Padding(0, 1),

		selectedComment: lipgloss.NewStyle().
			Background(t.CursorLineBg.Color()),

		groupHeader: lipgloss.NewStyle().
			Foreground(t.GroupHeader.Color()).
			Bold(true),

		blockingMarker: lipgloss.NewStyle().
			Foreground(t.Blocking.Color()).
			Bold(true),

		resolvedMarker: lipgloss.NewStyle().
			Foreground(t.Resolved.Color()),

		replyMeta: lipgloss.NewStyle().
			Foreground(t.ReplyMeta.Color()),

		cursor: lipgloss.NewStyle().
			Background(t.CursorLineBg.Color()),

		cursorAccent: lipgloss.NewStyle().
			Foreground(t.CursorAccent.Color()).
			Bold(true),

		cursorLineNum: lipgloss.NewStyle().
			Foreground(t.CursorAccent.Color()).
			Bold(true).
			Width(4).
			Align(lipgloss.Right),

		refLink: lipgloss.NewStyle().
			Foreground(t.Accent.Color()).
			Underline(true),

		rangeMarker: lipgloss.NewStyle().
			Foreground(t.Accent.Color()).
			Bold(true),

		selectedLine: lipgloss.NewStyle().
			Background(t.SelectionBg.Color()),

		virtualText: lipgloss.NewStyle().
			Foreground(t.VirtualText.Color()).
			Italic(true),

		newBadge: lipgloss.NewStyle().
			Foreground(t.New.Color()).
			Bold(true),

		roundSeparator: lipgloss.NewStyle().
			Foreground(t.DimSyntax.Color()),

		syntaxGlyph: lipgloss.NewStyle().
			Foreground(t.DimSyntax.Color()),

		boldSpan:   lipgloss.NewStyle().Bold(true),
		italicSpan: lipgloss.NewStyle().Italic(true),
		strikeSpan: lipgloss.NewStyle().Strikethrough(true).Foreground(t.DimSyntax.Color()),

		codeSpan: lipgloss.NewStyle().
			Foreground(t.Code.Color()),

		bullet: lipgloss.NewStyle().
			Foreground(t.Bullet.Color()).
			Bold(true),

		quoteBar: lipgloss.NewStyle().
			Foreground(t.Quote.Color()).
			Bold(true),

		modalOverlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Title.Color()).
			Padding(1, 2),

		heading1: lipgloss.NewStyle().Bold(true).Foreground(t.Heading1.Color()),
		heading2: lipgloss.NewStyle().Bold(true).Foreground(t.Heading2.Color()),
		heading3: lipgloss.NewStyle().Bold(true).Foreground(t.Heading3.Color()),
		heading4: lipgloss.NewStyle().Bold(true).Foreground(t.Heading4.Color()),
	}
}
