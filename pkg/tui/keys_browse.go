package tui

// Browse mode: read-only navigation of the document and comment sidebar.
// Line-select shares this screen (viewBrowse) but has its own key handler in
// keys_lineselect.go.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/rcliao/comments/pkg/comment"
)

// handleBrowseKeys handles keys in browse mode
func (m Model) handleBrowseKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		// Exit is a verdict (approved TUI design): q opens the verdict dialog
		if m.doc != nil {
			m.verdictReturnMode = ModeBrowse
			m.mode = ModeVerdict
			return m, nil
		}
		if m.startedWithFile {
			return m, tea.Quit
		}
		m.mode = ModeFilePicker
		m.doc = nil
		m.filename = ""
		m.ready = false
		return m, nil

	case "c":
		// Enter line selection mode to add comment; keep a restored or
		// previously used cursor position instead of jumping to the top
		m.mode = ModeLineSelect
		m.selectedLine = max(m.selectedLine, 1)

		// Completely reset the viewport to fix scroll offset issues
		m.documentViewport = newViewport(m.docPaneWidth(), m.height-2)
		m.refreshCursorView()
		return m, nil

	case "?":
		m.helpReturnMode = m.mode
		m.mode = ModeHelp
		return m, nil

	case "S":
		m.cycleSidebarDensity()
		return m, nil

	case "L":
		m.toggleLineSummaries()
		return m, nil

	case "t":
		m.openTOC()
		return m, nil

	case "j", "down":
		// Navigate comments
		visibleComments := m.visibleComments()
		if m.selectedComment < len(visibleComments)-1 {
			m.selectedComment++
			m.commentViewport.SetContent(m.renderComments())
			// Scroll document to center the selected comment
			m.scrollToComment(visibleComments[m.selectedComment])
		}
		return m, nil

	case "k", "up":
		visibleComments := m.visibleComments()
		if m.selectedComment > 0 {
			m.selectedComment--
			m.commentViewport.SetContent(m.renderComments())
			// Scroll document to center the selected comment
			m.scrollToComment(visibleComments[m.selectedComment])
		}
		return m, nil

	case "enter":
		// Expand selected comment thread
		visibleComments := m.visibleComments()
		if len(visibleComments) > 0 && m.selectedComment < len(visibleComments) {
			selectedThread := visibleComments[m.selectedComment]
			m.selectedThread = selectedThread
			m.mode = ModeThreadView
			m.threadViewport.SetContent(m.renderThread())
			// Scroll document to center the thread's comment
			m.scrollToComment(selectedThread)
			return m, nil
		}
		return m, nil

	case "R":
		// Toggle showing resolved comments
		m.showResolved = !m.showResolved
		m.commentViewport.SetContent(m.renderComments())
		return m, nil
	}

	return m, nil
}

// scrollToComment scrolls the document viewport to center the given comment
func (m *Model) scrollToComment(c *comment.Comment) {
	if m.doc == nil || c == nil {
		return
	}

	// Get the comment's line position (line-only tracking in v2.0)
	targetLine := c.Line
	if targetLine < 1 {
		return
	}

	// Rendered rows before the target line, accounting for line wrapping
	// (same wrap math as the document render — see docWrapWidth)
	displayRow := m.calculateDisplayRow(targetLine - 1)

	// Center the line in the viewport, clamped to the start
	m.documentViewport.SetYOffset(max(displayRow-m.documentViewport.Height()/2, 0))
}

// viewBrowse renders the browse/line-select view
func (m Model) viewBrowse() string {
	if !m.ready {
		return "Loading..."
	}

	modeStr := m.mode.String()
	title := m.styles.title.Render(fmt.Sprintf("📄 %s - %s", m.filename, modeStr))

	var helpText string
	if m.mode == ModeLineSelect {
		helpText = "j/k: move • r: open thread • f: follow ref • n/N: next/prev NEW • Tab: cycle threads • c: comment • s: suggest • t: TOC • ?: help • Esc: cancel"
	} else {
		quitText := "back"
		if m.startedWithFile {
			quitText = "quit"
		}
		helpText = fmt.Sprintf("j/k: navigate • c: comment • Enter: expand • t: TOC • S: sidebar • ?: help • q: %s", quitText)
	}
	help := m.styles.help.Render(helpText)

	// Layout: document on left, comments on right (unless sidebar is hidden)
	var content string
	if m.sidebarDensity == densityHidden {
		content = m.documentViewport.View()
	} else {
		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.documentViewport.View(),
			m.styles.commentPanel.Render(m.commentViewport.View()),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		help,
	)
}
