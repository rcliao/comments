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

	tea "charm.land/bubbletea/v2"

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

// [r] records a "commented" reply-pass: the human answered threads and hands
// the turn back — no gate judgment, exit 0, agents iterate.
func TestVerdictReplyPassRecordsCommented(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "note", Author: "rcliao"}})
	m.filename = filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(m.filename, []byte(tuiTestDoc), 0644); err != nil {
		t.Fatal(err)
	}
	m.verdictReturnMode = ModeBrowse
	m.mode = ModeVerdict

	done, cmd := m.handleVerdictKeys(keyMsg("r"))
	dm := done.(Model)
	if cmd == nil || dm.VerdictDecision != comment.DecisionCommented {
		t.Fatalf("r should record commented and quit, got %q", dm.VerdictDecision)
	}
	if len(dm.doc.Reviews) != 1 || dm.doc.Reviews[0].Decision != comment.DecisionCommented {
		t.Fatalf("commented signoff not recorded: %+v", dm.doc.Reviews)
	}
	if out := dm.renderVerdictBox(); !strings.Contains(out, "[r] Reply-pass") {
		t.Error("verdict dialog should offer the reply-pass action")
	}
}

// A TUI session holding stale state must not clobber what an agent wrote to
// disk meanwhile: signing off refreshes from disk first, so the agent's reply
// AND the human's verdict both survive (the live lost-update from dogfooding).
func TestVerdictSurvivesConcurrentAgentWrites(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(mdPath, []byte(tuiTestDoc), 0644); err != nil {
		t.Fatal(err)
	}
	doc := &comment.DocumentWithComments{Content: tuiTestDoc, Threads: []*comment.Comment{
		{ID: "c1", Line: 5, Author: "rcliao", Text: "question for the agent"},
	}}
	if err := comment.SaveToSidecar(mdPath, doc); err != nil {
		t.Fatal(err)
	}
	// Human opens the TUI (loads current state)
	loaded, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModelWithFile(loaded, mdPath)
	m.width, m.height = 100, 40
	m.handleResize()

	// Agent replies AND edits the doc on disk while the TUI is open
	external, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := comment.AddReplyToThread(external.Threads, "c1", "claude", "answered while you were reading"); err != nil {
		t.Fatal(err)
	}
	external.Content = tuiTestDoc + "\nAgent-added line.\n"
	if err := comment.SaveDocumentContent(mdPath, external); err != nil {
		t.Fatal(err)
	}
	if err := comment.SaveToSidecar(mdPath, external); err != nil {
		t.Fatal(err)
	}

	// Human signs off from the (stale) TUI
	m.verdictReturnMode = ModeBrowse
	m.mode = ModeVerdict
	done, _ := m.handleVerdictKeys(keyMsg("a"))
	dm := done.(Model)
	if dm.err != nil {
		t.Fatal(dm.err)
	}

	// Both writers' work survives on disk
	final, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(final.Content, "Agent-added line.") {
		t.Error("signoff clobbered the agent's document edit")
	}
	if len(final.Threads[0].Replies) != 1 {
		t.Error("signoff clobbered the agent's reply")
	}
	if len(final.Reviews) != 1 || final.Reviews[0].Decision != comment.DecisionApproved {
		t.Errorf("human's signoff lost: %+v", final.Reviews)
	}
}

// Adding a comment while the document changed on disk must land on the TEXT
// the human was looking at, not the stale line number (live report: comments
// landing on the wrong line). refreshDocFromDisk swaps content under a live
// cursor; the cursor must move with its line.
func TestAddCommentSurvivesExternalEditAboveCursor(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	content := "# Title\n\nAlpha.\n\nTarget line here.\n"
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	doc := &comment.DocumentWithComments{Content: content, Threads: []*comment.Comment{}}
	if err := comment.SaveToSidecar(mdPath, doc); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModelWithFile(loaded, mdPath)
	m.width, m.height = 100, 40
	m.handleResize()

	// Human puts the cursor on "Target line here." (line 5)
	m.mode = ModeLineSelect
	m.selectedLine = 5

	// An agent inserts three lines ABOVE while the TUI is open
	external, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	external.Content = "# Title\n\nNew para one.\nNew para two.\nNew para three.\n\nAlpha.\n\nTarget line here.\n"
	if err := comment.SaveDocumentContent(mdPath, external); err != nil {
		t.Fatal(err)
	}
	if err := comment.SaveToSidecar(mdPath, external); err != nil {
		t.Fatal(err)
	}

	// Human composes and saves at the cursor
	m.mode = ModeAddComment
	m.commentInput.SetValue("about the target")
	done, _ := m.handleAddCommentKeys(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	dm := done.(Model)
	if dm.err != nil {
		t.Fatal(dm.err)
	}

	final, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(final.Threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(final.Threads))
	}
	got := final.Threads[0].Line
	lines := strings.Split(final.Content, "\n")
	if got < 1 || got > len(lines) || lines[got-1] != "Target line here." {
		t.Errorf("comment landed on line %d (%q), want the line the human was looking at (Target line here., now line 9)",
			got, lines[min(got-1, len(lines)-1)])
	}
}

