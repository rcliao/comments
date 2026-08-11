package tui

// Frame assertions for the thread panel (the thread display picked in the
// Phase 2 live review): opening a thread must show document text AND thread
// text in the same frame, Esc returns to where the thread was opened with the
// cursor intact, and reply/resolve round-trip through the panel.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/rcliao/comments/pkg/comment"
)

// drive runs messages through the real Update loop and returns the new model.
func drive(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	var tm tea.Model = m
	for _, msg := range msgs {
		tm, _ = tm.Update(msg)
	}
	out, ok := tm.(Model)
	if !ok {
		t.Fatal("Update returned a non-tui.Model")
	}
	return out
}

// panelTestModel is a laid-out 100x40 model over a real temp-dir document
// (so save-on-reply works) with one thread at line 5 carrying a reply.
func panelTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(docPath, []byte(tuiTestDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := &comment.DocumentWithComments{
		Content: tuiTestDoc,
		Threads: []*comment.Comment{
			{
				ID: "c1", Line: 5, Author: "rcliao", Text: "tighten this paragraph",
				Replies: []*comment.Comment{
					{ID: "r1", Line: 5, Author: "claude", Text: "reworded in the next pass"},
				},
			},
		},
	}
	m := NewModelWithFile(doc, docPath)
	return drive(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
}

// openThreadAtLine5 walks line-select to the commented line and opens the
// thread through the real key route (c, 4x j, r).
func openThreadAtLine5(t *testing.T, m Model) Model {
	t.Helper()
	m = drive(t, m, keyMsg("c"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("r"))
	if m.mode != ModeThreadView {
		t.Fatalf("expected ModeThreadView after r on a commented line, got %v", m.mode)
	}
	if m.selectedThread == nil || m.selectedThread.ID != "c1" {
		t.Fatalf("expected thread c1 selected, got %+v", m.selectedThread)
	}
	return m
}

// frame renders the current screen with terminal escapes stripped.
func frame(m Model) string {
	return termEscapes.ReplaceAllString(m.viewContent(), "")
}

func TestThreadPanelShowsDocAndThreadInSameFrame(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))

	got := frame(m)
	// Document text still on screen: the panel takes the sidebar region, the
	// document pane is untouched
	if !strings.Contains(got, "Alpha body text.") {
		t.Errorf("document text missing from the frame:\n%s", got)
	}
	// Thread chrome and content in the SAME frame
	if !strings.Contains(got, "Thread at Line 5") {
		t.Errorf("thread header missing from the frame:\n%s", got)
	}
	if !strings.Contains(got, "reworded in the next pass") {
		t.Errorf("thread reply missing from the frame:\n%s", got)
	}
}

func TestThreadPanelWithHiddenSidebarTakesRightForty(t *testing.T) {
	m := panelTestModel(t)
	m = drive(t, m, keyMsg("S"), keyMsg("S")) // full → condensed → hidden
	if m.sidebarDensity != densityHidden {
		t.Fatalf("two S presses should hide the sidebar, got %d", m.sidebarDensity)
	}

	m = openThreadAtLine5(t, m)
	lay := m.threadPanelLayout()
	if want := m.width * 2 / 5; lay.w != want {
		t.Errorf("hidden sidebar: panel width should be 40%% (%d), got %d", want, lay.w)
	}
	got := frame(m)
	if !strings.Contains(got, "Alpha body text.") || !strings.Contains(got, "Thread at Line 5") {
		t.Errorf("hidden-sidebar panel should still show doc and thread in one frame:\n%s", got)
	}
}

func TestThreadPanelEscReturnsToLineSelectWithCursor(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))
	m = drive(t, m, keyMsg("esc"))
	if m.mode != ModeLineSelect {
		t.Errorf("esc should return to line-select, got %v", m.mode)
	}
	if m.selectedLine != 5 {
		t.Errorf("cursor should still be on line 5 after esc, got %d", m.selectedLine)
	}
	if m.selectedThread != nil {
		t.Error("esc should clear the selected thread")
	}
}

