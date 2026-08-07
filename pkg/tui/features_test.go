package tui

// Tests for the review-pack features: help overlay, queued suggestion
// decisions, sidebar density cycle, virtual-text line summaries, TOC
// overlay, and position persistence.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// --- Help overlay -----------------------------------------------------------

func TestHelpOverlayOpensAndAnyKeyCloses(t *testing.T) {
	m := testModel(nil)

	next, _ := m.handleBrowseKeys(keyMsg("?"))
	nm := next.(Model)
	if nm.mode != ModeHelp {
		t.Fatalf("? from browse should open help, got %v", nm.mode)
	}
	closed, _ := nm.handleHelpKeys(keyMsg("z"))
	if closed.(Model).mode != ModeBrowse {
		t.Errorf("any key should close help back to browse, got %v", closed.(Model).mode)
	}

	// From line-select, help returns to line-select
	lineSelectAt(m, 5)
	next2, _ := m.handleLineSelectKeys(keyMsg("?"))
	nm2 := next2.(Model)
	if nm2.mode != ModeHelp {
		t.Fatalf("? from line-select should open help, got %v", nm2.mode)
	}
	closed2, _ := nm2.handleHelpKeys(keyMsg("esc"))
	if closed2.(Model).mode != ModeLineSelect {
		t.Errorf("help should return to line-select, got %v", closed2.(Model).mode)
	}
}

func TestHelpOverlayGroupsByActivity(t *testing.T) {
	out := testStyles().renderHelpOverlay()
	for _, group := range []string{"Move", "Threads", "Compose", "Review", "Exit"} {
		if !strings.Contains(out, group) {
			t.Errorf("help overlay missing %q group", group)
		}
	}
	for _, key := range []string{"] / [", "Ctrl+S", "verdict"} {
		if !strings.Contains(out, key) {
			t.Errorf("help overlay missing binding %q", key)
		}
	}
}

// --- Queued suggestion decisions -------------------------------------------

func pendingSuggestion(id string, line int, original, proposed string) *comment.Comment {
	return &comment.Comment{
		ID: id, Author: "claude", Line: line, Text: "suggestion",
		IsSuggestion: true, StartLine: line, EndLine: line,
		OriginalText: original, ProposedText: proposed,
	}
}

func TestThreadViewQueuesInsteadOfApplying(t *testing.T) {
	sug := pendingSuggestion("s1", 5, "Alpha body text.", "Better alpha body.")
	m := testModel([]*comment.Comment{sug})
	m.selectedThread = sug
	m.mode = ModeThreadView

	next, _ := m.handleThreadViewKeys(keyMsg("a"))
	nm := next.(Model)
	if got, ok := nm.suggestionQueue["s1"]; !ok || !got {
		t.Fatalf("a should queue an accept, queue=%v", nm.suggestionQueue)
	}
	if sug.Accepted != nil {
		t.Error("a must not decide the suggestion before the verdict")
	}
	if strings.Contains(nm.doc.Content, "Better alpha body.") {
		t.Error("a must not mutate document content before the verdict")
	}

	// x overwrites the queued decision with a reject
	next2, _ := nm.handleThreadViewKeys(keyMsg("x"))
	nm2 := next2.(Model)
	if got, ok := nm2.suggestionQueue["s1"]; !ok || got {
		t.Errorf("x should queue a reject, queue=%v", nm2.suggestionQueue)
	}
	if sug.Accepted != nil {
		t.Error("x must not decide the suggestion before the verdict")
	}
	// Thread view surfaces the queued state
	if out := nm2.renderThread(); !strings.Contains(out, "QUEUED: REJECT") {
		t.Errorf("thread view should show queued decision, got:\n%s", out)
	}
}

