package tui

// Color themes: every color the TUI renders is a named role on Theme, and
// styles.go builds all package-level styles from the active theme via
// applyTheme. Selection: `comments view --theme <name>` or COMMENTS_THEME
// (flag wins); unknown names fall back to the default (nord).

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

// Theme maps every color role the TUI uses to a concrete color.
// Roles, not widgets: several widgets may share one role.
type Theme struct {
	// Markdown headings (whole-line, by level; Heading4 covers H4-H6)
	Heading1 lipgloss.Color
	Heading2 lipgloss.Color
	Heading3 lipgloss.Color
	Heading4 lipgloss.Color

	// Chrome
	Title      lipgloss.Color // titles, modal headers, modal borders, focus arrow
	DimSyntax  lipgloss.Color // dimmed markdown glyphs, round separators, dim labels
	LineNumber lipgloss.Color // gutter line numbers
	HelpText   lipgloss.Color // hint bars and modal help lines
	MetaText   lipgloss.Color // author/timestamp meta, context labels
	ReplyMeta  lipgloss.Color // reply author/timestamp rows in the sidebar
	Border     lipgloss.Color // plain borders (input fields, context boxes)

	// Cursor and selection
	CursorAccent lipgloss.Color // cursor arrow + focused line number
	CursorLineBg lipgloss.Color // cursorline background
	SelectionBg  lipgloss.Color // range selection / highlighted-line background
	SelectionFg  lipgloss.Color // foreground on highlighted lines
	Accent       lipgloss.Color // range markers, key hints, section labels

	// Gutter markers and badges
	Blocking    lipgloss.Color // unresolved blocking threads
	Marker      lipgloss.Color // unresolved thread marker + open counts
	Resolved    lipgloss.Color // resolved checkmark
	VirtualText lipgloss.Color // end-of-line thread summaries
	GroupHeader lipgloss.Color // sidebar group headers, panel/thread borders
	New         lipgloss.Color // NEW-activity badge

	// Inline markdown spans
	Code   lipgloss.Color // inline code spans
	Bullet lipgloss.Color // list bullets
	Quote  lipgloss.Color // blockquote bars

	// Alerts
	Warning lipgloss.Color // suggestion form accents

	// Comment type prefixes
	TypeQ lipgloss.Color // [Q] Question
	TypeS lipgloss.Color // [S] Suggestion
	TypeB lipgloss.Color // [B] Blocker/Bug
	TypeT lipgloss.Color // [T] Technical/TODO
	TypeE lipgloss.Color // [E] Editorial/Enhancement
}

// DefaultThemeName is applied at startup and on unknown theme names.
const DefaultThemeName = "nord"

