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
	if nm.gutterWidth() != 8 {
		t.Errorf("hidden gutter should be 8, got %d", nm.gutterWidth())
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
