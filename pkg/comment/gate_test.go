package comment

import (
	"os"
	"path/filepath"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func gateTestDoc() *DocumentWithComments {
	return &DocumentWithComments{
		Content: "# Doc\n\nBody\n",
		Threads: []*Comment{
			{ID: "c1", Author: "eric", Line: 3, Text: "must fix", Blocking: true},
			{ID: "c2", Author: "eric", Line: 3, Text: "nit"},
			{ID: "c3", Author: "eric", Line: 3, Text: "done", Resolved: true, Blocking: true},
			{ID: "c4", Author: "claude", Line: 3, Text: "edit", IsSuggestion: true, StartLine: 3, EndLine: 3, ProposedText: "new"},
			{ID: "c5", Author: "claude", Line: 3, Text: "edit", IsSuggestion: true, StartLine: 3, EndLine: 3, ProposedText: "new", Accepted: boolPtr(true)},
		},
	}
}

func TestEvaluateGateBlocking(t *testing.T) {
	result := EvaluateGate(gateTestDoc(), false)

	if result.Decision != DecisionChangesRequested {
		t.Errorf("expected changes_requested, got %s", result.Decision)
	}
	if len(result.Blocking) != 1 || result.Blocking[0].ID != "c1" {
		t.Errorf("expected blocking [c1], got %v", result.Blocking)
	}
	if len(result.NonBlocking) != 1 || result.NonBlocking[0].ID != "c2" {
		t.Errorf("expected non-blocking [c2], got %v", result.NonBlocking)
	}
	if len(result.PendingSuggestions) != 1 || result.PendingSuggestions[0].ID != "c4" {
		t.Errorf("expected pending suggestions [c4], got %v", result.PendingSuggestions)
	}
}

func TestEvaluateGateApprovedWhenBlockingResolved(t *testing.T) {
	doc := gateTestDoc()
	doc.Threads[0].Resolved = true

	result := EvaluateGate(doc, false)
	if result.Decision != DecisionApproved {
		t.Errorf("expected approved after resolving blocking thread, got %s", result.Decision)
	}
}

func TestEvaluateGateStrict(t *testing.T) {
	doc := gateTestDoc()
	doc.Threads[0].Resolved = true

	result := EvaluateGate(doc, true)
	if result.Decision != DecisionChangesRequested {
		t.Errorf("strict mode should fail on non-blocking/pending items, got %s", result.Decision)
	}

	doc.Threads[1].Resolved = true
	doc.Threads[3].Accepted = boolPtr(false)
	result = EvaluateGate(doc, true)
	if result.Decision != DecisionApproved {
		t.Errorf("strict mode should pass once everything is resolved/decided, got %s", result.Decision)
	}
}

func TestAddReviewRecordDerivesDecision(t *testing.T) {
	doc := gateTestDoc()

	record := AddReviewRecord(doc, "eric", "", "needs another pass", false)
	if record.Decision != DecisionChangesRequested {
		t.Errorf("expected derived changes_requested, got %s", record.Decision)
	}

	doc.Threads[0].Resolved = true
	record = AddReviewRecord(doc, "eric", "", "", false)
	if record.Decision != DecisionApproved {
		t.Errorf("expected derived approved, got %s", record.Decision)
	}
	if len(doc.Reviews) != 2 {
		t.Errorf("expected 2 review records, got %d", len(doc.Reviews))
	}
}

func TestReviewsAndBlockingRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "test.md")
	content := "# Doc\n\nBody\n"
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{
			{ID: "c1", Author: "eric", Line: 3, Text: "must fix", Blocking: true},
		},
	}
	AddReviewRecord(doc, "eric", "", "", false)

	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}

	if !loaded.Threads[0].Blocking {
		t.Error("Blocking flag lost in round-trip")
	}
	if len(loaded.Reviews) != 1 || loaded.Reviews[0].Decision != DecisionChangesRequested {
		t.Errorf("Reviews lost in round-trip: %v", loaded.Reviews)
	}
}
