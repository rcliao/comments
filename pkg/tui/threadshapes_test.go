package tui

// PROTOTYPE tests — Phase 2 of docs/plan-tui-in-context.md. Frame assertions
// for the three thread-display shapes: every shape must show document text
// AND thread text in the same frame, D cycles shapes live, Esc returns to
// where the thread was opened with the cursor intact. Deleted together with
// the losing shapes.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"

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

// shapeTestModel is a laid-out 100x40 model over a real temp-dir document
// (so save-on-reply works) with one thread at line 5 carrying a reply.
func shapeTestModel(t *testing.T) Model {
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

func TestThreadShapesShowDocAndThreadInSameFrame(t *testing.T) {
	m := openThreadAtLine5(t, shapeTestModel(t))

	for i, shape := range []string{"floating", "panel", "drawer"} {
		if i > 0 {
			m = drive(t, m, keyMsg("D"))
		}
		got := frame(m)
		// Document text still on screen (line 5, never covered by any shape:
		// floating opens below it, panel keeps the doc pane, drawer scrolls
		// it above the band)
		if !strings.Contains(got, "Alpha body text.") {
			t.Errorf("%s: document text missing from the frame:\n%s", shape, got)
		}
		// Thread chrome and content in the SAME frame
		if !strings.Contains(got, "Thread at Line 5") {
			t.Errorf("%s: thread header missing from the frame:\n%s", shape, got)
		}
		if !strings.Contains(got, "["+shape+"]") {
			t.Errorf("%s: shape name missing from the thread header:\n%s", shape, got)
		}
		if !strings.Contains(got, "reworded in the next pass") {
			t.Errorf("%s: thread reply missing from the frame:\n%s", shape, got)
		}
	}
}

func TestThreadShapeCycleKeyAdvancesAndWraps(t *testing.T) {
	m := openThreadAtLine5(t, shapeTestModel(t))
	if m.threadShape != shapeFloating {
		t.Fatalf("default shape should be floating, got %v", m.threadShape)
	}
	want := []threadShape{shapePanel, shapeDrawer, shapeFloating}
	for _, w := range want {
		m = drive(t, m, keyMsg("D"))
		if m.threadShape != w {
			t.Fatalf("D should cycle to %v, got %v", w, m.threadShape)
		}
		if m.mode != ModeThreadView {
			t.Fatalf("D must stay in thread view, got %v", m.mode)
		}
	}
}

func TestThreadShapeEscReturnsToLineSelectWithCursor(t *testing.T) {
	for presses, shape := range []string{"floating", "panel", "drawer"} {
		m := openThreadAtLine5(t, shapeTestModel(t))
		for range presses {
			m = drive(t, m, keyMsg("D"))
		}
		m = drive(t, m, keyMsg("esc"))
		if m.mode != ModeLineSelect {
			t.Errorf("%s: esc should return to line-select, got %v", shape, m.mode)
		}
		if m.selectedLine != 5 {
			t.Errorf("%s: cursor should still be on line 5 after esc, got %d", shape, m.selectedLine)
		}
		if m.selectedThread != nil {
			t.Errorf("%s: esc should clear the selected thread", shape)
		}
	}
}

func TestThreadShapeEscReturnsToBrowse(t *testing.T) {
	// Opened from browse (enter on the sidebar) rather than line-select
	m := drive(t, shapeTestModel(t), keyMsg("enter"))
	if m.mode != ModeThreadView {
		t.Fatalf("enter should open the thread from browse, got %v", m.mode)
	}
	got := frame(m)
	if !strings.Contains(got, "Alpha body text.") || !strings.Contains(got, "[floating]") {
		t.Errorf("browse-opened thread should composite over the document:\n%s", got)
	}
	m = drive(t, m, keyMsg("esc"))
	if m.mode != ModeBrowse {
		t.Errorf("esc should return to browse, got %v", m.mode)
	}
}

func TestFloatingFallsBackToDrawerOnSmallTerminal(t *testing.T) {
	m := drive(t, shapeTestModel(t), tea.WindowSizeMsg{Width: 60, Height: 14})
	m = openThreadAtLine5(t, m)

	if m.threadShape != shapeFloating {
		t.Fatalf("requested shape should stay floating, got %v", m.threadShape)
	}
	if got := m.effectiveThreadShape(); got != shapeDrawer {
		t.Fatalf("floating should fall back to drawer below %dx%d, got %v",
			minFloatingWidth, minFloatingHeight, got)
	}
	got := frame(m)
	if !strings.Contains(got, "[drawer]") {
		t.Errorf("fallback frame should carry the effective shape name:\n%s", got)
	}
	if !strings.Contains(got, "Alpha body text.") {
		t.Errorf("anchor line should stay visible above the fallback drawer:\n%s", got)
	}
}

func TestThreadShapeReplyRoundTripInFloating(t *testing.T) {
	m := openThreadAtLine5(t, shapeTestModel(t))

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
		t.Fatalf("reply not appended through the shape flow: %+v", replies)
	}
	if !strings.Contains(frame(m), "sounds good") {
		t.Error("new reply should be visible in the thread frame (newest activity in view)")
	}
}

func TestThreadShapeResolveRoundTripInFloating(t *testing.T) {
	m := openThreadAtLine5(t, shapeTestModel(t))
	m = drive(t, m, keyMsg("x"))
	if m.mode != ModeResolve {
		t.Fatalf("x on a regular thread should open resolve, got %v", m.mode)
	}
	m = drive(t, m, keyMsg("y"))
	if !m.doc.Threads[0].Resolved {
		t.Error("y should resolve the thread through the shape flow")
	}
}

// TestIntegrationThreadShapeCycle drives a REAL program (teatest) through
// open thread → cycle all three shapes → esc → quit. The v2 renderer diffs
// cells and only emits what changed between frames, so the shape-cycle waits
// assert on the newly drawn thread chrome only — the doc+thread-in-one-frame
// guarantee is covered per shape by TestThreadShapesShowDocAndThreadInSameFrame,
// which renders complete frames.
func TestIntegrationThreadShapeCycle(t *testing.T) {
	tm := teatest.NewTestModel(t, integrationModel(t), teatest.WithInitialTermSize(100, 40))
	waitForOutput(t, tm, "j/k: navigate")

	tm.Send(keyMsg("c"))
	waitForOutput(t, tm, "r: open thread")
	for range 4 {
		tm.Send(keyMsg("j"))
	}
	tm.Send(keyMsg("r"))
	waitForOutput(t, tm, "[floating]", "Thread at Line 5", "Alpha body text.")

	tm.Send(keyMsg("D"))
	waitForOutput(t, tm, "[panel]")
	tm.Send(keyMsg("D"))
	waitForOutput(t, tm, "[drawer]")
	tm.Send(keyMsg("D"))
	waitForOutput(t, tm, "[floating]")

	tm.Send(keyMsg("esc"))
	tm.Send(keyMsg("ctrl+c"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	fm, ok := tm.FinalModel(t).(Model)
	if !ok {
		t.Fatal("final model is not tui.Model")
	}
	if fm.mode != ModeLineSelect || fm.selectedLine != 5 {
		t.Errorf("esc should return to line-select at line 5, got %v line %d", fm.mode, fm.selectedLine)
	}
	if fm.threadShape != shapeFloating {
		t.Errorf("full D cycle should wrap back to floating, got %v", fm.threadShape)
	}
}
