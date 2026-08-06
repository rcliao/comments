package tui

// End-to-end integration tests driving a REAL Bubbletea program through
// Charm's teatest harness (Phase 1 of docs/plan-tui-in-context.md): open a
// document with comments, enter line-select, walk the cursor, dive into a
// thread, return, and quit — asserting on the frames the program actually
// emits, so the v2 port is verified end-to-end rather than unit-by-unit.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/rcliao/comments/pkg/comment"
)

// integrationModel builds a model over a real on-disk document with one
// commented thread (line 5) that carries a reply.
func integrationModel(t *testing.T) Model {
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
	return NewModelWithFile(doc, docPath)
}

// termEscapes matches CSI/OSC/simple terminal escape sequences. Bubbletea
// v2's renderer diffs cells and interleaves positioning sequences inside
// rows, so raw frame bytes can split any text run; assertions match against
// the escape-stripped stream instead.
var termEscapes = regexp.MustCompile(`\x1b(\[[0-9;?]*[ -/]*[@-~]|\][^\x07\x1b]*(\x07|\x1b\\)|[@-_])`)

// waitForOutput waits until the program output contains every want, with
// terminal escape sequences stripped. One call per program phase: teatest's
// WaitFor consumes the output stream, so markers that belong to the same
// frame must be checked together, and a later call only sees output emitted
// after the previous call returned.
func waitForOutput(t *testing.T, tm *teatest.TestModel, wants ...string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		plain := termEscapes.ReplaceAllString(string(bts), "")
		for _, want := range wants {
			if !strings.Contains(plain, want) {
				return false
			}
		}
		return true
	}, teatest.WithDuration(3*time.Second))
}

func TestIntegrationOpenNavigateThreadReturnQuit(t *testing.T) {
	tm := teatest.NewTestModel(t, integrationModel(t), teatest.WithInitialTermSize(100, 40))

	// Open: browse mode (its hint bar) shows the document and the sidebar
	// thread in the same frame. The title bar is no marker here: the v2
	// renderer clips rows at the terminal width, and the temp-dir path
	// pushes the "- BROWSE" suffix off screen.
	waitForOutput(t, tm, "j/k: navigate", "Alpha body text.", "tighten this paragraph")

	// Enter line-select (its hint bar) and walk the cursor to commented line 5
	tm.Send(keyMsg("c"))
	waitForOutput(t, tm, "r: open thread")
	for range 4 {
		tm.Send(keyMsg("j"))
	}

	// Dive into the thread: full thread view with root text and the reply
	tm.Send(keyMsg("r"))
	waitForOutput(t, tm, "Thread at Line 5", "reworded in the next pass")

	// Return to line-select, then quit silently
	tm.Send(keyMsg("esc"))
	tm.Send(keyMsg("ctrl+c"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// The final model proves the whole route: esc landed back in
	// line-select at the cursor line, and ctrl+c quit without a verdict
	fm, ok := tm.FinalModel(t).(Model)
	if !ok {
		t.Fatal("final model is not tui.Model")
	}
	if fm.mode != ModeLineSelect {
		t.Errorf("esc from thread view should return to line-select, got %v", fm.mode)
	}
	if fm.selectedLine != 5 {
		t.Errorf("cursor should still be on line 5 after the round trip, got %d", fm.selectedLine)
	}
	if fm.selectedThread != nil {
		t.Error("returning from thread view should clear the selected thread")
	}
	if fm.VerdictDecision != "" {
		t.Errorf("ctrl+c must quit without a verdict, got %q", fm.VerdictDecision)
	}
}

// TestIntegrationThreadReplyPopupOverLiveView drives a REAL program (teatest)
// through open thread → reply popup over the live doc+panel → esc esc → quit.
// The v2 renderer diffs cells between frames, so each wait asserts on the
// newly drawn chrome; full-frame doc+dialog guarantees are covered by
// TestDialogsShowDocumentInSameFrame, which renders complete frames.
func TestIntegrationThreadReplyPopupOverLiveView(t *testing.T) {
	tm := teatest.NewTestModel(t, integrationModel(t), teatest.WithInitialTermSize(100, 40))
	waitForOutput(t, tm, "j/k: navigate")

	tm.Send(keyMsg("c"))
	waitForOutput(t, tm, "r: open thread")
	for range 4 {
		tm.Send(keyMsg("j"))
	}
	tm.Send(keyMsg("r"))
	waitForOutput(t, tm, "Thread at Line 5", "reworded in the next pass")

	// r opens the reply popup composited over the live doc + thread panel
	tm.Send(keyMsg("r"))
	waitForOutput(t, tm, "Reply to Thread")

	// Esc pops the popup back to the panel; Esc again closes the panel
	tm.Send(keyMsg("esc"))
	tm.Send(keyMsg("esc"))
	tm.Send(keyMsg("ctrl+c"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	fm, ok := tm.FinalModel(t).(Model)
	if !ok {
		t.Fatal("final model is not tui.Model")
	}
	if fm.mode != ModeLineSelect || fm.selectedLine != 5 {
		t.Errorf("esc esc should land in line-select at line 5, got %v line %d", fm.mode, fm.selectedLine)
	}
	if fm.selectedThread != nil {
		t.Error("closing the panel should clear the selected thread")
	}
	if fm.VerdictDecision != "" {
		t.Errorf("ctrl+c must quit without a verdict, got %q", fm.VerdictDecision)
	}
}

func TestIntegrationVerdictApproveRecordsSignoff(t *testing.T) {
	tm := teatest.NewTestModel(t, integrationModel(t), teatest.WithInitialTermSize(100, 40))
	waitForOutput(t, tm, "j/k: navigate")

	// q opens the verdict dialog over the review; a approves and quits
	tm.Send(keyMsg("q"))
	waitForOutput(t, tm, "Submit review for")
	tm.Send(keyMsg("a"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	fm, ok := tm.FinalModel(t).(Model)
	if !ok {
		t.Fatal("final model is not tui.Model")
	}
	if fm.VerdictDecision != comment.DecisionApproved {
		t.Errorf("verdict a should approve, got %q", fm.VerdictDecision)
	}
	if len(fm.doc.Reviews) != 1 || fm.doc.Reviews[0].Decision != comment.DecisionApproved {
		t.Errorf("signoff not recorded through the real program: %v", fm.doc.Reviews)
	}
}

func TestIntegrationResizeReflowsLayout(t *testing.T) {
	tm := teatest.NewTestModel(t, integrationModel(t), teatest.WithInitialTermSize(120, 40))
	waitForOutput(t, tm, "Alpha body text.")

	// A live resize must not wedge the program: the next frame still renders
	tm.Send(tea.WindowSizeMsg{Width: 60, Height: 20})
	tm.Send(keyMsg("c"))
	waitForOutput(t, tm, "r: open thread")

	tm.Send(keyMsg("ctrl+c"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	fm := tm.FinalModel(t).(Model)
	if got := fm.documentViewport.Height(); got != 18 {
		t.Errorf("resize should reach the viewports (height 20-2=18), got %d", got)
	}
}