// P sorts the sidebar into walkthrough order: the threads ARE the highlight
// layer of a big artifact — priority-high decisions/asks first, doc order
// within a priority; second press restores document order.
func TestPrioritySortWalkthroughOrder(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Author: "claude", Text: "minor nit", Priority: "low"},
		{ID: "c2", Line: 9, Author: "claude", Text: "pivotal decision", Priority: "high"},
		{ID: "c3", Line: 7, Author: "claude", Text: "context note"},
	})
	m.width, m.height = 100, 40
	m.handleResize()

	next, _ := m.handleBrowseKeys(keyMsg("P"))
	nm := next.(Model)
	got := nm.visibleComments()
	if got[0].ID != "c2" || got[1].ID != "c3" || got[2].ID != "c1" {
		t.Fatalf("walkthrough order should be high, default, low — got %s %s %s", got[0].ID, got[1].ID, got[2].ID)
	}
	if out := nm.renderComments(); !strings.Contains(out, "by priority") || !strings.Contains(out, "↑") {
		t.Errorf("sidebar should badge the mode and the high thread:\n%s", out)
	}

	back, _ := nm.handleBrowseKeys(keyMsg("P"))
	bm := back.(Model)
	got = bm.visibleComments()
	if got[0].ID != "c1" || got[1].ID != "c3" || got[2].ID != "c2" {
		t.Errorf("second P should restore document order, got %s %s %s", got[0].ID, got[1].ID, got[2].ID)
	}
}

