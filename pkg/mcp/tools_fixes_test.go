package mcp

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
	"testing"
)

func writeValidationParityFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	templateDir := filepath.Join(dir, ".comments", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	template := `template: parity
version: 1
doc:
  check_citations: true
sections:
  - heading: Research Question
    required: true
    enumerates_questions: true
  - heading: Summary
    required: true
  - heading: Findings
    required: true
    answers_questions: true
markers:
  needs_clarification: "[NEEDS CLARIFICATION:"
  max: 1
`
	if err := os.WriteFile(filepath.Join(templateDir, "parity.yaml"), []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(dir, "research.md")
	content := "# R\n\n## Research Question\n\nQ1. Covered?\nQ2. Missing?\n\n## Findings\n\n### F1 [Q1]\n\nSee pkg/nope.go:9.\n\n[NEEDS CLARIFICATION: one]\n[NEEDS CLARIFICATION: two]\n"
	if err := os.WriteFile(doc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc, content
}

func TestValidateAndGateUseSamePathAwareRulesOverMCP(t *testing.T) {
	session := startTestSession(t)
	doc, _ := writeValidationParityFixture(t)
	validated := callTool(t, session, "comments_validate", map[string]any{"filepath": doc, "template": "parity"})
	var got []string
	for _, raw := range validated["violations"].([]any) {
		got = append(got, raw.(map[string]any)["rule"].(string))
	}
	want := []string{"missing_section", "uncovered_question", "unresolved_marker", "unresolved_marker", "too_many_markers", "unresolvable_citation"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MCP validation rules/order = %v, want %v", got, want)
	}

	callTool(t, session, "comments_seed", map[string]any{"filepath": doc, "template": "parity", "markers_only": true})
	gate := callTool(t, session, "comments_gate", map[string]any{"filepath": doc})
	files := gate["files"].([]any)
	violations := files[0].(map[string]any)["violations"].([]any)
	if len(violations) != len(want) || gate["decision"] != comment.DecisionChangesRequested {
		t.Fatalf("MCP gate skipped recorded-template validation: %v", gate)
	}
}

func TestAnalyzeMCPReturnsCoverageManifest(t *testing.T) {
	session := startTestSession(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	research := filepath.Join(dir, "research.md")
	researchBody := "# R\n\n## Research Question\n\nQ1. What?\n\n## Findings\n\n### F1 — answer [Q1]\n\nFact.\n"
	if err := os.WriteFile(research, []byte(researchBody), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(plan, []byte("# P\n\n## Current State\n\nUse research.md:9.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := callTool(t, session, "comments_analyze", map[string]any{"filepath": plan, "against": research})
	coverage := got["coverage"].([]any)
	if len(coverage) != 1 || coverage[0].(map[string]any)["status"] != "cited" {
		t.Fatalf("unexpected analyze payload: %v", got)
	}
	if got["ready"] != false || got["structure_unchecked"] != true {
		t.Fatalf("untemplated artifact must expose structure_unchecked: %v", got)
	}
}

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