// themes is the registry of selectable themes.
var themes = map[string]Theme{
	// nord: the official Nord palette (https://www.nordtheme.com).
	// Polar night #2E3440-#4C566A, snow storm #D8DEE9-#ECEFF4,
	// frost #8FBCBB/#88C0D0/#81A1C1/#5E81AC, aurora #BF616A/#D08770/#EBCB8B/#A3BE8C/#B48EAD.
	"nord": {
		Heading1: "#88C0D0", Heading2: "#81A1C1", Heading3: "#8FBCBB", Heading4: "#5E81AC",
		Title: "#B48EAD", DimSyntax: "#4C566A", LineNumber: "#4C566A", HelpText: "#4C566A",
		MetaText: "#D8DEE9", ReplyMeta: "#4C566A", Border: "#4C566A",
		CursorAccent: "#B48EAD", CursorLineBg: "#3B4252", SelectionBg: "#434C5E",
		SelectionFg: "#ECEFF4", Accent: "#88C0D0",
		Blocking: "#BF616A", Marker: "#B48EAD", Resolved: "#A3BE8C",
		VirtualText: "#4C566A", GroupHeader: "#81A1C1", New: "#EBCB8B",
		Code: "#D08770", Bullet: "#D08770", Quote: "#A3BE8C", Warning: "#EBCB8B",
		TypeQ: "#EBCB8B", TypeS: "#81A1C1", TypeB: "#BF616A", TypeT: "#B48EAD", TypeE: "#88C0D0",
	},

	// dracula: the canonical Dracula palette (https://draculatheme.com).
	"dracula": {
		Heading1: "#BD93F9", Heading2: "#FF79C6", Heading3: "#8BE9FD", Heading4: "#50FA7B",
		Title: "#BD93F9", DimSyntax: "#6272A4", LineNumber: "#6272A4", HelpText: "#6272A4",
		MetaText: "#6272A4", ReplyMeta: "#6272A4", Border: "#6272A4",
		CursorAccent: "#FF79C6", CursorLineBg: "#44475A", SelectionBg: "#44475A",
		SelectionFg: "#F8F8F2", Accent: "#8BE9FD",
		Blocking: "#FF5555", Marker: "#FF79C6", Resolved: "#50FA7B",
		VirtualText: "#6272A4", GroupHeader: "#BD93F9", New: "#FFB86C",
		Code: "#F1FA8C", Bullet: "#FFB86C", Quote: "#50FA7B", Warning: "#F1FA8C",
		TypeQ: "#F1FA8C", TypeS: "#8BE9FD", TypeB: "#FF5555", TypeT: "#BD93F9", TypeE: "#50FA7B",
	},

	// gruvbox: the canonical Gruvbox dark palette (https://github.com/morhetz/gruvbox).
	"gruvbox": {
		Heading1: "#FE8019", Heading2: "#FABD2F", Heading3: "#B8BB26", Heading4: "#83A598",
		Title: "#D3869B", DimSyntax: "#928374", LineNumber: "#928374", HelpText: "#928374",
		MetaText: "#A89984", ReplyMeta: "#928374", Border: "#665C54",
		CursorAccent: "#FE8019", CursorLineBg: "#3C3836", SelectionBg: "#504945",
		SelectionFg: "#FBF1C7", Accent: "#83A598",
		Blocking: "#FB4934", Marker: "#D3869B", Resolved: "#B8BB26",
		VirtualText: "#928374", GroupHeader: "#83A598", New: "#FABD2F",
		Code: "#8EC07C", Bullet: "#FE8019", Quote: "#8EC07C", Warning: "#FABD2F",
		TypeQ: "#FABD2F", TypeS: "#83A598", TypeB: "#FB4934", TypeT: "#D3869B", TypeE: "#8EC07C",
	},

	// ansi: the exact 256-color values the TUI shipped with before themes
	// existed (backward-compatible legacy look).
	"ansi": {
		Heading1: "99", Heading2: "33", Heading3: "39", Heading4: "45",
		Title: "170", DimSyntax: "240", LineNumber: "240", HelpText: "241",
		MetaText: "242", ReplyMeta: "243", Border: "240",
		CursorAccent: "205", CursorLineBg: "237", SelectionBg: "235",
		SelectionFg: "255", Accent: "39",
		Blocking: "196", Marker: "212", Resolved: "28",
		VirtualText: "240", GroupHeader: "63", New: "205",
		Code: "173", Bullet: "214", Quote: "108", Warning: "3",
		TypeQ: "220", TypeS: "33", TypeB: "196", TypeT: "13", TypeE: "14",
	},
}

// activeTheme is the theme currently applied to the package-level styles.
// Read it for one-off local styles; never write it directly (use applyTheme).
var activeTheme Theme

func init() {
	applyTheme(themes[DefaultThemeName])
}

// ThemeByName looks a theme up by name. Unknown names return the default
// (nord) theme and ok=false.
func ThemeByName(name string) (t Theme, ok bool) {
	if t, ok = themes[name]; ok {
		return t, true
	}
	return themes[DefaultThemeName], false
}

// ThemeNames returns the selectable theme names, sorted.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetTheme applies the named theme to all TUI styles. Unknown names apply
// the default (nord) and return false so the caller can warn.
func SetTheme(name string) bool {
	t, ok := ThemeByName(name)
	applyTheme(t)
	return ok
}