// Condensed rows are content-first: sigils + emoji + text fill the width,
// @author · id trail dimmed — no word badges, no doubled type marker.
func TestCondensedRowContentFirst(t *testing.T) {
	m := testModel([]*comment.Comment{{
		ID: "cma7o", Line: 3, Author: "claude", Priority: "high", Blocking: true,
		Text: "[Q] Recording your chat veto: mermaid-ER path rejected",
	}})
	m.width, m.height = 100, 40
	m.handleResize()
	m.sidebarDensity = densityCondensed
	out := stripANSI(m.renderComments())
	for _, want := range []string{"↑", "⛔", "❓ Recording", "cma7o"} {
		if !strings.Contains(out, want) {
			t.Errorf("condensed row missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "· @claude") {
		t.Errorf("author should trail dimmed:\n%s", out)
	}
	for _, gone := range []string{"[HIGH]", "[BLOCKING]", "❓ [Q]"} {
		if strings.Contains(out, gone) {
			t.Errorf("condensed row should not contain %q:\n%s", gone, out)
		}
	}
}

// / search (keybind review): incremental jump, empty-Enter repeat of the
// last query with wrap, Esc restoring the origin; n/N stay NEW-nav.
func TestSlashSearch(t *testing.T) {
	m := testModel(nil) // tuiTestDoc: Alpha at 5, Beta at 9
	m.width, m.height = 100, 40
	m.handleResize()
	m.mode = ModeLineSelect
	m.selectedLine = 1

	// / opens the prompt; typing jumps incrementally
	next, _ := m.handleLineSelectKeys(keyMsg("/"))
	sm := next.(Model)
	if sm.mode != ModeSearch {
		t.Fatalf("/ should open search, got %v", sm.mode)
	}
	for _, ch := range "beta" {
		n, _ := sm.handleSearchKeys(keyMsg(string(ch)))
		sm = n.(Model)
	}
	if sm.selectedLine != 7 && sm.selectedLine != 9 {
		t.Fatalf("incremental search should jump to a Beta line, got %d", sm.selectedLine)
	}
	betaLine := sm.selectedLine

	// Enter accepts and remembers the query
	n2, _ := sm.handleSearchKeys(keyMsg("enter"))
	am := n2.(Model)
	if am.mode != ModeLineSelect || am.searchQuery != "beta" {
		t.Fatalf("enter should accept, got mode=%v query=%q", am.mode, am.searchQuery)
	}

	// / + empty Enter = next match (wraps back to the same line when unique-ish)
	n3, _ := am.handleLineSelectKeys(keyMsg("/"))
	rm := n3.(Model)
	n4, _ := rm.handleSearchKeys(keyMsg("enter"))
	rm = n4.(Model)
	if rm.selectedLine == 0 || rm.searchQuery != "beta" {
		t.Fatalf("empty enter should repeat the search, got line=%d", rm.selectedLine)
	}

	// Esc restores the origin
	n5, _ := rm.handleLineSelectKeys(keyMsg("/"))
	em := n5.(Model)
	for _, ch := range "alpha" {
		n, _ := em.handleSearchKeys(keyMsg(string(ch)))
		em = n.(Model)
	}
	if em.selectedLine == betaLine {
		t.Fatal("typing alpha should have moved the cursor")
	}
	n6, _ := em.handleSearchKeys(keyMsg("esc"))
	fm := n6.(Model)
	if fm.selectedLine != rm.selectedLine {
		t.Errorf("esc should restore the pre-search cursor %d, got %d", rm.selectedLine, fm.selectedLine)
	}

	// Browse / lands in line-select at the match
	bm := testModel(nil)
	bm.width, bm.height = 100, 40
	bm.handleResize()
	bm.mode = ModeBrowse
	n7, _ := bm.handleBrowseKeys(keyMsg("/"))
	brm := n7.(Model)
	if brm.mode != ModeSearch || brm.searchReturnMode != ModeLineSelect {
		t.Errorf("browse / should search into line-select, got %v -> %v", brm.mode, brm.searchReturnMode)
	}
}

// Search cycling inside the prompt: tab hops to the next match past the
// cursor (the fix for "the highlight didn't move" when the query matches
// the current line), shift+tab goes back, and the prompt shows n/total.
func TestSlashSearchCycling(t *testing.T) {
	m := testModel(nil) // "Beta" on lines 7 and 9
	m.width, m.height = 100, 40
	m.handleResize()
	m.mode = ModeLineSelect
	m.selectedLine = 7 // already ON a matching line

	next, _ := m.handleLineSelectKeys(keyMsg("/"))
	sm := next.(Model)
	for _, ch := range "beta" {
		n, _ := sm.handleSearchKeys(keyMsg(string(ch)))
		sm = n.(Model)
	}
	if sm.selectedLine != 7 {
		t.Fatalf("query matching the origin stays put, got %d", sm.selectedLine)
	}
	n2, _ := sm.handleSearchKeys(keyMsg("tab"))
	sm = n2.(Model)
	if sm.selectedLine != 9 {
		t.Fatalf("tab should hop to the next match (9), got %d", sm.selectedLine)
	}
	n3, _ := sm.handleSearchKeys(keyMsg("shift+tab"))
	sm = n3.(Model)
	if sm.selectedLine != 7 {
		t.Fatalf("shift+tab should hop back to 7, got %d", sm.selectedLine)
	}
	if out := stripANSI(sm.viewSearch()); !strings.Contains(out, "1/2") {
		t.Errorf("prompt should show the match counter:\n%s", out)
	}
}

// hlsearch: while the prompt is open, match substrings on every matching
// line carry the search-hit background; style-only (bytes identical
// ANSI-stripped) and gone once the search closes.
func TestSearchHighlightsAllMatches(t *testing.T) {
	m := testModel(nil)
	m.width, m.height = 100, 40
	m.handleResize()
	m.mode = ModeLineSelect
	m.selectedLine = 1
	next, _ := m.handleLineSelectKeys(keyMsg("/"))
	sm := next.(Model)
	for _, ch := range "body" {
		n, _ := sm.handleSearchKeys(keyMsg(string(ch)))
		sm = n.(Model)
	}
	hit := sm.styles.searchHit.Render("body")
	out := sm.renderDocumentWithCursor()
	if got := strings.Count(out, hit); got < 1 {
		t.Fatalf("expected highlighted matches in the doc render, found %d", got)
	}
	if stripANSI(sm.highlightMatches("Alpha body text.", "body")) != "Alpha body text." {
		t.Error("highlighting must not change bytes")
	}
	// Accepting the search drops the highlight
	n2, _ := sm.handleSearchKeys(keyMsg("enter"))
	am := n2.(Model)
	if strings.Contains(am.renderDocumentWithCursor(), hit) {
		t.Error("highlight should clear when the prompt closes")
	}
}