func TestThreadPanelEscReturnsToBrowse(t *testing.T) {
	// Opened from browse (enter on the sidebar) rather than line-select
	m := drive(t, panelTestModel(t), keyMsg("enter"))
	if m.mode != ModeThreadView {
		t.Fatalf("enter should open the thread from browse, got %v", m.mode)
	}
	got := frame(m)
	if !strings.Contains(got, "Alpha body text.") || !strings.Contains(got, "Thread at Line 5") {
		t.Errorf("browse-opened thread should composite over the document:\n%s", got)
	}
	m = drive(t, m, keyMsg("esc"))
	if m.mode != ModeBrowse {
		t.Errorf("esc should return to browse, got %v", m.mode)
	}
}

func TestThreadPanelReplyRoundTrip(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))

	m = drive(t, m, keyMsg("r"))
	if m.mode != ModeReply {
		t.Fatalf("r should open the reply dialog, got %v", m.mode)
	}
	for _, ch := range "sounds good" {
		m = drive(t, m, keyMsg(string(ch)))
	}
	m = drive(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	if m.mode != ModeThreadView {
		t.Fatalf("ctrl+s should land back in the thread view, got %v", m.mode)
	}
	replies := m.doc.Threads[0].Replies
	if len(replies) != 2 || replies[1].Text != "sounds good" {
		t.Fatalf("reply not appended through the panel flow: %+v", replies)
	}
	if !strings.Contains(frame(m), "sounds good") {
		t.Error("new reply should be visible in the thread frame (newest activity in view)")
	}
}

// The reply composer docks inside the panel instead of floating over the
// middle of the screen: one frame carries the document, the thread it replies
// to, and the composer — and the panel still draws exactly one header.
func TestThreadPanelReplyDocksInPanel(t *testing.T) {
	m := drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("r"))
	got := frame(m)

	for _, want := range []string{"# Title", "reworded in the next pass", "Enter your reply...", "Ctrl+S: save reply"} {
		if !strings.Contains(got, want) {
			t.Errorf("composing frame missing %q (doc + thread + composer must share the frame):\n%s", want, got)
		}
	}
	if n := strings.Count(got, "Thread at "); n != 1 {
		t.Errorf("composing must not add a second header, found %d:\n%s", n, got)
	}

	// The composer takes its rows from the thread viewport, so the panel box
	// keeps its height rather than spilling past the screen.
	lay := m.threadPanelLayout()
	if h := lipgloss.Height(m.renderThreadPanelBox(lay)); h != lay.h {
		t.Errorf("panel height with composer = %d, want %d (composer must take rows from the thread)", h, lay.h)
	}
	if w := lipgloss.Width(m.renderThreadPanelBox(lay)); w > lay.w {
		t.Errorf("panel width with composer = %d, want <= %d (nothing may wrap past the box)", w, lay.w)
	}
}

// Esc and Ctrl+S both hand the composer's rows back to the thread.
func TestThreadPanelComposerReleasesRowsOnClose(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))
	full := m.threadViewport.Height()

	m = drive(t, m, keyMsg("r"))
	if got := m.threadViewport.Height(); got >= full {
		t.Fatalf("thread viewport should shrink for the composer: %d -> %d", full, got)
	}
	m = drive(t, m, keyMsg("esc"))
	if got := m.threadViewport.Height(); got != full {
		t.Errorf("esc should restore the thread viewport to %d rows, got %d", full, got)
	}
}

// The reply composer is its own textarea, so the reply flow must leave the
// add-comment textarea completely untouched — no borrowed width or height to
// hand back, nothing to leak.
func TestReplyFlowNeverTouchesCommentInput(t *testing.T) {
	base := openThreadAtLine5(t, panelTestModel(t))
	wantW, wantH := base.commentInput.Width(), base.commentInput.Height()

	for _, tc := range []struct {
		name string
		exit []tea.Msg
	}{
		{"esc", []tea.Msg{keyMsg("esc")}},
		{"empty save", []tea.Msg{tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}}},
		{"save", []tea.Msg{keyMsg("o"), keyMsg("k"), tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("r"))
			m = drive(t, m, keyMsg("x"), keyMsg("y")) // typing lands in the composer
			if m.commentInput.Value() != "" {
				t.Errorf("reply text leaked into the add-comment textarea: %q", m.commentInput.Value())
			}
			m = drive(t, m, tc.exit...)
			if got := m.commentInput.Width(); got != wantW {
				t.Errorf("commentInput width after %s = %d, want %d (untouched)", tc.name, got, wantW)
			}
			if got := m.commentInput.Height(); got != wantH {
				t.Errorf("commentInput height after %s = %d, want %d (untouched)", tc.name, got, wantH)
			}
		})
	}
}

