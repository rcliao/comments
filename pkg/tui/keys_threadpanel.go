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
	"strings"

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

// composerTextareaRows is how many text rows the in-panel reply composer gets.
// The composer docks inside the panel rather than floating over the middle of
// the screen, so the thread you are replying to stays beside your reply.
const composerTextareaRows = 4

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

// composerRows is the vertical space the docked reply composer takes inside
// the panel: a separator rule plus the textarea's own rows. Zero when not
// composing. Derived from the textarea so the two cannot drift.
func (m Model) composerRows() int {
	if m.mode != ModeReply {
		return 0
	}
	return 1 + m.commentInput.Height()
}

// threadPaneRows is the height left for the threadViewport inside the panel
// once chrome and (when composing) the composer have taken their rows.
func (m Model) threadPaneRows(lay panelLayout) int {
	return max(lay.h-panelChromeRows-m.composerRows(), 1)
}

// applyComposerLayout sizes the reply composer to the panel and gives the
// thread viewport the rows that are left. Scroll position is preserved (the
// viewport clamps its own offset), so entering reply keeps your place in the
// thread. Call it on reply open, on resize while replying, and after leaving
// reply mode to hand the rows back.
func (m *Model) applyComposerLayout() {
	if !m.ready || m.selectedThread == nil {
		return
	}
	lay := m.threadPanelLayout()
	m.commentInput.SetHeight(composerTextareaRows)
	// -4: two border columns and the two columns lipgloss v2 counts inside
	// the panel's Width() (see renderThreadPanelBox)
	m.commentInput.SetWidth(max(lay.w-4, 1))
	m.threadViewport.SetHeight(m.threadPaneRows(lay))
}

// closeComposer undoes applyComposerLayout: the composer's rows go back to the
// thread (scroll survives — the viewport clamps its own offset) and the shared
// textarea returns to the size the centered dialogs expect. Call it with the
// mode already off ModeReply.
func (m *Model) closeComposer() {
	m.commentInput.SetWidth(m.dialogTextareaWidth())
	if m.dialogInputRows > 0 {
		m.commentInput.SetHeight(m.dialogInputRows)
	}
	if !m.ready || m.selectedThread == nil {
		return
	}
	m.threadViewport.SetHeight(m.threadPaneRows(m.threadPanelLayout()))
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
	m.threadViewport.SetHeight(m.threadPaneRows(lay))
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
// and a hint line, inside a rounded border. This is the thread's ONE header —
// renderThreadWidth draws no location line of its own.
func (m Model) renderThreadPanelBox(lay panelLayout) string {
	inner := lay.w - 2
	// lipgloss v2 Width() counts the border, so the text inside the box is
	// two columns narrower than the box width — truncate/rule to that or the
	// last characters wrap onto their own line.
	content := max(inner-2, 1)

	icon := threadTypeIcon(m.selectedThread)
	locationStr := fmt.Sprintf("Line %d", m.threadAnchorLine())
	if m.selectedThread.SectionPath != "" {
		locationStr = fmt.Sprintf("%s (Line %d)", m.selectedThread.SectionPath, m.threadAnchorLine())
	}
	headerText := fmt.Sprintf("%s Thread at %s", icon, locationStr)

	actionText := "x: resolve"
	if m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
		actionText = "a/x: queue accept/reject"
	}
	helpText := fmt.Sprintf("r: reply • %s • c: comment • Esc: close", actionText)

	header := m.styles.title.Render(truncate(headerText, content, "…"))

	// Composing: the reply docks below the thread inside this panel, and the
	// panel keys it replaces (r/x/c) are dead until it closes, so its help
	// line takes over the panel's.
	rows := []string{header, m.threadViewport.View()}
	if m.mode == ModeReply {
		rows = append(rows,
			m.styles.help.Render(strings.Repeat("─", content)),
			m.commentInput.View(),
		)
		helpText = "Ctrl+S: save reply • Esc: cancel"
	}
	rows = append(rows, m.styles.help.Render(truncate(helpText, content, "…")))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.theme.Title.Color()).
		Width(inner).
		Height(lay.h - 2).
		MaxWidth(lay.w).
		MaxHeight(lay.h).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