func TestVerdictAppliesQueuedDecisionsAtomically(t *testing.T) {
	accept := pendingSuggestion("s1", 5, "Alpha body text.", "one\ntwo")
	reject := pendingSuggestion("s2", 9, "Beta body text.", "never applied")
	after := &comment.Comment{ID: "c3", Line: 9, Text: "anchor after edit", Author: "rcliao"}
	m := testModel([]*comment.Comment{accept, reject, after})
	m.filename = filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(m.filename, []byte(tuiTestDoc), 0644); err != nil {
		t.Fatal(err)
	}
	m.verdictReturnMode = ModeBrowse
	m.mode = ModeVerdict
	m.queueDecision("s1", true)
	m.queueDecision("s2", false)

	// Esc keeps the queue
	back, _ := m.handleVerdictKeys(keyMsg("esc"))
	bm := back.(Model)
	if bm.mode != ModeBrowse || len(bm.suggestionQueue) != 2 {
		t.Fatalf("esc should keep the queue, mode=%v queue=%v", bm.mode, bm.suggestionQueue)
	}

	// Submitting applies everything and records the signoff
	bm.mode = ModeVerdict
	done, cmd := bm.handleVerdictKeys(keyMsg("a"))
	dm := done.(Model)
	if dm.err != nil {
		t.Fatalf("verdict apply failed: %v", dm.err)
	}
	if cmd == nil || dm.VerdictDecision != comment.DecisionApproved {
		t.Fatalf("verdict should record approval and quit, got %q", dm.VerdictDecision)
	}
	if !strings.Contains(dm.doc.Content, "one\ntwo") {
		t.Error("accepted suggestion not applied to content")
	}
	if strings.Contains(dm.doc.Content, "never applied") {
		t.Error("rejected suggestion must not be applied")
	}
	if accept.Accepted == nil || !*accept.Accepted {
		t.Error("accepted suggestion not marked accepted")
	}
	if reject.Accepted == nil || *reject.Accepted {
		t.Error("rejected suggestion not marked rejected")
	}
	// s1 replaced 1 line with 2: anchors after line 5 shift down by one
	if after.Line != 10 {
		t.Errorf("comment anchor should shift from 9 to 10 after applied edit, got %d", after.Line)
	}
	if len(dm.suggestionQueue) != 0 {
		t.Errorf("queue should be cleared after applying, got %v", dm.suggestionQueue)
	}
	if len(dm.doc.Reviews) != 1 || dm.doc.Reviews[0].Decision != comment.DecisionApproved {
		t.Errorf("signoff not recorded: %v", dm.doc.Reviews)
	}
}

func TestVerdictDialogShowsQueueCount(t *testing.T) {
	sug := pendingSuggestion("s1", 5, "Alpha body text.", "x")
	m := testModel([]*comment.Comment{sug})
	m.queueDecision("s1", true)
	if out := m.viewVerdict(); !strings.Contains(out, "1 queued suggestion decision") {
		t.Errorf("verdict dialog should show queue count, got:\n%s", out)
	}
}

// The TUI verdict records the same ReviewRecord `comments signoff` writes,
// note included — an agent polling check_review or watching --until signoff
// gets the human's message either way.
func TestVerdictRecordsNoteLikeSignoff(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "note", Author: "rcliao"}})
	m.filename = filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(m.filename, []byte(tuiTestDoc), 0644); err != nil {
		t.Fatal(err)
	}
	m.width, m.height = 100, 40
	m.handleResize()
	m.verdictReturnMode = ModeBrowse
	m.mode = ModeVerdict

	// n opens the note; "a" and "c" are letters there, not verdicts
	noted, _ := m.handleVerdictKeys(keyMsg("n"))
	nm := noted.(Model)
	if nm.mode != ModeVerdictNote {
		t.Fatalf("n should focus the review note, got %v", nm.mode)
	}
	for _, ch := range "back to a case" {
		next, _ := nm.handleVerdictNoteKeys(keyMsg(string(ch)))
		nm = next.(Model)
	}
	if nm.VerdictDecision != "" {
		t.Fatalf("typing in the note must not submit a verdict, got %q", nm.VerdictDecision)
	}
	if nm.mode != ModeVerdictNote {
		t.Fatalf("typing should stay in the note, got %v", nm.mode)
	}

	// Esc returns to the dialog keeping the text, which the box shows back
	back, _ := nm.handleVerdictNoteKeys(keyMsg("esc"))
	bm := back.(Model)
	if bm.mode != ModeVerdict {
		t.Fatalf("esc should return to the verdict dialog, got %v", bm.mode)
	}
	if out := bm.renderVerdictBox(); !strings.Contains(out, "back to a case") {
		t.Errorf("verdict dialog should show the note it will record:\n%s", out)
	}

	done, _ := bm.handleVerdictKeys(keyMsg("c"))
	dm := done.(Model)
	if len(dm.doc.Reviews) != 1 {
		t.Fatalf("expected one review record, got %v", dm.doc.Reviews)
	}
	rec := dm.doc.Reviews[0]
	if rec.Decision != comment.DecisionChangesRequested || rec.Note != "back to a case" {
		t.Errorf("review record = %+v, want changes_requested with the typed note", rec)
	}

	// And it survives the round trip through the sidecar, where signoff,
	// gate and check_review read it from
	reloaded, _, err := comment.LoadFromSidecar(dm.filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Reviews) != 1 || reloaded.Reviews[0].Note != "back to a case" {
		t.Errorf("note not persisted to the sidecar: %+v", reloaded.Reviews)
	}
}

// --- Sidebar density cycle --------------------------------------------------

