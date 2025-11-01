package comment

import (
	"strings"
	"testing"
	"time"
)

func TestApplyLineSuggestion(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 3,
			EndLine:   3,
			Original:  "Line 3",
		},
		ProposedText: "New Line 3",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Line 2
New Line 3
Line 4
Line 5`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplyLineSuggestionMultipleLines(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 2,
			EndLine:   3,
			Original:  "Line 2\nLine 3",
		},
		ProposedText: "Replaced Line 2 and 3\nWith two new lines",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Replaced Line 2 and 3
With two new lines
Line 4`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplyLineSuggestionDelete(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 2,
			EndLine:   2,
			Original:  "Line 2",
		},
		ProposedText: "", // Delete line
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Line 3`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplyCharRangeSuggestion(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog"

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionCharRange,
		Selection: &Selection{
			ByteOffset: 4,     // "quick"
			Length:     5,
			Original:   "quick",
		},
		ProposedText: "fast",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := "The fast brown fox jumps over the lazy dog"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestApplyCharRangeSuggestionInsert(t *testing.T) {
	content := "Hello world"

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionCharRange,
		Selection: &Selection{
			ByteOffset: 5, // After "Hello"
			Length:     0, // Insert
			Original:   "",
		},
		ProposedText: ",",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := "Hello, world"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestApplyMultiLineSuggestion(t *testing.T) {
	content := `# Title

## Section 1
Content here

## Section 2
More content`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionMultiLine,
		Selection: &Selection{
			StartLine: 3,
			EndLine:   4,
			Original:  "## Section 1\nContent here",
		},
		ProposedText: `## Updated Section 1
Enhanced content here
With multiple lines`,
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `# Title

## Updated Section 1
Enhanced content here
With multiple lines

## Section 2
More content`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplyDiffHunkSuggestion(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5`

	diffHunk := `@@ -2,3 +2,3 @@
 Line 2
-Line 3
+Line 3 Modified
 Line 4`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionDiffHunk,
		Selection: &Selection{
			StartLine: 2,
		},
		ProposedText: diffHunk,
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Line 2
Line 3 Modified
Line 4
Line 5`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplyDiffHunkAddLines(t *testing.T) {
	content := `Line 1
Line 2
Line 4`

	diffHunk := `@@ -2,1 +2,2 @@
 Line 2
+Line 3
 Line 4`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionDiffHunk,
		Selection:      &Selection{StartLine: 2},
		ProposedText:   diffHunk,
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Line 2
Line 3
Line 4`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionValidation(t *testing.T) {
	content := "Test content"

	// Test non-suggestion comment
	comment := &Comment{
		ID:             "c1",
		SuggestionType: SuggestionNone,
	}

	_, err := ApplySuggestion(content, comment)
	if err == nil {
		t.Error("Expected error for non-suggestion comment")
	}

	// Test missing selection
	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection:      nil,
	}

	_, err = ApplySuggestion(content, suggestion)
	if err == nil {
		t.Error("Expected error for missing selection")
	}
}

func TestApplySuggestionOriginalMismatch(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 2,
			EndLine:   2,
			Original:  "Wrong content", // Doesn't match actual line
		},
		ProposedText: "New line",
	}

	_, err := ApplySuggestion(content, suggestion)
	if err == nil {
		t.Error("Expected error for original text mismatch")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("Expected mismatch error, got: %v", err)
	}
}

func TestApplySuggestionOutOfRange(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 10, // Out of range
			EndLine:   10,
		},
		ProposedText: "New line",
	}

	_, err := ApplySuggestion(content, suggestion)
	if err == nil {
		t.Error("Expected error for out of range line")
	}
}

