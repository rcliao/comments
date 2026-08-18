package tui

// Tests for the mode-descriptor registry and resize handling: every
// registered mode must route keys, non-key messages, and View() without
// panicking, and viewports must track the window size (including tiny
// terminals).

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/rcliao/comments/pkg/comment"
)

// allModes lists every ViewMode in the enum (keep in sync with modes.go)
var allModes = []ViewMode{
	ModeFilePicker, ModeBrowse, ModeLineSelect, ModeAddComment, ModeThreadView,
	ModeReply, ModeResolve, ModeAddSuggestion, ModeChooseTarget,
	ModeSelectSuggestionType, ModeSelectRange, ModeVerdict, ModeVerdictNote,
	ModeHelp, ModeTOC, ModeRefPeek, ModeSearch,
}

// dispatchModel builds a ready model in the given mode with enough state for
// any mode's handler and view: a loaded doc, a selected thread, the cursor on
// a heading line (choose-target derefs its section), and an active range.
func dispatchModel(t *testing.T, mode ViewMode) Model {
	t.Helper()
	var m Model
	if mode == ModeFilePicker {
		m = NewModel()
	} else {
		m = *testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "note", Author: "a"}})
	}
	m.width, m.height = 100, 40
	m.handleResize()
	if m.doc != nil {
		m.selectedThread = m.doc.Threads[0]
	}
	m.selectedLine = 1 // "# Title": a heading line, valid for every mode
	m.rangeStartLine, m.rangeEndLine = 1, 2
	m.verdictReturnMode = ModeBrowse
	m.helpReturnMode = ModeBrowse
	m.tocReturnMode = ModeBrowse
	m.refPeekReturnMode = ModeLineSelect
	m.refPeekList = []resolvedRef{{}} // viewRefPeek indexes the list; one unresolved ref
	m.refPeekErr = "unresolved"
	m.mode = mode
	return m
}

func TestModeRegistryCoversEveryMode(t *testing.T) {
	for _, mode := range allModes {
		d, ok := modeRegistry[mode]
		if !ok {
			t.Errorf("mode %v missing from modeRegistry", mode)
			continue
		}
		if d.handleKeys == nil {
			t.Errorf("mode %v registered without a key handler", mode)
		}
		if d.view == nil {
			t.Errorf("mode %v registered without a view", mode)
		}
	}
	if len(modeRegistry) != len(allModes) {
		t.Errorf("registry has %d entries, enum has %d — keep them in sync", len(modeRegistry), len(allModes))
	}
}

func TestHandleKeyPressRoutesEveryModeWithoutPanic(t *testing.T) {
	for _, mode := range allModes {
		m := dispatchModel(t, mode)
		next, _ := m.handleKeyPress(keyMsg("z")) // benign: unbound everywhere
		if next == nil {
			t.Errorf("mode %v: handleKeyPress returned nil model", mode)
		}
	}
}

func TestUpdateByModeRoutesEveryModeWithoutPanic(t *testing.T) {
	for _, mode := range allModes {
		m := dispatchModel(t, mode)
		next, _ := m.updateByMode(tea.MouseWheelMsg{})
		if next == nil {
			t.Errorf("mode %v: updateByMode returned nil model", mode)
		}
	}
}

func TestViewRendersEveryModeWithoutPanic(t *testing.T) {
	for _, mode := range allModes {
		m := dispatchModel(t, mode)
		if out := m.View().Content; out == "" {
			t.Errorf("mode %v: View returned empty output", mode)
		}
		if out := m.View().Content; out == "Unknown mode" {
			t.Errorf("mode %v: View fell through to the unknown-mode fallback", mode)
		}
	}
}