func TestSidebarDensityCycle(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "note", Author: "rcliao"},
	})
	m.width, m.height = 100, 40

	next, _ := m.handleBrowseKeys(keyMsg("S"))
	nm := next.(Model)
	if nm.sidebarDensity != densityCondensed {
		t.Fatalf("first S should condense, got %d", nm.sidebarDensity)
	}
	if out := nm.renderComments(); strings.Contains(out, "▼") {
		t.Errorf("condensed sidebar must collapse every group, got:\n%s", out)
	}

	next2, _ := nm.handleBrowseKeys(keyMsg("S"))
	nm2 := next2.(Model)
	if nm2.sidebarDensity != densityHidden {
		t.Fatalf("second S should hide sidebar, got %d", nm2.sidebarDensity)
	}
	if out := nm2.renderComments(); out != "" {
		t.Errorf("hidden sidebar should render nothing, got %q", out)
	}
	if nm2.docPaneWidth() != nm2.width-2 {
		t.Errorf("hidden sidebar should give document the full width, got %d of %d", nm2.docPaneWidth(), nm2.width)
	}

	next3, _ := nm2.handleBrowseKeys(keyMsg("S"))
	nm3 := next3.(Model)
	if nm3.sidebarDensity != densityFull {
		t.Errorf("third S should return to full, got %d", nm3.sidebarDensity)
	}
	if nm3.docPaneWidth() != 60 {
		t.Errorf("full density should restore the 60%% split, got %d", nm3.docPaneWidth())
	}
}

// --- Virtual-text line summaries -------------------------------------------

func TestLineSummaryPure(t *testing.T) {
	st := testStyles()
	if got := st.lineSummary(nil, time.Time{}); got != "" {
		t.Errorf("no threads should yield empty summary, got %q", got)
	}
	threads := []*comment.Comment{
		{Author: "rcliao", Text: "open one"},
		{Author: "claude", Text: "done", Resolved: true},
	}
	got := st.lineSummary(threads, time.Time{})
	if !strings.Contains(got, "@rcliao") || !strings.Contains(got, "×2") || !strings.Contains(got, "1 open") {
		t.Errorf("summary should read like `· @rcliao ×2 1 open`, got %q", got)
	}
}

func TestDocumentLineSummariesToggle(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "note", Author: "rcliao"},
	})

	if out := m.renderDocument(); !strings.Contains(out, "· @rcliao ×1 1 open") {
		t.Errorf("summaries should be on by default in renderDocument, got:\n%s", out)
	}
	lineSelectAt(m, 5)
	if out := m.renderDocumentWithCursor(); !strings.Contains(out, "· @rcliao") {
		t.Errorf("summaries should render in cursor view too, got:\n%s", out)
	}

	next, _ := m.handleLineSelectKeys(keyMsg("L"))
	nm := next.(Model)
	if nm.showLineSummaries {
		t.Fatal("L should toggle summaries off")
	}
	if out := nm.renderDocument(); strings.Contains(out, "· @rcliao") {
		t.Errorf("toggled-off summaries must not render, got:\n%s", out)
	}
}

// --- TOC overlay ------------------------------------------------------------

func TestBuildTOCCountsOpenComments(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "open in alpha", Author: "r"},
		{ID: "c2", Line: 9, Text: "resolved in beta", Author: "r", Resolved: true},
	})
	entries := buildTOC(m.documentSections, m.doc.Threads)
	if len(entries) != 3 {
		t.Fatalf("expected 3 sections (Title, Alpha, Beta), got %d: %v", len(entries), entries)
	}
	byPath := map[string]tocEntry{}
	for _, e := range entries {
		byPath[e.path] = e
	}
	if byPath["Title"].open != 1 {
		t.Errorf("Title should count 1 open (nested), got %d", byPath["Title"].open)
	}
	if byPath["Title > Alpha"].open != 1 {
		t.Errorf("Alpha should count 1 open, got %d", byPath["Title > Alpha"].open)
	}
	if byPath["Title > Beta"].open != 0 {
		t.Errorf("Beta should count 0 open (resolved excluded), got %d", byPath["Title > Beta"].open)
	}
}

