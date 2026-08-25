package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

// TestListSectionFilterTreeInclusive pins the comments_list section-filter
// semantics: an exact section path matches the section and all its descendant
// sub-sections (tree-inclusive), but NOT a sibling section whose title merely
// shares a string prefix. This matches the CLI (comment.GetCommentsInSection).
func TestListSectionFilterTreeInclusive(t *testing.T) {
	session := startTestSession(t)
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	content := "# Doc\n\n## Problem\n\nProblem body.\n\n### Symptoms\n\nSymptom body.\n\n## Problem Details\n\nDetails body.\n"
	if err := os.WriteFile(doc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "in problem", "line": 5,
	})
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "in symptoms", "line": 9,
	})
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "in details", "line": 13,
	})

	// Exact path: section itself + nested descendants, sibling excluded
	listed := callTool(t, session, "comments_list", map[string]any{
		"filepath": doc, "section": "Doc > Problem",
	})
	texts := map[string]bool{}
	for _, raw := range listed["comments"].([]any) {
		texts[raw.(map[string]any)["text"].(string)] = true
	}
	if listed["total"] != float64(2) || !texts["in problem"] || !texts["in symptoms"] {
		t.Errorf("expected section + descendant comments, got %v", texts)
	}
	if texts["in details"] {
		t.Error("sibling section 'Problem Details' must not match filter 'Doc > Problem'")
	}

	// Root section includes the whole tree
	listed = callTool(t, session, "comments_list", map[string]any{
		"filepath": doc, "section": "Doc",
	})
	if listed["total"] != float64(3) {
		t.Errorf("expected all 3 comments under root section, got %v", listed["total"])
	}

	// Nonexistent section: empty result, not an error
	listed = callTool(t, session, "comments_list", map[string]any{
		"filepath": doc, "section": "Doc > Missing",
	})
	if listed["total"] != float64(0) {
		t.Errorf("expected 0 comments for unknown section, got %v", listed["total"])
	}
}

// TestBatchReplyAtomicOnBadThreadID verifies the CLI batch-reply contract over
// MCP: if any thread ID is missing, the whole batch is rejected before any
// reply is added, and the error names the missing ID.
func TestBatchReplyAtomicOnBadThreadID(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "root thread", "line": 5,
	})
	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	goodID := listed["comments"].([]any)[0].(map[string]any)["id"].(string)

	errText := callToolExpectError(t, session, "comments_batch_reply", map[string]any{
		"filepath": doc,
		"replies": []any{
			map[string]any{"thread_id": goodID, "author": "claude", "text": "valid reply"},
			map[string]any{"thread_id": "c_missing", "author": "claude", "text": "dangling reply"},
		},
	})
	if !strings.Contains(errText, "c_missing") {
		t.Errorf("error should name the missing thread ID, got: %s", errText)
	}
	if !strings.Contains(errText, "no replies added") {
		t.Errorf("error should state the batch was rejected atomically, got: %s", errText)
	}

	// Atomic: the valid reply must NOT have been added either
	listed = callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	comments := listed["comments"].([]any)
	if len(comments) != 1 {
		t.Fatalf("expected only the root comment after rejected batch, got %d", len(comments))
	}
	if rc := comments[0].(map[string]any)["reply_count"]; rc != float64(0) {
		t.Errorf("expected 0 replies after rejected batch, got %v", rc)
	}

	// A fully valid batch still works
	ok := callTool(t, session, "comments_batch_reply", map[string]any{
		"filepath": doc,
		"replies": []any{
			map[string]any{"thread_id": goodID, "author": "claude", "text": "now valid"},
		},
	})
	if ok["success"] != true || ok["added_count"] != float64(1) {
		t.Errorf("valid batch should succeed: %v", ok)
	}
	if _, hasFailed := ok["failed"]; hasFailed {
		t.Errorf("successful batch should not report failed items: %v", ok)
	}
}