func TestHandleKeyPressDispatchesThroughRegistry(t *testing.T) {
	// Line-select: j moves the cursor
	ls := dispatchModel(t, ModeLineSelect)
	next, _ := ls.handleKeyPress(keyMsg("j"))
	if got := next.(Model).selectedLine; got != 2 {
		t.Errorf("line-select j should move cursor from 1 to 2, got %d", got)
	}

	// Browse: ? opens help
	br := dispatchModel(t, ModeBrowse)
	next2, _ := br.handleKeyPress(keyMsg("?"))
	if got := next2.(Model).mode; got != ModeHelp {
		t.Errorf("browse ? should open help, got %v", got)
	}

	// Verdict: esc returns to the recorded mode
	vd := dispatchModel(t, ModeVerdict)
	next3, _ := vd.handleKeyPress(keyMsg("esc"))
	if got := next3.(Model).mode; got != ModeBrowse {
		t.Errorf("verdict esc should return to browse, got %v", got)
	}
}

func TestHandleResizeViewportDimensions(t *testing.T) {
	sizes := []struct{ w, h int }{
		{200, 60},
		{120, 40},
		{80, 24},
		{40, 12},
		{10, 3}, // tiny terminal: dims stay consistent, no panic
	}
	for _, s := range sizes {
		m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "note", Author: "a"}})
		next, _ := m.Update(tea.WindowSizeMsg{Width: s.w, Height: s.h})
		nm := next.(Model)

		if !nm.ready {
			t.Fatalf("%dx%d: resize should initialize viewports and set ready", s.w, s.h)
		}
		if nm.documentViewport.Width() != nm.docPaneWidth() {
			t.Errorf("%dx%d: document width = %d, want %d", s.w, s.h, nm.documentViewport.Width(), nm.docPaneWidth())
		}
		// Panes get everything the chrome does not: title, review rail, hint
		// bar. Asserted through chromeRows so the test cannot drift from the
		// layout the way nine hard-coded height-2 literals did.
		wantHeight := max(s.h-chromeRows, 1)
		if nm.documentViewport.Height() != wantHeight {
			t.Errorf("%dx%d: document height = %d, want %d", s.w, s.h, nm.documentViewport.Height(), wantHeight)
		}
		wantPanel := max(s.w-nm.docPaneWidth()-4, 0)
		if nm.commentViewport.Width() != wantPanel {
			t.Errorf("%dx%d: panel width = %d, want %d (never negative)", s.w, s.h, nm.commentViewport.Width(), wantPanel)
		}
		if nm.threadViewport.Width() != s.w-4 {
			t.Errorf("%dx%d: thread width = %d, want %d", s.w, s.h, nm.threadViewport.Width(), s.w-4)
		}
	}
}

func TestHandleResizeUpdatesExistingViewports(t *testing.T) {
	m := testModel(nil)
	first, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	second, _ := first.(Model).Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	nm := second.(Model)

	wantHeight := 20 - chromeRows
	if nm.documentViewport.Width() != nm.docPaneWidth() || nm.documentViewport.Height() != wantHeight {
		t.Errorf("second resize should update dims in place, got %dx%d",
			nm.documentViewport.Width(), nm.documentViewport.Height())
	}
	if nm.threadViewport.Width() != 76 || nm.threadViewport.Height() != wantHeight {
		t.Errorf("thread viewport not resized, got %dx%d", nm.threadViewport.Width(), nm.threadViewport.Height())
	}
}

func TestHandleResizeHiddenSidebarGivesDocumentFullWidth(t *testing.T) {
	m := testModel(nil)
	m.sidebarDensity = densityHidden
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	nm := next.(Model)

	if nm.documentViewport.Width() != 98 {
		t.Errorf("hidden sidebar: document width = %d, want 98 (width-2)", nm.documentViewport.Width())
	}
	if nm.commentViewport.Width() != 0 {
		t.Errorf("hidden sidebar: panel width = %d, want 0", nm.commentViewport.Width())
	}
}

func TestHandleResizeSkipsFilePicker(t *testing.T) {
	m := NewModel() // ModeFilePicker
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	nm := next.(Model)
	if nm.ready {
		t.Error("file picker mode should not initialize document viewports on resize")
	}
}