// The composer grows as you type and the thread gives up exactly those rows,
// so the panel box never changes size.
func TestComposerGrowsWithContent(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))
	fullThread := m.threadViewport.Height() // no composer docked yet
	m = drive(t, m, keyMsg("r"))
	lay := m.threadPanelLayout()
	startRows, startThread := m.replyInput.Height(), m.threadViewport.Height()
	if startRows != composerMinRows {
		t.Fatalf("composer should rest at %d rows, got %d", composerMinRows, startRows)
	}

	// Enough newlines to push past the resting height
	for range composerMinRows + 3 {
		m = drive(t, m, keyMsg("a"), tea.KeyPressMsg{Code: tea.KeyEnter})
	}

	grownRows, grownThread := m.replyInput.Height(), m.threadViewport.Height()
	if grownRows <= startRows {
		t.Errorf("composer should grow with the text: %d -> %d", startRows, grownRows)
	}
	if grownThread >= startThread {
		t.Errorf("thread pane should give up the rows the composer took: %d -> %d", startThread, grownThread)
	}
	if got := startRows - grownRows + startThread - grownThread; got != 0 {
		t.Errorf("rows must move between composer and thread, not appear: delta %d", got)
	}
	if h := lipgloss.Height(m.renderThreadPanelBox(lay)); h != lay.h {
		t.Errorf("panel height with a grown composer = %d, want %d", h, lay.h)
	}
	if !strings.Contains(frame(m), "Ctrl+S: save reply") {
		t.Error("composer help line should survive growth")
	}

	// Esc shrinks it back: Reset() recalculates the height, and the thread
	// reclaims every row — the composer's and the separator's
	m = drive(t, m, keyMsg("esc"))
	if got := m.replyInput.Height(); got != composerMinRows {
		t.Errorf("composer should reset to %d rows on cancel, got %d", composerMinRows, got)
	}
	if got := m.threadViewport.Height(); got != fullThread {
		t.Errorf("thread pane should be back to its full %d rows, got %d", fullThread, got)
	}
}

// Growth stops before it eats the thread: past the cap the composer scrolls
// internally, and long text stays typable (MaxContentHeight, not MaxHeight,
// is what bounds content).
func TestComposerGrowthCapLeavesThreadVisible(t *testing.T) {
	m := drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("r"))
	lay := m.threadPanelLayout()

	for range 60 {
		m = drive(t, m, keyMsg("z"), tea.KeyPressMsg{Code: tea.KeyEnter})
	}

	if got := m.threadViewport.Height(); got < composerMinThreadRows {
		t.Errorf("thread must keep at least %d rows visible, got %d", composerMinThreadRows, got)
	}
	if h := lipgloss.Height(m.renderThreadPanelBox(lay)); h != lay.h {
		t.Errorf("panel height past the growth cap = %d, want %d", h, lay.h)
	}
	// The text past the cap is still there — the composer scrolls, it does not
	// refuse input
	if n := m.replyInput.LineCount(); n < 60 {
		t.Errorf("long replies must stay typable past the visible cap, got %d lines", n)
	}
	if !strings.Contains(frame(m), "reworded in the next pass") {
		t.Error("thread content should still be visible beside a long reply")
	}
}

// A bracketed paste is a NON-key message: it reaches the composer through the
// registry's updateViewport, not the key handler, and can add many rows at
// once. The thread pane has to re-fit there too or the panel overflows.
func TestComposerGrowsOnPaste(t *testing.T) {
	m := drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("r"))
	lay := m.threadPanelLayout()
	startRows := m.replyInput.Height()

	m = drive(t, m, tea.PasteMsg{Content: "one\ntwo\nthree\nfour\nfive\nsix\nseven"})

	if got := m.replyInput.Height(); got <= startRows {
		t.Errorf("paste should grow the composer: %d -> %d", startRows, got)
	}
	// The pane must have re-fitted around the grown composer. Asserting the
	// rendered height would not catch this: the box clips at MaxHeight, so an
	// unsynced pane silently swallows thread rows instead of overflowing.
	if got, want := m.threadViewport.Height(), m.threadPaneRows(lay); got != want {
		t.Errorf("thread pane height after paste = %d, want %d (paste path must re-fit the pane)", got, want)
	}
	if h := lipgloss.Height(m.renderThreadPanelBox(lay)); h != lay.h {
		t.Errorf("panel height after paste = %d, want %d", h, lay.h)
	}
	if !strings.Contains(m.replyInput.Value(), "seven") {
		t.Errorf("pasted text missing from the composer: %q", m.replyInput.Value())
	}
}

