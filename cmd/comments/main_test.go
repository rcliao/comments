package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

func TestAnalyzeCommandJSONMatchesSharedEngine(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	research := filepath.Join(dir, "research.md")
	researchBody := "# R\n\n## Research Question\n\nQ1. What?\n\n## Findings\n\n### F1 [Q1]\n\nFact.\n"
	if err := os.WriteFile(research, []byte(researchBody), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(dir, "plan.md")
	planBody := "# P\n\n## Current State\n\nUse research.md:9.\n"
	if err := os.WriteFile(plan, []byte(planBody), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runCapture(t, "analyze", plan, "--against", research, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("analyze code=%d stderr=%q", code, stderr)
	}
	var got comment.Analysis
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid analyze JSON: %v\n%s", err, stdout)
	}
	want := comment.AnalyzeDocument(planBody, plan, nil, researchBody, research)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CLI payload drifted from shared engine\n got: %+v\nwant: %+v", got, want)
	}
}

func TestValidateCommandUsesPathAwareSharedRules(t *testing.T) {
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

	code, stdout, _ := runCapture(t, "validate", doc, "--template", "parity", "--json")
	if code != 1 {
		t.Fatalf("validate code=%d, want 1; stdout=%s", code, stdout)
	}
	var payload struct {
		Violations []comment.Violation `json:"violations"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, violation := range payload.Violations {
		got = append(got, violation.Rule)
	}
	want := []string{"missing_section", "uncovered_question", "unresolved_marker", "unresolved_marker", "too_many_markers", "unresolvable_citation"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CLI validation rules/order = %v, want %v", got, want)
	}
}

// runCapture invokes run() with os.Stdout and os.Stderr captured, returning
// the exit code and everything written to each stream.
func runCapture(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	defer func() {
		os.Stdout, os.Stderr = origOut, origErr
	}()

	outCh := make(chan string)
	errCh := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, outR)
		outCh <- b.String()
	}()
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, errR)
		errCh <- b.String()
	}()

	code = run(args)

	_ = outW.Close()
	_ = errW.Close()
	return code, <-outCh, <-errCh
}

// writeTestDoc creates a small markdown document in a temp dir.
func writeTestDoc(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc.md")
	content := "# Title\n\nFirst paragraph line.\n\n## Section\n\nSecond paragraph line.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// listedIDs runs `list --format json` and returns the thread IDs from stdout.
func listedIDs(t *testing.T, doc string) []string {
	t.Helper()
	code, stdout, _ := runCapture(t, "get", doc, "--json")
	if code != 0 {
		t.Fatalf("get --json exited %d", code)
	}
	var list struct {
		Comments []struct {
			ID string `json:"id"`
		} `json:"comments"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("get --json stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	ids := make([]string, 0, len(list.Comments))
	for _, c := range list.Comments {
		ids = append(ids, c.ID)
	}
	return ids
}

// recordVerdict is the test-only stand-in for a human's review pass. There is
// deliberately no CLI command for it: a non-interactive signoff let an agent
// record an approval under the human's name, so the verdict is written only by
// `comments view` / `comments serve` — and, here, by the same core call they use.
func recordVerdict(t *testing.T, doc, author, decision string) comment.ReviewRecord {
	t.Helper()
	loaded, _, err := comment.LoadDocument(doc)
	if err != nil {
		t.Fatalf("load %s: %v", doc, err)
	}
	record, err := comment.RecordVerdict(doc, loaded, author, decision, "")
	if err != nil {
		t.Fatalf("record verdict: %v", err)
	}
	return record
}

// TestAddGateSignoffRoundTrip exercises the add -> gate(10) -> resolve ->
// gate(0) -> signoff flow, asserting the preserved exit codes.
func TestAddGateSignoffRoundTrip(t *testing.T) {
	doc := writeTestDoc(t)

	code, stdout, stderr := runCapture(t, "add", doc,
		"--line", "3", "--author", "eric", "--text", "fix this first", "--blocking")
	if code != 0 {
		t.Fatalf("add exited %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Comment added") {
		t.Errorf("add stdout missing confirmation: %q", stdout)
	}
	if stderr != "" {
		t.Errorf("add wrote to stderr on a clean sidecar: %q", stderr)
	}

	// Open blocking comment: gate fails with the dedicated exit code
	code, _, _ = runCapture(t, "gate", doc)
	if code != comment.GateExitCode {
		t.Fatalf("gate with open blocking comment exited %d, want %d", code, comment.GateExitCode)
	}

	// gate --json: stdout stays pure JSON even when the gate fails
	code, stdout, stderr = runCapture(t, "gate", doc, "--json")
	if code != comment.GateExitCode {
		t.Fatalf("gate --json exited %d, want %d", code, comment.GateExitCode)
	}
	var gateOut struct {
		Decision string `json:"decision"`
	}
	if err := json.Unmarshal([]byte(stdout), &gateOut); err != nil {
		t.Fatalf("gate --json stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	if gateOut.Decision != comment.DecisionChangesRequested {
		t.Errorf("gate decision = %q, want %q", gateOut.Decision, comment.DecisionChangesRequested)
	}
	if stderr != "" {
		t.Errorf("gate --json wrote to stderr: %q", stderr)
	}

	// Resolve the blocking thread; gate passes with exit 0
	ids := listedIDs(t, doc)
	if len(ids) != 1 {
		t.Fatalf("expected 1 thread, got %v", ids)
	}
	code, _, stderr = runCapture(t, "reply", doc, "--thread", ids[0], "--author", "eric", "--text", "fixed", "--resolve")
	if code != 0 {
		t.Fatalf("reply --resolve exited %d, stderr: %s", code, stderr)
	}
	code, _, _ = runCapture(t, "gate", doc)
	if code != 0 {
		t.Fatalf("gate after resolve exited %d, want 0", code)
	}

	// An empty decision derives from the gate: approved once nothing blocks
	if record := recordVerdict(t, doc, "eric", ""); record.Decision != comment.DecisionApproved {
		t.Errorf("derived verdict = %q, want %q", record.Decision, comment.DecisionApproved)
	}

	// The verdict is the human's: no command may write it
	for _, removed := range []string{"signoff", "accept", "reject", "batch-accept"} {
		if code, _, stderr := runCapture(t, removed, doc); code != 1 || !strings.Contains(stderr, "Unknown command") {
			t.Errorf("%s must not exist as a command (exit %d, stderr %q)", removed, code, stderr)
		}
	}
}

// TestGetJSONStdoutCleanWithLoadWarnings edits the document out-of-band so
// loading re-anchors the comment: the warning must go to stderr while stdout
// remains valid JSON.
func TestGetJSONStdoutCleanWithLoadWarnings(t *testing.T) {
	doc := writeTestDoc(t)

	code, _, stderr := runCapture(t, "add", doc,
		"--line", "3", "--author", "eric", "--text", "anchored note")
	if code != 0 {
		t.Fatalf("add exited %d, stderr: %s", code, stderr)
	}

	// Move the commented line by inserting new content above it
	content := "# Title\n\nInserted line one.\nInserted line two.\n\nFirst paragraph line.\n\n## Section\n\nSecond paragraph line.\n"
	if err := os.WriteFile(doc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runCapture(t, "get", doc, "--json")
	if code != 0 {
		t.Fatalf("get exited %d, stderr: %s", code, stderr)
	}
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(stdout), &list); err != nil {
		t.Fatalf("stdout is not valid JSON after load warnings: %v\nstdout: %s", err, stdout)
	}
	if list.Total != 1 {
		t.Fatalf("expected 1 comment, got %d", list.Total)
	}
	if !strings.Contains(stderr, "re-anchored") {
		t.Errorf("expected re-anchoring notice on stderr, got: %q", stderr)
	}

	// The re-anchored line was persisted by the CLI (not by the load itself):
	// a second list sees the migrated sidecar and prints no further warnings
	code, _, stderr = runCapture(t, "get", doc, "--json")
	if code != 0 {
		t.Fatalf("second get exited %d", code)
	}
	if stderr != "" {
		t.Errorf("second get should load a clean sidecar, stderr: %q", stderr)
	}
}

// TestErrorsGoToStderr asserts error paths print to stderr only and exit 1.
func TestErrorsGoToStderr(t *testing.T) {
	doc := writeTestDoc(t)

	code, stdout, stderr := runCapture(t, "add", doc, "--line", "3", "--author", "eric")
	if code != 1 {
		t.Errorf("add without --text exited %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("error message leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "Error: comment 1: text is required") {
		t.Errorf("stderr missing error message: %q", stderr)
	}

	code, stdout, stderr = runCapture(t, "bogus-command")
	if code != 1 {
		t.Errorf("unknown command exited %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("unknown-command output leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "Unknown command: bogus-command") {
		t.Errorf("stderr missing unknown-command message: %q", stderr)
	}

	code, stdout, stderr = runCapture(t, "get", filepath.Join(t.TempDir(), "missing.md"))
	if code != 1 {
		t.Errorf("get on missing file exited %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("missing-file error leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "Error loading document") {
		t.Errorf("stderr missing load error: %q", stderr)
	}
}

// A verdict stores the reviewed content as the reviewer's baseline; the inbox
// then names what moved since. A reply-pass leaves the baseline alone.
func TestVerdictStoresBaselineAndInboxReportsChangedSections(t *testing.T) {
	doc := writeTestDoc(t)

	recordVerdict(t, doc, "eric", comment.DecisionApproved)
	base, ok := comment.LoadReviewBaseline(doc, "eric")
	if !ok {
		t.Fatal("an approved verdict must store a baseline")
	}
	original, _ := os.ReadFile(doc)
	if base != string(original) {
		t.Errorf("baseline = %q, want the reviewed content", base)
	}

	// Agent edits the second section
	edited := strings.Replace(string(original), "Second paragraph line.", "Second paragraph REWRITTEN.\nAnd a new line.", 1)
	if err := os.WriteFile(doc, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}

	type inboxOut struct {
		Files []struct {
			Changes *struct {
				Reviewer        string   `json:"reviewer"`
				ChangedLines    int      `json:"changed_lines"`
				ChangedSections []string `json:"changed_sections"`
			} `json:"changes"`
		} `json:"files"`
	}
	readInbox := func(args ...string) inboxOut {
		t.Helper()
		code, stdout, stderr := runCapture(t, append([]string{"inbox", doc, "--json"}, args...)...)
		if code != 0 {
			t.Fatalf("inbox exited %d: %s", code, stderr)
		}
		var out inboxOut
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("inbox --json invalid: %v\n%s", err, stdout)
		}
		return out
	}

	// No --reviewer: the inbox diffs against the latest reviewer's baseline
	ch := readInbox().Files[0].Changes
	if ch == nil || ch.Reviewer != "eric" || ch.ChangedLines != 2 || len(ch.ChangedSections) != 1 || ch.ChangedSections[0] != "Title > Section" {
		t.Errorf("inbox changes = %+v, want 2 lines in [Title > Section] since eric", ch)
	}
	code, stdout, _ := runCapture(t, "inbox", doc)
	if code != 0 || !strings.Contains(stdout, "2 line(s), 0 deletion(s) since @eric's last verdict") {
		t.Errorf("text inbox should report what changed, got:\n%s", stdout)
	}

	// A reply-pass does NOT move the baseline: marks keep reading since the verdict
	recordVerdict(t, doc, "eric", comment.DecisionCommented)
	if after, _ := comment.LoadReviewBaseline(doc, "eric"); after != base {
		t.Error("a commented reply-pass must not replace the baseline")
	}

	// No baseline for another reviewer → no changes block at all
	if ch := readInbox("--reviewer", "someone-else").Files[0].Changes; ch != nil {
		t.Errorf("inbox must omit changes when the reviewer has no baseline, got %+v", ch)
	}
}
