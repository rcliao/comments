package tui

// PROTOTYPE — Phase 2 of docs/plan-tui-in-context.md: three thread-display
// shapes the human picks between by driving them. Opening a thread no longer
// swaps the screen; ModeThreadView renders the thread OVER the live
// browse/line-select view via the lipgloss v2 compositor, in one of three
// shapes cycled live with D:
//
//   floating — bordered overlay composited near the anchor line (below it if
//              there is room, above otherwise), ~70% of the document pane
//              wide, height capped with internal scroll. Falls back to the
//              drawer on terminals smaller than minFloatingWidth/Height.
//   panel    — side-panel takeover: the thread replaces the comment-sidebar
//              region (right ~40%), full height; document pane untouched.
//   drawer   — bottom drawer: the thread in a full-width band (~40% height);
//              the document — scrolled so the anchor line stays visible —
//              above it.
//
// All three render the SAME content: renderThreadWidth (the existing thread
// renderer, width-parameterized) inside the threadViewport, so scrolling and
// the a/x/r actions work identically. Everything shape-specific lives in this
// file so the two losing shapes can be deleted wholesale.

import (
	"fmt"

	lipgloss "charm.land/lipgloss/v2"
)

// threadShape enumerates the prototype thread-display shapes. The zero value
// (floating) is the default shape.
type threadShape int

const (
	shapeFloating threadShape = iota
	shapePanel
	shapeDrawer
	threadShapeCount // number of shapes; keep last for the D cycle
)

func (s threadShape) String() string {
	switch s {
	case shapeFloating:
		return "floating"
	case shapePanel:
		return "panel"
	case shapeDrawer:
		return "drawer"
	}
	return "unknown"
}

// Small-terminal fallback: the floating overlay needs room to sit near the
// anchor line without covering the whole document; below these dimensions it
// renders as the bottom drawer instead (the shape header shows the effective
// shape, so the fallback is visible).
const (
	minFloatingWidth  = 70
	minFloatingHeight = 16
)

// threadRenderPad is the margin renderThreadWidth reserves: its widest inner
// box renders at width-8 content plus 2 border columns (width-6 total), so
// passing innerWidth+threadRenderPad makes the thread content exactly fill a
// shape's inner width.
const threadRenderPad = 6

// shapeChromeRows is the vertical chrome around the threadViewport inside a
// shape box: top+bottom border, header line, help line.
const shapeChromeRows = 4

// shapeLayout is the outer geometry of the thread box on the screen.
type shapeLayout struct {
	shape      threadShape // effective shape (after small-terminal fallback)
	x, y, w, h int
}

// effectiveThreadShape applies the small-terminal fallback.
func (m *Model) effectiveThreadShape() threadShape {
	if m.threadShape == shapeFloating && (m.width < minFloatingWidth || m.height < minFloatingHeight) {
		return shapeDrawer
	}
	return m.threadShape
}

// threadShapeLayout computes where the current shape's box sits. Pure on the
// model — callable from View. The screen rows are: title (row 0), content
// (rows 1..height-2), help (row height-1); shapes keep title and help visible.
func (m *Model) threadShapeLayout() shapeLayout {
	switch m.effectiveThreadShape() {
	case shapePanel:
		// Cover the sidebar region (right of the document pane); when the
		// sidebar is hidden, still take the right 40% of the screen
		w := max(m.width-m.docPaneWidth(), m.width*2/5)
		w = min(w, m.width)
		return shapeLayout{shapePanel, m.width - w, 1, w, max(m.height-2, 3)}

	case shapeDrawer:
		h := max(m.height*2/5, 8)
		h = min(h, max(m.height-2, 3))
		y := max(m.height-1-h, 0)
		// Keep the anchor line visible above the band. Scrolling normally
		// handles this (ensureAnchorAboveDrawer), but a document shorter
		// than its viewport cannot scroll at all (SetYOffset clamps to 0) —
		// then the drawer shrinks instead, down to a 1-row thread window.
		anchorRow := 1 + m.calculateDisplayRow(m.threadAnchorLine()-1) - m.documentViewport.YOffset()
		if anchorRow >= y && anchorRow < m.height-2 {
			if ny := anchorRow + 1; m.height-1-ny >= shapeChromeRows+1 {
				y = ny
				h = m.height - 1 - y
			}
		}
		return shapeLayout{shapeDrawer, 0, y, m.width, h}
	}

	// Floating: sized to content (capped), anchored to the thread's line
	w := min(max(m.docPaneWidth()*7/10, 44), max(m.width-4, 20))
	rows := lipgloss.Height(m.renderThreadWidth(w - 2 + threadRenderPad))
	h := min(rows+shapeChromeRows, max(m.height-8, 8))

	// Screen row of the anchor line: title row + display row - scroll offset
	anchorRow := 1 + m.calculateDisplayRow(m.threadAnchorLine()-1) - m.documentViewport.YOffset()
	y := anchorRow + 1 // prefer just below the anchor line
	if y+h > m.height-1 {
		y = anchorRow - h // no room below: sit above
	}
	y = min(max(y, 1), max(m.height-1-h, 1))

	x := 4 // slight indent into the document pane
	if x+w > m.width {
		x = max(m.width-w, 0)
	}
	return shapeLayout{shapeFloating, x, y, w, h}
}

