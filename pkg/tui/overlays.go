package tui

// Help and TOC overlays: full-screen keybinding reference (?) and section
// jump list (t). Both are read-only overlays over the review flow.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/markdown"
)

// helpGroup is one titled section of the keybinding reference
type helpGroup struct {
	title    string
	bindings [][2]string // key, description
}

// helpGroups returns the keybinding reference grouped by review activity.
// Pure data so the overlay is testable without a terminal.
func helpGroups() []helpGroup {
	return []helpGroup{
		{"Move", [][2]string{
			{"j/k", "move cursor / selection"},
			{"Ctrl+D / Ctrl+U", "half page down / up"},
			{"g / G", "first / last line"},
			{"] / [", "next / previous open comment"},
			{"t", "table of contents (jump to section)"},
		}},
		{"Threads", [][2]string{
			{"Enter", "expand selected thread"},
			{"r", "reply / dive into thread at cursor"},
			{"Tab", "cycle threads stacked on one line"},
			{"R", "toggle resolved threads in sidebar"},
			{"x", "resolve thread (in thread view)"},
		}},
		{"Compose", [][2]string{
			{"c", "add comment at cursor line"},
			{"s", "suggest an edit (range or section)"},
			{"Ctrl+S", "save comment / reply / suggestion"},
			{"Ctrl+P / Ctrl+T", "cycle priority / type while composing"},
			{"Esc", "cancel compose"},
		}},
		{"Review", [][2]string{
			{"a / x", "queue suggestion accept / reject (applied at verdict)"},
			{"S", "cycle sidebar density (full / condensed / hidden)"},
			{"L", "toggle end-of-line thread summaries"},
		}},
		{"Exit", [][2]string{
			{"q", "verdict dialog: approve / request changes"},
			{"Ctrl+C", "quit silently (no verdict; keeps queue unapplied)"},
			{"?", "this help (any key closes)"},
		}},
	}
}

// renderHelpOverlay renders the grouped keybinding reference as plain text
func renderHelpOverlay() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("Keybindings"))
	b.WriteString("\n")
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	for _, g := range helpGroups() {
		b.WriteString("\n")
		b.WriteString(groupHeaderStyle.Render(g.title))
		b.WriteString("\n")
		for _, kv := range g.bindings {
			b.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render(fmt.Sprintf("%-16s", kv[0])), kv[1]))
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("press any key to close"))
	return b.String()
}

// handleHelpKeys closes the help overlay on any key
func (m Model) handleHelpKeys(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = m.helpReturnMode
	return m, nil
}

// viewHelp renders the full-screen help overlay
func (m Model) viewHelp() string {
	box := modalOverlayStyle.Render(renderHelpOverlay())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// tocEntry is one row of the TOC overlay: a section path with its open count
type tocEntry struct {
	path string
	line int
	open int
}

// buildTOC flattens the section tree into TOC rows with per-section open
// comment counts (a section's count includes threads in its sub-sections)
func buildTOC(ds *markdown.DocumentStructure, threads []*comment.Comment) []tocEntry {
	if ds == nil {
		return nil
	}
	entries := []tocEntry{}
	var walk func(secs []*markdown.Section)
	walk = func(secs []*markdown.Section) {
		for _, s := range secs {
			open := 0
			for _, t := range threads {
				if isOpenThread(t) && t.Line >= s.StartLine && t.Line <= s.EndLine {
					open++
				}
			}
			entries = append(entries, tocEntry{
				path: s.GetFullPath(ds.SectionsByID),
				line: s.StartLine,
				open: open,
			})
			walk(s.Children)
		}
	}
	walk(ds.Sections)
	return entries
}

// renderTOC renders the TOC rows with the selected row highlighted
func renderTOC(entries []tocEntry, selected int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("Table of Contents"))
	b.WriteString("\n\n")
	openStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	for i, e := range entries {
		cursor := "  "
		style := lipgloss.NewStyle()
		if i == selected {
			cursor = cursorStyle.Render("▶") + " "
			style = selectedCommentStyle
		}
		row := fmt.Sprintf("%s (line %d)", e.path, e.line)
		b.WriteString(cursor + style.Render(row))
		if e.open > 0 {
			b.WriteString(" " + openStyle.Render(fmt.Sprintf("· %d open", e.open)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: move • Enter: jump • Esc: close"))
	return b.String()
}

// openTOC enters the TOC overlay from the current mode
func (m *Model) openTOC() {
	m.tocEntries = buildTOC(m.documentSections, m.doc.Threads)
	if len(m.tocEntries) == 0 {
		return
	}
	m.tocReturnMode = m.mode
	m.tocSelected = 0
	m.mode = ModeTOC
}

// handleTOCKeys navigates the TOC overlay; Enter jumps to the section heading
func (m Model) handleTOCKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.tocSelected < len(m.tocEntries)-1 {
			m.tocSelected++
		}
		return m, nil

	case "k", "up":
		if m.tocSelected > 0 {
			m.tocSelected--
		}
		return m, nil

	case "enter":
		if m.tocSelected < 0 || m.tocSelected >= len(m.tocEntries) {
			m.mode = m.tocReturnMode
			return m, nil
		}
		entry := m.tocEntries[m.tocSelected]
		m.selectedLine = entry.line
		m.mode = ModeLineSelect
		m.documentViewport = viewport.New(m.docPaneWidth(), m.height-2)
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
		return m, nil

	case "esc", "q", "t":
		m.mode = m.tocReturnMode
		return m, nil
	}
	return m, nil
}

// viewTOC renders the TOC overlay
func (m Model) viewTOC() string {
	box := modalOverlayStyle.Render(renderTOC(m.tocEntries, m.tocSelected))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
