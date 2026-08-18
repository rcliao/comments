package tui

// Browse mode: read-only navigation of the document and comment sidebar.
// Line-select shares this screen (viewBrowse) but has its own key handler in
// keys_lineselect.go.

import (
	"fmt"
	"strings"

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
		m.documentViewport = newViewport(m.docPaneWidth(), m.contentHeight())
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

	case "#":
		m.hideLineNumbers = !m.hideLineNumbers
		m.refreshDocumentPane()
		return m, nil

	case "/":
		return m.openSearch()

	case "g":
		// vim: first thread
		if len(m.visibleComments()) > 0 {
			m.selectedComment = 0
			m.scrollSidebarToSelected()
			m.scrollToComment(m.visibleComments()[0])
			m.refreshDocumentPane()
		}
		return m, nil

	case "G":
		// vim: last thread
		if vc := m.visibleComments(); len(vc) > 0 {
			m.selectedComment = len(vc) - 1
			m.scrollSidebarToSelected()
			m.scrollToComment(vc[m.selectedComment])
			m.refreshDocumentPane()
		}
		return m, nil

	case "ctrl+d":
		// vim: half page down the DOCUMENT (browse could only scroll by
		// thread-hopping before — docs with few comments were unscrollable)
		m.documentViewport.SetYOffset(m.documentViewport.YOffset() + m.documentViewport.Height()/2)
		return m, nil

	case "ctrl+u":
		m.documentViewport.SetYOffset(max(m.documentViewport.YOffset()-m.documentViewport.Height()/2, 0))
		return m, nil

	case "j", "down":
		// Navigate comments
		visibleComments := m.visibleComments()
		m.clampSelectedComment(len(visibleComments))
		if m.selectedComment < len(visibleComments)-1 {
			m.selectedComment++
			m.scrollSidebarToSelected()
			// Scroll document to center the selected comment and move the
			// focus-line highlight with it (sidebar->doc sync)
			m.scrollToComment(visibleComments[m.selectedComment])
			m.refreshDocumentPane()
		}
		return m, nil

	case "k", "up":
		visibleComments := m.visibleComments()
		m.clampSelectedComment(len(visibleComments))
		if m.selectedComment > 0 {
			m.selectedComment--
			m.scrollSidebarToSelected()
			// Scroll document to center the selected comment and move the
			// focus-line highlight with it (sidebar->doc sync)
			m.scrollToComment(visibleComments[m.selectedComment])
			m.refreshDocumentPane()
		}
		return m, nil

	case "enter":
		// Expand selected comment thread
		visibleComments := m.visibleComments()
		if len(visibleComments) > 0 && m.selectedComment < len(visibleComments) {
			selectedThread := visibleComments[m.selectedComment]
			m.selectedThread = selectedThread
			m.mode = ModeThreadView
			// Scroll document to center the thread's comment, then size the
			// thread panel against that scroll position (keys_threadpanel.go)
			m.scrollToComment(selectedThread)
			m.refreshDocumentPane()
			m.applyThreadPanel()
			return m, nil
		}
		return m, nil

	case "P":
		// Walkthrough order: sidebar sorts what stops you first — blocking
		// threads, then priority-high (the agent's pivotal decisions/asks) —
		// and back to document order on second press.
		// Selection follows the same clamp rule as R: the visible ORDER
		// changed under it.
		m.sortByPriority = !m.sortByPriority
		m.selectedComment = 0
		m.scrollSidebarToSelected()
		return m, nil

	case "R":
		// Toggle showing resolved comments; the visible set just changed, so
		// the selection index from the old set must be clamped or the next
		// j/k indexes past the shorter list (found by live panic: R with the
		// selection deep in a mostly-resolved doc, then k)
		m.showResolved = !m.showResolved
		m.clampSelectedComment(len(m.visibleComments()))
		m.scrollSidebarToSelected()
		return m, nil
	}

	return m, nil
}

// clampSelectedComment keeps the sidebar selection inside the visible set —
// call it whenever the visible-comments filter changes under a live selection.
func (m *Model) clampSelectedComment(visible int) {
	if m.selectedComment >= visible {
		m.selectedComment = max(visible-1, 0)
	}
	if m.selectedComment < 0 {
		m.selectedComment = 0
	}
}

// scrollSidebarToSelected re-renders the comment sidebar and scrolls just
// enough to keep the selected thread visible — browse-mode selection moves
// (j/k/g/G) used to re-render without scrolling, so the highlight walked off
// the bottom of the sidebar once threads overflowed it. Works at any density
// (row-based, not ▼-anchored like refreshSidebar's line-select scroll).
func (m *Model) scrollSidebarToSelected() {
	content, selStart, selEnd := m.renderCommentsAnchored()
	m.commentViewport.SetContent(content)

	top := m.commentViewport.YOffset()
	height := m.commentViewport.Height()
	maxOffset := max(strings.Count(content, "\n")+1-height, 0)
	if selStart < top {
		// One row of context above (the group header when first in group)
		m.commentViewport.SetYOffset(max(selStart-1, 0))
	} else if selEnd-1 > top+height-1 {
		// Bring the block's end into view, but never push its start off
		m.commentViewport.SetYOffset(min(min(max(selEnd-height, 0), selStart), maxOffset))
	}
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

// titleBar renders the `📄 path - MODE` bar. Long paths are truncated
// (rune-safe) so the mode indicator stays visible — the v2 renderer clips
// rows at the terminal width, which used to push the mode suffix off screen.
func (m Model) titleBar(modeStr string) string {
	prefix := "📄 "
	suffix := " - " + modeStr
	name := m.filename
	if avail := m.width - lipgloss.Width(prefix) - lipgloss.Width(suffix); m.width > 0 && lipgloss.Width(name) > avail {
		name = truncate(name, max(avail, 1), "…")
	}
	return m.styles.title.Render(prefix + name + suffix)
}

// viewBrowse renders the browse/line-select view
func (m Model) viewBrowse() string {
	if !m.ready {
		return "Loading..."
	}

	title := m.titleBar(m.mode.String())

	var helpText string
	if m.mode == ModeLineSelect {
		// Peek discoverability: when the cursor line carries several
		// references, surface the f/Tab cycle in the hint bar
		followHint := "f: follow ref"
		if n := len(m.refsByLine[m.selectedLine]); n > 1 {
			followHint = fmt.Sprintf("f/Tab: cycle %d refs", n)
		}
		helpText = "j/k: move • r: open thread • " + followHint + " • n/N: next/prev NEW • Tab: cycle threads • c: comment • s: suggest • t: TOC • ?: help • Esc: cancel"
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
		m.renderRail(m.width),
		content,
		help,
	)
}
