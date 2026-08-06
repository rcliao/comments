package tui

// Thread panel — THE thread display, picked in the Phase 2 live review of
// docs/plan-tui-in-context.md (floating overlay and bottom drawer were the
// losing prototypes; deleted wholesale). Opening a thread never swaps the
// screen: ModeThreadView composites the thread over the live
// browse/line-select view as a side-panel takeover — the thread replaces the
// comment-sidebar region (right of the document pane), full content height,
// document pane untouched. When the sidebar is hidden the panel still takes
// the right 40% of the screen.

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
)

// threadRenderPad is the margin renderThreadWidth reserves: its widest inner
// box renders at width-8 content plus 2 border columns (width-6 total), so
// passing innerWidth+threadRenderPad makes the thread content exactly fill
// the panel's inner width.
const threadRenderPad = 6

// panelChromeRows is the vertical chrome around the threadViewport inside the
// panel box: top+bottom border, header line, help line.
const panelChromeRows = 4

// panelLayout is the outer geometry of the thread panel on the screen.
type panelLayout struct {
	x, y, w, h int
}

// threadPanelLayout computes where the panel sits: the sidebar region (right
// of the document pane), or the right 40% when the sidebar is hidden. The
// screen rows are: title (row 0), content (rows 1..height-2), help (row
// height-1); the panel keeps title and help visible.
func (m *Model) threadPanelLayout() panelLayout {
	w := max(m.width-m.docPaneWidth(), m.width*2/5)
	w = min(w, m.width)
	return panelLayout{m.width - w, 1, w, max(m.height-2, 3)}
}

// threadAnchorLine is the document line the open thread anchors to.
func (m *Model) threadAnchorLine() int {
	if m.selectedThread == nil {
		return 1
	}
	return max(m.selectedThread.Line, 1)
}

// applyThreadPanel (re)sizes the threadViewport for the panel and fills it
// with the thread rendered at the panel's width. Call it wherever the thread
// content or geometry changes: thread open, resize, reply added, decision
// queued.
func (m *Model) applyThreadPanel() {
	if m.selectedThread == nil || !m.ready {
		return
	}
	lay := m.threadPanelLayout()
	m.threadViewport.SetWidth(lay.w - 2)
	m.threadViewport.SetHeight(max(lay.h-panelChromeRows, 1))
	m.threadViewport.SetContent(m.renderThreadWidth(lay.w - 2 + threadRenderPad))
	// When the thread doesn't fit the panel, land on the newest activity
	// (the bottom of the timeline); j/k still scroll back up
	if m.threadViewport.TotalLineCount() > m.threadViewport.Height() {
		m.threadViewport.GotoBottom()
	} else {
		m.threadViewport.GotoTop()
	}
}

// refreshThreadPane re-renders the open thread at the panel width, preserving
// the scroll position (unlike applyThreadPanel, which re-derives geometry and
// jumps to the newest activity).
func (m *Model) refreshThreadPane() {
	if m.selectedThread == nil {
		return
	}
	if !m.ready {
		// Unit tests drive handlers without a laid-out screen
		m.threadViewport.SetContent(m.renderThread())
		return
	}
	lay := m.threadPanelLayout()
	m.threadViewport.SetContent(m.renderThreadWidth(lay.w - 2 + threadRenderPad))
}

// viewThreadPanel renders ModeThreadView: the live browse/line-select view
// with the thread panel composited over the sidebar region.
func (m Model) viewThreadPanel() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}
	if !m.ready {
		return "Loading..."
	}
	lay := m.threadPanelLayout()
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(m.viewBrowse()).Z(0),
		lipgloss.NewLayer(m.renderThreadPanelBox(lay)).X(lay.x).Y(lay.y).Z(1),
	).Render()
}

// renderThreadPanelBox draws the panel chrome: header, the thread viewport,
// and a hint line, inside a rounded border.
func (m Model) renderThreadPanelBox(lay panelLayout) string {
	inner := lay.w - 2

	icon := "💬"
	if m.selectedThread.SectionPath != "" {
		icon = "📍"
	}
	headerText := fmt.Sprintf("%s Thread at Line %d", icon, m.threadAnchorLine())

	actionText := "x: resolve"
	if m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
		actionText = "a/x: queue accept/reject"
	}
	helpText := fmt.Sprintf("r: reply • %s • Esc: close", actionText)

	header := m.styles.title.Render(truncate(headerText, max(inner-1, 1), "…"))
	help := m.styles.help.Render(truncate(helpText, max(inner-1, 1), "…"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.theme.Title.Color()).
		Width(inner).
		Height(lay.h - 2).
		MaxWidth(lay.w).
		MaxHeight(lay.h).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, m.threadViewport.View(), help))
}