func TestTOCOverlayNavigateAndJump(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "x", Author: "r"}})
	m.width, m.height = 100, 40

	next, _ := m.handleBrowseKeys(keyMsg("t"))
	nm := next.(Model)
	if nm.mode != ModeTOC {
		t.Fatalf("t should open TOC, got %v", nm.mode)
	}

	// j/j moves to the third entry (Title > Beta, heading line 7)
	step, _ := nm.handleTOCKeys(keyMsg("j"))
	step2, _ := step.(Model).handleTOCKeys(keyMsg("j"))
	sm := step2.(Model)
	if sm.tocSelected != 2 {
		t.Fatalf("j/j should select entry 2, got %d", sm.tocSelected)
	}

	jumped, _ := sm.handleTOCKeys(keyMsg("enter"))
	jm := jumped.(Model)
	if jm.mode != ModeLineSelect || jm.selectedLine != 7 {
		t.Errorf("Enter should jump to Beta heading (line 7) in line-select, got mode=%v line=%d", jm.mode, jm.selectedLine)
	}

	// Esc closes back to the mode the TOC was opened from
	reopened, _ := m.handleBrowseKeys(keyMsg("t"))
	closed, _ := reopened.(Model).handleTOCKeys(keyMsg("esc"))
	if closed.(Model).mode != ModeBrowse {
		t.Errorf("esc should close TOC back to browse, got %v", closed.(Model).mode)
	}

	rendered := nm.styles.renderTOC(nm.tocEntries, 0)
	if !strings.Contains(rendered, "Title > Alpha") || !strings.Contains(rendered, "1 open") {
		t.Errorf("TOC render should show paths and open counts, got:\n%s", rendered)
	}
}

// --- Position persistence ---------------------------------------------------

func TestViewStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc.md")

	if _, ok := loadViewState(docPath); ok {
		t.Fatal("no state saved yet: load should report not-found")
	}
	if err := saveViewState(docPath, viewState{SelectedLine: 42, YOffset: 17}); err != nil {
		t.Fatalf("saveViewState: %v", err)
	}
	st, ok := loadViewState(docPath)
	if !ok || st.SelectedLine != 42 || st.YOffset != 17 {
		t.Errorf("round trip failed: ok=%v state=%+v", ok, st)
	}

	// Second document in the same directory shares the state file
	otherPath := filepath.Join(dir, "other.md")
	if err := saveViewState(otherPath, viewState{SelectedLine: 7}); err != nil {
		t.Fatalf("saveViewState other: %v", err)
	}
	st, _ = loadViewState(docPath)
	if st.SelectedLine != 42 {
		t.Errorf("state for doc.md clobbered by other.md: %+v", st)
	}
}

func TestNewModelWithFileRestoresPosition(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc.md")
	if err := saveViewState(docPath, viewState{SelectedLine: 9, YOffset: 3}); err != nil {
		t.Fatalf("saveViewState: %v", err)
	}

	doc := &comment.DocumentWithComments{Content: tuiTestDoc}
	m := NewModelWithFile(doc, docPath)
	if m.selectedLine != 9 || m.restoredYOffset != 3 {
		t.Errorf("expected restored line 9 / offset 3, got line=%d offset=%d", m.selectedLine, m.restoredYOffset)
	}
}

func TestCtrlCSavesViewStateOnQuit(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc.md")
	doc := &comment.DocumentWithComments{Content: tuiTestDoc}
	m := NewModelWithFile(doc, docPath)
	m.selectedLine = 5

	next, cmd := m.Update(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
	_ = next
	st, ok := loadViewState(docPath)
	if !ok || st.SelectedLine != 5 {
		t.Errorf("ctrl+c should persist the reading position, ok=%v state=%+v", ok, st)
	}
}

// --- Error-state escape hatch ----------------------------------------------

func TestErrorStateClearedByAnyKey(t *testing.T) {
	m := testModel(nil)
	m.err = errors.New("boom")

	if !strings.Contains(m.View().Content, "Error: boom") {
		t.Fatal("View should show the error screen while err is set")
	}
	if !strings.Contains(m.View().Content, "Press any key to continue") {
		t.Error("error screen should advertise the any-key escape hatch")
	}

	next, _ := m.handleKeyPress(keyMsg("x"))
	nm := next.(Model)
	if nm.err != nil {
		t.Errorf("any key should clear the error, still set: %v", nm.err)
	}
	if nm.mode != ModeBrowse {
		t.Errorf("with a loaded doc, clearing the error should return to browse, got %v", nm.mode)
	}
	if strings.Contains(nm.View().Content, "Error: boom") {
		t.Error("error screen should be gone after a key press")
	}
}

func TestErrorStateWithoutDocReturnsToFilePicker(t *testing.T) {
	m := NewModel()
	m.err = errors.New("load failed")

	next, _ := m.Update(keyMsg("j"))
	nm := next.(Model)
	if nm.err != nil {
		t.Errorf("key press via Update should clear the error, still set: %v", nm.err)
	}
	if nm.mode != ModeFilePicker {
		t.Errorf("with no doc loaded, clearing the error should return to the file picker, got %v", nm.mode)
	}
}