func TestCanApplySuggestion(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"

	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 2,
			EndLine:   2,
			Original:  "Line 2",
		},
		ProposedText: "Modified",
	}

	err := CanApplySuggestion(content, suggestion)
	if err != nil {
		t.Errorf("CanApplySuggestion failed: %v", err)
	}

	// Test invalid suggestion
	badSuggestion := &Comment{
		ID:             "s2",
		SuggestionType: SuggestionLine,
		Selection: &Selection{
			StartLine: 100,
			EndLine:   100,
		},
		ProposedText: "X",
	}

	err = CanApplySuggestion(content, badSuggestion)
	if err == nil {
		t.Error("Expected error for invalid suggestion")
	}
}

func TestApplyMultipleSuggestions(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5`

	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 5,
				EndLine:   5,
				Original:  "Line 5",
			},
			ProposedText: "Modified Line 5",
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 2,
				EndLine:   2,
				Original:  "Line 2",
			},
			ProposedText: "Modified Line 2",
		},
	}

	// Note: Should apply bottom-to-top to avoid position drift
	// But our implementation applies in order given

	result, applied, err := ApplyMultipleSuggestions(content, suggestions)
	if err != nil {
		t.Fatalf("ApplyMultipleSuggestions failed: %v", err)
	}

	if len(applied) != 2 {
		t.Errorf("Expected 2 applied suggestions, got %d", len(applied))
	}

	// Verify both suggestions were applied
	if !strings.Contains(result, "Modified Line 5") {
		t.Error("First suggestion was not applied")
	}
	if !strings.Contains(result, "Modified Line 2") {
		t.Error("Second suggestion was not applied")
	}
}

func TestApplyMultipleSuggestionsSkipsNonPending(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"

	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptanceAccepted, // Already accepted, should skip
			Selection:       &Selection{StartLine: 1, EndLine: 1},
			ProposedText:    "X",
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending, // Should apply
			Selection: &Selection{
				StartLine: 2,
				EndLine:   2,
				Original:  "Line 2",
			},
			ProposedText: "Modified",
		},
	}

	result, applied, err := ApplyMultipleSuggestions(content, suggestions)
	if err != nil {
		t.Fatalf("ApplyMultipleSuggestions failed: %v", err)
	}

	if len(applied) != 1 {
		t.Errorf("Expected 1 applied suggestion, got %d", len(applied))
	}

	if applied[0] != "s2" {
		t.Errorf("Expected s2 to be applied, got %s", applied[0])
	}

	if !strings.Contains(result, "Modified") {
		t.Error("Pending suggestion was not applied")
	}
}

func TestHelperMethods(t *testing.T) {
	timestamp := time.Now()

	// Test IsSuggestion
	suggestion := &Comment{
		ID:             "s1",
		SuggestionType: SuggestionLine,
		Timestamp:      timestamp,
	}
	if !suggestion.IsSuggestion() {
		t.Error("IsSuggestion should return true for suggestion")
	}

	comment := &Comment{
		ID:             "c1",
		SuggestionType: SuggestionNone,
		Timestamp:      timestamp,
	}
	if comment.IsSuggestion() {
		t.Error("IsSuggestion should return false for non-suggestion")
	}

	// Test IsPending
	pending := &Comment{
		ID:              "s1",
		SuggestionType:  SuggestionLine,
		AcceptanceState: AcceptancePending,
		Timestamp:       timestamp,
	}
	if !pending.IsPending() {
		t.Error("IsPending should return true for pending suggestion")
	}

	// Test IsAccepted
	accepted := &Comment{
		ID:              "s1",
		SuggestionType:  SuggestionLine,
		AcceptanceState: AcceptanceAccepted,
		Timestamp:       timestamp,
	}
	if !accepted.IsAccepted() {
		t.Error("IsAccepted should return true for accepted suggestion")
	}

	// Test IsRejected
	rejected := &Comment{
		ID:              "s1",
		SuggestionType:  SuggestionLine,
		AcceptanceState: AcceptanceRejected,
		Timestamp:       timestamp,
	}
	if !rejected.IsRejected() {
		t.Error("IsRejected should return true for rejected suggestion")
	}
}
