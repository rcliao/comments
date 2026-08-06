package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

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
	code, stdout, _ := runCapture(t, "list", doc, "--format", "json")
	if code != 0 {
		t.Fatalf("list --format json exited %d", code)
	}
	var comments []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &comments); err != nil {
		t.Fatalf("list --format json stdout is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	ids := make([]string, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.ID)
	}
	return ids
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
	code, _, stderr = runCapture(t, "resolve", doc, "--thread", ids[0])
	if code != 0 {
		t.Fatalf("resolve exited %d, stderr: %s", code, stderr)
	}
	code, _, _ = runCapture(t, "gate", doc)
	if code != 0 {
		t.Fatalf("gate after resolve exited %d, want 0", code)
	}

	// Signoff records the derived (approved) decision
	code, stdout, stderr = runCapture(t, "signoff", doc, "--author", "eric")
	if code != 0 {
		t.Fatalf("signoff exited %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Review recorded: approved") {
		t.Errorf("signoff stdout missing decision: %q", stdout)
	}
}

// TestListJSONStdoutCleanWithLoadWarnings edits the document out-of-band so
// loading re-anchors the comment: the warning must go to stderr while stdout
// remains valid JSON.
func TestListJSONStdoutCleanWithLoadWarnings(t *testing.T) {
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

	code, stdout, stderr := runCapture(t, "list", doc, "--format", "json")
	if code != 0 {
		t.Fatalf("list exited %d, stderr: %s", code, stderr)
	}
	var comments []map[string]any
	if err := json.Unmarshal([]byte(stdout), &comments); err != nil {
		t.Fatalf("stdout is not valid JSON after load warnings: %v\nstdout: %s", err, stdout)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if !strings.Contains(stderr, "re-anchored") {
		t.Errorf("expected re-anchoring notice on stderr, got: %q", stderr)
	}

	// The re-anchored line was persisted by the CLI (not by the load itself):
	// a second list sees the migrated sidecar and prints no further warnings
	code, _, stderr = runCapture(t, "list", doc, "--format", "json")
	if code != 0 {
		t.Fatalf("second list exited %d", code)
	}
	if stderr != "" {
		t.Errorf("second list should load a clean sidecar, stderr: %q", stderr)
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
	if !strings.Contains(stderr, "Error: --text flag is required") {
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

	code, stdout, stderr = runCapture(t, "list", filepath.Join(t.TempDir(), "missing.md"))
	if code != 1 {
		t.Errorf("list on missing file exited %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("missing-file error leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "Error loading document") {
		t.Errorf("stderr missing load error: %q", stderr)
	}
}