// A short terminal must not produce a negative-height viewport or a box that
// outgrows the screen.
func TestThreadPanelComposerOnShortTerminal(t *testing.T) {
	m := drive(t, openThreadAtLine5(t, panelTestModel(t)),
		tea.WindowSizeMsg{Width: 80, Height: 10}, keyMsg("r"))

	if got := m.threadViewport.Height(); got < 1 {
		t.Errorf("thread viewport height must stay >= 1 on a short terminal, got %d", got)
	}
	lay := m.threadPanelLayout()
	if h := lipgloss.Height(m.renderThreadPanelBox(lay)); h > m.height {
		t.Errorf("panel height %d overflows the %d-row screen", h, m.height)
	}
}

func TestThreadPanelSingleHeader(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))
	got := frame(m)
	if n := strings.Count(got, "Thread at "); n != 1 {
		t.Errorf("panel must draw exactly one thread header, found %d:\n%s", n, got)
	}
	if strings.Contains(got, "Document Context:") {
		t.Errorf("panel must not re-print document context — the doc beside it IS the context:\n%s", got)
	}
}

func TestThreadPanelCFallThroughStartsCommentFlow(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))

	// c closes the panel and lands in the comment flow at the cursor line
	m = drive(t, m, keyMsg("c"))
	if m.mode != ModeLineSelect {
		t.Fatalf("c with the panel open should fall through to line-select, got %v", m.mode)
	}
	if m.selectedThread != nil {
		t.Error("c should close the panel (selected thread cleared)")
	}
	if m.selectedLine != 5 {
		t.Errorf("comment flow should start at the cursor line 5, got %d", m.selectedLine)
	}

	// and the flow continues as usual: c on the (non-heading) cursor line
	// opens the add-comment popup
	m = drive(t, m, keyMsg("c"))
	if m.mode != ModeAddComment {
		t.Fatalf("second c should open add-comment, got %v", m.mode)
	}
	if got := frame(m); !strings.Contains(got, "Add Comment at Line 5") {
		t.Errorf("add-comment popup should target line 5:\n%s", got)
	}
}

func TestThreadPanelQFallThroughOpensVerdict(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))

	m = drive(t, m, keyMsg("q"))
	if m.mode != ModeVerdict {
		t.Fatalf("q with the panel open should open the verdict dialog, got %v", m.mode)
	}
	got := frame(m)
	if !strings.Contains(got, "Submit review for") || !strings.Contains(got, "Thread at Line 5") {
		t.Errorf("verdict popup should layer over the doc+panel view:\n%s", got)
	}

	// Esc returns to the open panel, not to browse
	m = drive(t, m, keyMsg("esc"))
	if m.mode != ModeThreadView || m.selectedThread == nil {
		t.Errorf("esc from verdict should restore the open panel, got %v", m.mode)
	}
}

func TestThreadPanelHelpFallThrough(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))

	m = drive(t, m, keyMsg("?"))
	if m.mode != ModeHelp {
		t.Fatalf("? with the panel open should open help, got %v", m.mode)
	}
	m = drive(t, m, keyMsg("z"))
	if m.mode != ModeThreadView || m.selectedThread == nil {
		t.Errorf("closing help should restore the open panel, got %v", m.mode)
	}
}

func TestThreadPanelResolveRoundTrip(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))
	m = drive(t, m, keyMsg("x"))
	if m.mode != ModeResolve {
		t.Fatalf("x on a regular thread should open resolve, got %v", m.mode)
	}
	m = drive(t, m, keyMsg("y"))
	if !m.doc.Threads[0].Resolved {
		t.Error("y should resolve the thread through the panel flow")
	}
}