// TestStatusCountsAndStaleness verifies thread vs comment counts on a doc with
// nested replies, and that is_stale flips true when the markdown file changes
// after the sidecar was written.
func TestStatusCountsAndStaleness(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	// Two root threads, one nested reply on the first
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "first thread", "line": 5,
	})
	callTool(t, session, "comments_add", map[string]any{
		"filepath": doc, "author": "eric", "text": "second thread", "line": 9,
	})
	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	var firstID, secondID string
	for _, raw := range listed["comments"].([]any) {
		c := raw.(map[string]any)
		switch c["text"] {
		case "first thread":
			firstID = c["id"].(string)
		case "second thread":
			secondID = c["id"].(string)
		}
	}
	callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc, "thread_id": firstID, "author": "claude", "text": "a reply",
	})
	callTool(t, session, "comments_resolve", map[string]any{"filepath": doc, "thread_id": secondID})

	status := callTool(t, session, "comments_status", map[string]any{"filepath": doc})
	if status["total_threads"] != float64(2) {
		t.Errorf("expected 2 root threads, got %v", status["total_threads"])
	}
	if status["total_comments"] != float64(3) {
		t.Errorf("expected 3 total comments (2 roots + 1 reply), got %v", status["total_comments"])
	}
	if status["resolved_threads"] != float64(1) || status["unresolved_threads"] != float64(1) {
		t.Errorf("expected 1 resolved / 1 unresolved root thread, got %v / %v",
			status["resolved_threads"], status["unresolved_threads"])
	}
	if status["is_stale"] != false {
		t.Errorf("expected is_stale false while content unchanged, got %v", status["is_stale"])
	}

	// Modify the markdown out-of-band: sidecar hash no longer matches
	content, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doc, append(content, []byte("\nAppended out-of-band.\n")...), 0644); err != nil {
		t.Fatal(err)
	}

	status = callTool(t, session, "comments_status", map[string]any{"filepath": doc})
	if status["is_stale"] != true {
		t.Errorf("expected is_stale true after out-of-band edit, got %v", status["is_stale"])
	}

	// Loading re-validates and refreshes the sidecar, so staleness clears
	status = callTool(t, session, "comments_status", map[string]any{"filepath": doc})
	if status["is_stale"] != false {
		t.Errorf("expected is_stale false after revalidation, got %v", status["is_stale"])
	}
}

// TestAcceptShiftsLowerSuggestion verifies that accepting the upper of two
// stacked suggestions over MCP recalculates the lower suggestion's line range,
// so a subsequent accept edits the right lines.
func TestAcceptShiftsLowerSuggestion(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	// Fixture line 5: "It is slow."  line 9: "Some notes."

	upper := callTool(t, session, "comments_suggest", map[string]any{
		"filepath": doc, "author": "claude", "text": "expand",
		"start_line": 5, "end_line": 5,
		"original_text": "It is slow.",
		"proposed_text": "It is slow.\nVery slow.\nExtremely slow.",
	})
	lower := callTool(t, session, "comments_suggest", map[string]any{
		"filepath": doc, "author": "claude", "text": "improve notes",
		"start_line": 9, "end_line": 9,
		"original_text": "Some notes.",
		"proposed_text": "Better notes.",
	})
	upperID := upper["suggestion_id"].(string)
	lowerID := lower["suggestion_id"].(string)

	callTool(t, session, "comments_accept", map[string]any{"filepath": doc, "suggestion_id": upperID})

	// The lower suggestion must have shifted down by 2 (3 lines replaced 1)
	listed := callTool(t, session, "comments_list", map[string]any{"filepath": doc})
	var lowerAfter map[string]any
	for _, raw := range listed["comments"].([]any) {
		c := raw.(map[string]any)
		if c["id"] == lowerID {
			lowerAfter = c
		}
	}
	if lowerAfter == nil {
		t.Fatal("lower suggestion not found after accepting upper")
	}
	if lowerAfter["start_line"] != float64(11) || lowerAfter["end_line"] != float64(11) {
		t.Errorf("expected lower suggestion shifted to lines 11-11, got %v-%v",
			lowerAfter["start_line"], lowerAfter["end_line"])
	}

	// Accepting the lower suggestion now edits the right lines
	callTool(t, session, "comments_accept", map[string]any{"filepath": doc, "suggestion_id": lowerID})

	content, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "Extremely slow.") {
		t.Errorf("upper suggestion not applied:\n%s", text)
	}
	if !strings.Contains(text, "Better notes.") || strings.Contains(text, "Some notes.") {
		t.Errorf("lower suggestion applied to wrong lines:\n%s", text)
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 11 || lines[10] != "Better notes." {
		t.Errorf("expected line 11 to be the replaced notes line, got %q", lines[min(10, len(lines)-1)])
	}
}

// comments_status carries changed_since for a reviewer with a stored verdict
// baseline — lines, deletions, and the innermost sections touched — and omits
// it entirely for a reviewer who never signed off.
func TestStatusReportsChangedSinceReviewerVerdict(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	before := callTool(t, session, "comments_status", map[string]any{"filepath": doc, "reviewer": "eric"})
	if _, present := before["changed_since"]; present {
		t.Fatalf("no baseline → changed_since must be absent, got %v", before["changed_since"])
	}

	content, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := comment.SaveReviewBaseline(doc, "eric", string(content)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(doc, append(content, []byte("\n## Added\n\nNew section body.\n")...), 0644); err != nil {
		t.Fatal(err)
	}

	status := callTool(t, session, "comments_status", map[string]any{"filepath": doc, "reviewer": "eric"})
	cs, ok := status["changed_since"].(map[string]any)
	if !ok {
		t.Fatalf("changed_since missing: %v", status)
	}
	if cs["reviewer"] != "eric" || cs["changed_lines"].(float64) < 3 || cs["deletions"] != float64(0) {
		t.Errorf("changed_since = %v", cs)
	}
	secs := cs["changed_sections"].([]any)
	if len(secs) == 0 || !strings.HasSuffix(secs[len(secs)-1].(string), "Added") {
		t.Errorf("changed_sections should name the added section, got %v", secs)
	}
}
