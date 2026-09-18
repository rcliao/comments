package mcp

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
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

// The gate is no longer an MCP tool (the agent reads its decision through
// comments_inbox), so the parity pinned here is cross-surface: what
// comments_validate reports over MCP is exactly what the core gate — the one
// `comments gate` and the inbox both call — fails the document on.
func TestValidateOverMCPMatchesCoreGateRules(t *testing.T) {
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

	gate, err := comment.BuildGateReport(doc, false, "parity", 0)
	if err != nil {
		t.Fatal(err)
	}
	var gated []string
	for _, v := range gate.Files[0].Violations {
		gated = append(gated, v.Rule)
	}
	if !reflect.DeepEqual(gated, want) || gate.Decision != comment.DecisionChangesRequested {
		t.Fatalf("gate skipped explicit-template validation: decision %s, rules %v", gate.Decision, gated)
	}
}

// The agent's read of the same verdict: a template recorded in frontmatter
// reaches comments_inbox as per-file violations and a changes_requested
// decision, with no open thread at all. Without this an agent would see an
// empty inbox and call a structurally broken doc done.
func TestInboxCarriesTemplateViolations(t *testing.T) {
	session := startTestSession(t)
	doc, content := writeValidationParityFixture(t)
	frontmatter := "---\ncomments:\n  template: parity\n---\n\n"
	if err := os.WriteFile(doc, []byte(frontmatter+content), 0o644); err != nil {
		t.Fatal(err)
	}

	inbox := callTool(t, session, "comments_inbox", map[string]any{"filepath": doc})
	if inbox["decision"] != comment.DecisionChangesRequested || inbox["count"] != float64(0) {
		t.Fatalf("violations alone must fail the inbox decision: %v", inbox)
	}
	file := inbox["files"].([]any)[0].(map[string]any)
	if file["template"] != "parity" {
		t.Errorf("inbox file should name its template, got %v", file["template"])
	}
	validated := callTool(t, session, "comments_validate", map[string]any{"filepath": doc})
	if got, want := len(file["violations"].([]any)), len(validated["violations"].([]any)); got != want || got == 0 {
		t.Errorf("inbox reports %d violations, comments_validate %d", got, want)
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

// TestReplyAtomicOnBadThreadID verifies the reply contract over MCP: if any
// thread ID is missing, the whole call is rejected before any reply is added,
// and the error names both the item and the missing ID.
func TestReplyAtomicOnBadThreadID(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)

	goodID := addOne(t, session, doc, map[string]any{
		"author": "eric", "text": "root thread", "line": 5,
	})

	errText := callToolExpectError(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies": []any{
			map[string]any{"thread_id": goodID, "author": "claude", "text": "valid reply"},
			map[string]any{"thread_id": "c_missing", "author": "claude", "text": "dangling reply"},
		},
	})
	if !strings.Contains(errText, "c_missing") {
		t.Errorf("error should name the missing thread ID, got: %s", errText)
	}
	if !strings.Contains(errText, "reply 2") {
		t.Errorf("error should name the failing item, got: %s", errText)
	}

	// Atomic: the valid reply must NOT have been added either
	comments := listThreads(t, session, doc)
	if len(comments) != 1 {
		t.Fatalf("expected only the root comment after rejected call, got %d", len(comments))
	}
	if rc := comments[0].(map[string]any)["reply_count"]; rc != float64(0) {
		t.Errorf("expected 0 replies after rejected call, got %v", rc)
	}

	// A fully valid call still works
	ok := callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies": []any{
			map[string]any{"thread_id": goodID, "author": "claude", "text": "now valid"},
		},
	})
	if ok["count"] != float64(1) || len(ok["replied"].([]any)) != 1 || ok["replied"].([]any)[0] != goodID {
		t.Errorf("valid reply should succeed: %v", ok)
	}
	if resolved := ok["resolved"].([]any); len(resolved) != 0 {
		t.Errorf("a plain reply must not resolve anything: %v", ok)
	}
}

// Text may be empty only when resolving: a bare resolve is allowed (the fix is
// in the document), an empty plain reply is a mistake worth refusing.
func TestReplyRequiresTextUnlessResolving(t *testing.T) {
	session := startTestSession(t)
	doc := writeFixture(t)
	id := addOne(t, session, doc, map[string]any{"author": "eric", "text": "note", "line": 5})

	errText := callToolExpectError(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": id, "author": "claude"}},
	})
	if !strings.Contains(errText, "reply 1: text is required") {
		t.Errorf("expected a text-required error, got: %s", errText)
	}

	done := callTool(t, session, "comments_reply", map[string]any{
		"filepath": doc,
		"replies":  []any{map[string]any{"thread_id": id, "author": "claude", "resolve": true}},
	})
	if len(done["replied"].([]any)) != 0 || done["resolved"].([]any)[0] != id {
		t.Errorf("bare resolve should resolve without replying: %v", done)
	}
}