// Sidebar->doc sync (live-review request): navigating threads in the sidebar
// highlights the focused thread's line in the document pane, and the open
// panel highlights its anchor line.
func TestSidebarFocusHighlightsDocLine(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 3, Text: "first", Author: "a"},
		{ID: "c2", Line: 5, Text: "second", Author: "b"},
	})
	m.width, m.height = 120, 40
	m.handleResize()
	m.mode = ModeBrowse
	m.selectedComment = 0
	m.refreshDocumentPane()

	frame1 := m.renderDocument()
	if !lineHasFocusBg(frame1, 3) || lineHasFocusBg(frame1, 5) {
		t.Errorf("focus should sit on line 3 only:\n%s", frame1)
	}

	// j moves sidebar selection to the second thread -> focus moves to line 5
	next, _ := m.handleBrowseKeys(keyMsg("j"))
	nm := next.(Model)
	frame2 := nm.renderDocument()
	if lineHasFocusBg(frame2, 3) || !lineHasFocusBg(frame2, 5) {
		t.Errorf("after j, focus should move to line 5:\n%s", frame2)
	}

	// opening the panel keeps the anchor line highlighted
	opened, _ := nm.handleBrowseKeys(keyMsg("enter"))
	om := opened.(Model)
	if om.mode != ModeThreadView {
		t.Fatal("enter should open the thread panel")
	}
	if !lineHasFocusBg(om.renderDocument(), 5) {
		t.Errorf("panel open should highlight its anchor line 5")
	}
}

// lineHasFocusBg reports whether the rendered doc line numbered n carries the
// cursor/focus background style. The gutter is marker-then-number, so the
// line number is the first all-digit field in the row.
func lineHasFocusBg(frame string, n int) bool {
	want := fmt.Sprintf("%d", n)
	for _, row := range strings.Split(frame, "\n") {
		for _, f := range strings.Fields(stripANSI(row)) {
			if f == want {
				return strings.Contains(row, "\x1b[48;") || strings.Contains(row, ";48;")
			}
			// stop at the first field that is pure digits but not ours —
			// that's a different line's number
			isNum := len(f) > 0
			for _, r := range f {
				if r < '0' || r > '9' {
					isNum = false
					break
				}
			}
			if isNum {
				break
			}
		}
	}
	return false
}

// # toggles the line-number column in both document renders; wrap width and
// scroll math follow the narrower gutter so nothing reflows inconsistently.
func TestHideLineNumbersToggle(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 3, Text: "note", Author: "a"}})
	m.width, m.height = 100, 40
	m.handleResize()
	m.mode = ModeBrowse

	if !strings.Contains(stripANSI(m.renderDocument()), " 3 ") {
		t.Fatal("line numbers should show by default")
	}

	next, _ := m.handleBrowseKeys(keyMsg("#"))
	nm := next.(Model)
	if !nm.hideLineNumbers {
		t.Fatal("# should hide line numbers")
	}
	frame := stripANSI(nm.renderDocument())
	for _, row := range strings.Split(frame, "\n") {
		fields := strings.Fields(row)
		if len(fields) > 0 && fields[0] == "3" {
			t.Errorf("hidden mode must not render the number column: %q", row)
		}
	}
	if nm.gutterWidth() != 3 {
		t.Errorf("hidden gutter should be 3 (marker cell only), got %d", nm.gutterWidth())
	}

	// toggle back in line-select too
	nm.mode = ModeLineSelect
	nm.selectedLine = 3
	back, _ := nm.handleLineSelectKeys(keyMsg("#"))
	bm := back.(Model)
	if bm.hideLineNumbers {
		t.Fatal("# in line-select should toggle numbers back on")
	}
	if !strings.Contains(stripANSI(bm.renderDocumentWithCursor()), " 3 ") {
		t.Error("numbers should be visible again after toggling back")
	}
}