// threadAnchorLine is the document line the open thread anchors to.
func (m *Model) threadAnchorLine() int {
	if m.selectedThread == nil {
		return 1
	}
	return max(m.selectedThread.Line, 1)
}

// applyThreadShape (re)sizes the threadViewport for the current shape and
// fills it with the thread rendered at the shape's width. Call it wherever
// the thread content or geometry changes: thread open, D cycle, resize,
// reply added, decision queued.
func (m *Model) applyThreadShape() {
	if m.selectedThread == nil || !m.ready {
		return
	}
	lay := m.threadShapeLayout()
	if lay.shape == shapeDrawer {
		// Scroll the anchor above the band first, then re-derive the
		// geometry: the layout reads the scroll offset (and shrinks the
		// drawer when the document cannot scroll)
		m.ensureAnchorAboveDrawer(lay)
		lay = m.threadShapeLayout()
	}
	m.threadViewport.SetWidth(lay.w - 2)
	m.threadViewport.SetHeight(max(lay.h-shapeChromeRows, 1))
	m.threadViewport.SetContent(m.renderThreadWidth(lay.w - 2 + threadRenderPad))
	// When the thread doesn't fit the shape, land on the newest activity
	// (the bottom of the timeline); j/k still scroll back up
	if m.threadViewport.TotalLineCount() > m.threadViewport.Height() {
		m.threadViewport.GotoBottom()
	} else {
		m.threadViewport.GotoTop()
	}
}

// refreshThreadPane re-renders the open thread at its current shape width,
// preserving the scroll position (unlike applyThreadShape, which re-derives
// geometry and jumps to the newest activity).
func (m *Model) refreshThreadPane() {
	if m.selectedThread == nil {
		return
	}
	if !m.ready {
		// Unit tests drive handlers without a laid-out screen
		m.threadViewport.SetContent(m.renderThread())
		return
	}
	lay := m.threadShapeLayout()
	m.threadViewport.SetContent(m.renderThreadWidth(lay.w - 2 + threadRenderPad))
}

// ensureAnchorAboveDrawer scrolls the document so the anchor line stays
// visible in the strip above the drawer.
func (m *Model) ensureAnchorAboveDrawer(lay shapeLayout) {
	visibleRows := max(lay.y-1, 1) // document rows above the drawer
	row := m.calculateDisplayRow(m.threadAnchorLine() - 1)
	off := m.documentViewport.YOffset()
	if row-off >= visibleRows || row < off {
		m.documentViewport.SetYOffset(max(row-visibleRows/2, 0))
	}
}

// viewThreadShaped renders ModeThreadView: the live browse/line-select view
// with the thread box composited over (or beside) it in the current shape.
func (m Model) viewThreadShaped() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}
	if !m.ready {
		return "Loading..."
	}
	lay := m.threadShapeLayout()
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(m.viewBrowse()).Z(0),
		lipgloss.NewLayer(m.renderThreadShapeBox(lay)).X(lay.x).Y(lay.y).Z(1),
	).Render()
}

// renderThreadShapeBox draws the shape chrome: header (with the current shape
// name), the thread viewport, and a hint line, inside a rounded border.
func (m Model) renderThreadShapeBox(lay shapeLayout) string {
	inner := lay.w - 2

	icon := "💬"
	if m.selectedThread.SectionPath != "" {
		icon = "📍"
	}
	headerText := fmt.Sprintf("%s Thread at Line %d · [%s]", icon, m.threadAnchorLine(), lay.shape)
	if lay.shape != m.threadShape {
		headerText += " (small-terminal fallback)"
	}

	actionText := "x: resolve"
	if m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
		actionText = "a/x: queue accept/reject"
	}
	helpText := fmt.Sprintf("r: reply • %s • D: cycle shape • Esc: close", actionText)

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
