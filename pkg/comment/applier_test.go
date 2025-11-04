package comment

import (
	"strings"
	"testing"
)

func TestApplyMultiLineSuggestion(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    2,
		EndLine:      3,
		OriginalText: "Line 2\nLine 3",
		ProposedText: "Modified Line 2\nModified Line 3",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Modified Line 2
Modified Line 3
Line 4
Line 5`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionSingleLine(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    3,
		EndLine:      3,
		OriginalText: "Line 3",
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

func TestApplySuggestionDelete(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    2,
		EndLine:      2,
		OriginalText: "Line 2",
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

func TestApplySuggestionMultiLineExpansion(t *testing.T) {
	content := `# Title

## Section 1
Content here

## Section 2
More content`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    3,
		EndLine:      4,
		OriginalText: "## Section 1\nContent here",
		ProposedText: `## Updated Section 1
Enhanced content here
With multiple lines
And more detail`,
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `# Title

## Updated Section 1
Enhanced content here
With multiple lines
And more detail

## Section 2
More content`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionValidation(t *testing.T) {
	content := "Test content"

	// Test non-suggestion comment
	comment := &Comment{
		ID:           "c1",
		IsSuggestion: false,
	}

	_, err := ApplySuggestion(content, comment)
	if err == nil {
		t.Error("Expected error for non-suggestion comment")
	}
}

func TestApplySuggestionOriginalMismatch(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    2,
		EndLine:      2,
		OriginalText: "Wrong content", // Doesn't match actual line
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
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    10, // Out of range
		EndLine:      10,
		ProposedText: "New line",
	}

	_, err := ApplySuggestion(content, suggestion)
	if err == nil {
		t.Error("Expected error for out of range line")
	}
}

func TestApplySuggestionInvalidLineRange(t *testing.T) {
	content := "Line 1\nLine 2\nLine 3"

	// EndLine before StartLine
	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    3,
		EndLine:      1,
		ProposedText: "X",
	}

	_, err := ApplySuggestion(content, suggestion)
	if err == nil {
		t.Error("Expected error for invalid line range")
	}

	// StartLine < 1
	suggestion2 := &Comment{
		ID:           "s2",
		IsSuggestion: true,
		StartLine:    0,
		EndLine:      1,
		ProposedText: "X",
	}

	_, err = ApplySuggestion(content, suggestion2)
	if err == nil {
		t.Error("Expected error for StartLine < 1")
	}
}

func TestApplySuggestionWithoutOriginalText(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	// OriginalText is optional - suggestion should still work
	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    2,
		EndLine:      2,
		OriginalText: "", // Empty - no verification
		ProposedText: "Modified Line 2",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Modified Line 2
Line 3`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionMultipleLineDelete(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4
Line 5`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    2,
		EndLine:      4,
		OriginalText: "Line 2\nLine 3\nLine 4",
		ProposedText: "", // Delete all three lines
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Line 5`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionAtEnd(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    3,
		EndLine:      3,
		OriginalText: "Line 3",
		ProposedText: "Modified Line 3",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Line 1
Line 2
Modified Line 3`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionAtStart(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    1,
		EndLine:      1,
		OriginalText: "Line 1",
		ProposedText: "Modified Line 1",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Modified Line 1
Line 2
Line 3`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestApplySuggestionEntireDocument(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    1,
		EndLine:      3,
		OriginalText: "Line 1\nLine 2\nLine 3",
		ProposedText: "Completely new content\nWith different structure",
	}

	result, err := ApplySuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("ApplySuggestion failed: %v", err)
	}

	expected := `Completely new content
With different structure`

	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestPreviewSuggestion(t *testing.T) {
	content := `Line 1
Line 2
Line 3
Line 4`

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    2,
		EndLine:      3,
		OriginalText: "Line 2\nLine 3",
		ProposedText: "Modified Line 2\nModified Line 3",
	}

	preview, err := PreviewSuggestion(content, suggestion)
	if err != nil {
		t.Fatalf("PreviewSuggestion failed: %v", err)
	}

	// Check that preview contains key elements
	if !strings.Contains(preview, "=== Suggestion Preview ===") {
		t.Error("Preview should contain header")
	}

	if !strings.Contains(preview, "Lines 2-3") {
		t.Error("Preview should contain line range")
	}

	if !strings.Contains(preview, "--- Original") {
		t.Error("Preview should contain original section")
	}

	if !strings.Contains(preview, "+++ Proposed") {
		t.Error("Preview should contain proposed section")
	}

	if !strings.Contains(preview, "- Line 2") {
		t.Error("Preview should contain original line 2")
	}

	if !strings.Contains(preview, "+ Modified Line 2") {
		t.Error("Preview should contain proposed line 2")
	}
}

func TestPreviewSuggestionInvalidRange(t *testing.T) {
	content := "Line 1\nLine 2"

	suggestion := &Comment{
		ID:           "s1",
		IsSuggestion: true,
		StartLine:    10,
		EndLine:      10,
		ProposedText: "X",
	}

	_, err := PreviewSuggestion(content, suggestion)
	if err == nil {
		t.Error("Expected error for invalid range")
	}
}

func TestPreviewSuggestionNonSuggestion(t *testing.T) {
	content := "Line 1\nLine 2"

	comment := &Comment{
		ID:           "c1",
		IsSuggestion: false,
	}

	_, err := PreviewSuggestion(content, comment)
	if err == nil {
		t.Error("Expected error for non-suggestion")
	}
}