// Regression for a live panic: R (hide resolved) shrinks the visible set
// while the sidebar selection is deep in it; the next k indexed past the
// shorter list (index out of range at keys_browse.go). The selection must
// clamp when the filter changes.
func TestResolvedToggleClampsSelection(t *testing.T) {
	// 4 threads, the last 3 resolved; selection on the last one
	threads := []*comment.Comment{
		{ID: "c1", Line: 3, Author: "a", Text: "open"},
		{ID: "c2", Line: 5, Author: "a", Text: "r1", Resolved: true},
		{ID: "c3", Line: 7, Author: "a", Text: "r2", Resolved: true},
		{ID: "c4", Line: 9, Author: "a", Text: "r3", Resolved: true},
	}
	m := *testModel(threads)
	m.width, m.height = 100, 40
	m.handleResize()
	m.showResolved = true
	m.selectedComment = 3

	// R hides resolved -> 1 visible; then k must not panic
	next, _ := m.handleBrowseKeys(keyMsg("R"))
	nm := next.(Model)
	if got := len(nm.visibleComments()); got != 1 {
		t.Fatalf("expected 1 visible after hiding resolved, got %d", got)
	}
	if nm.selectedComment != 0 {
		t.Errorf("selection should clamp to 0, got %d", nm.selectedComment)
	}
	next, _ = nm.handleBrowseKeys(keyMsg("k"))
	nm = next.(Model)
	next, _ = nm.handleBrowseKeys(keyMsg("j"))
	nm = next.(Model)
	if nm.selectedComment != 0 {
		t.Errorf("j/k on a 1-item list should stay at 0, got %d", nm.selectedComment)
	}
}

// Agents refer humans to threads by ID ("answer c7q39") — the ID must be
// findable in the TUI: panel header and sidebar rows.
func TestThreadIDVisibleInPanelAndSidebar(t *testing.T) {
	m := openThreadAtLine5(t, panelTestModel(t))
	if got := frame(m); !strings.Contains(got, "c1 · Thread at") {
		t.Errorf("panel header should lead with the thread ID:\n%s", got)
	}
	// Sidebar rows carry the ID in the dim meta tail in both densities
	// (content-first layout: text gets the width, @author · id trails)
	if got := stripANSI(m.renderComments()); !strings.Contains(got, "· c1") {
		t.Errorf("sidebar (full) should carry the thread ID in the meta tail:\n%s", got)
	}
	m.sidebarDensity = densityCondensed
	if got := stripANSI(m.renderComments()); !strings.Contains(got, "@rcliao · c1") {
		t.Errorf("sidebar (condensed) should end rows with @author · id:\n%s", got)
	}
}

// shift+enter inserts a newline like enter: enhanced-keyboard terminals
// deliver it as a distinct key the default binding would drop.
func TestShiftEnterInsertsNewlineInComposer(t *testing.T) {
	m := drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("r"))
	m = drive(t, m, keyMsg("a"),
		tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift},
		keyMsg("b"))
	if got := m.replyInput.Value(); got != "a\nb" {
		t.Errorf("shift+enter should insert a newline, got %q", got)
	}
}

// Reply headers are the scan lines between long replies: each carries a
// trailing rule, so the boundary between consecutive multi-paragraph replies
// is visible (live report: replies fused into one wall of text).
func TestReplyHeadersCarryScanRules(t *testing.T) {
	m := panelTestModel(t)
	// Add a second reply so there are two boundaries to see
	if err := comment.AddReplyToThread(m.doc.Threads, "c1", "rcliao", "a long human reply\nwith two lines"); err != nil {
		t.Fatal(err)
	}
	m = openThreadAtLine5(t, m)
	got := frame(m)
	rules := strings.Count(got, "· ")
	if rules < 2 {
		t.Fatalf("expected author headers for both replies, got:\n%s", got)
	}
	if !strings.Contains(got, "──────") {
		t.Errorf("reply headers should end in a scan rule:\n%s", got)
	}
}

// The number column is digit-fit: a 2-digit doc gets a 2-cell column — no
// ghost spaces between the comment marker and the number (live review).
func TestGutterNumberColumnDigitFit(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 3, Text: "n", Author: "a", Blocking: true}})
	m.width, m.height = 100, 40
	m.handleResize()
	if got := m.lineNumWidth(); got != 2 {
		t.Fatalf("10-line doc should get a 2-cell number column, got %d", got)
	}
	if got := m.gutterWidth(); got != 6 {
		t.Fatalf("gutter should be 3+2+1=6, got %d", got)
	}
	row := ""
	for _, r := range strings.Split(stripANSI(m.renderDocument()), "\n") {
		if strings.Contains(r, "⛔1") {
			row = r
			break
		}
	}
	if row == "" || strings.Contains(row, "⛔1  3") {
		t.Errorf("marker should sit one cell from the number, got %q", row)
	}
	if !strings.Contains(row, "⛔1 3") {
		t.Errorf("expected snug marker+number, got %q", row)
	}
}
