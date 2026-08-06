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

// acceptAndRecalc mirrors the CLI/TUI accept flow: apply the suggestion to
// the content, mark it accepted, and recalculate all comment positions.
func acceptAndRecalc(t *testing.T, content string, threads []*Comment, s *Comment) string {
	t.Helper()
	newContent, err := ApplySuggestion(content, s)
	if err != nil {
		t.Fatalf("ApplySuggestion(%s) failed: %v", s.ID, err)
	}
	if err := AcceptSuggestion(threads, s.ID); err != nil {
		t.Fatalf("AcceptSuggestion(%s) failed: %v", s.ID, err)
	}
	RecalculateCommentLines(threads, s.StartLine, s.EndLine, SuggestionLinesAdded(s))
	return newContent
}

func TestAcceptStackedSuggestionsGrow(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6"

	first := NewSuggestion("alice", 2, 2, "expand", "line2", "line2a\nline2b\nline2c")
	second := NewSuggestion("bob", 5, 5, "fix", "line5", "LINE5")
	commentAbove := NewComment("carol", 1, "above the edit")
	commentBelow := NewComment("carol", 6, "below the edit")
	threads := []*Comment{first, second, commentAbove, commentBelow}

	// Accept the first suggestion: replaces 1 line with 3 (delta +2)
	content = acceptAndRecalc(t, content, threads, first)

	// The second pending suggestion must have shifted by +2
	if second.StartLine != 7 || second.EndLine != 7 {
		t.Fatalf("second suggestion range = %d-%d, want 7-7", second.StartLine, second.EndLine)
	}
	// Plain comment below the edit shifts; comment above does not
	if commentBelow.Line != 8 {
		t.Errorf("commentBelow line = %d, want 8", commentBelow.Line)
	}
	if commentAbove.Line != 1 {
		t.Errorf("commentAbove line = %d, want 1", commentAbove.Line)
	}

	// Accepting the second suggestion now edits the right text
	// (OriginalText verification proves the range points at "line5")
	content = acceptAndRecalc(t, content, threads, second)

	expected := "line1\nline2a\nline2b\nline2c\nline3\nline4\nLINE5\nline6"
	if content != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, content)
	}
}

func TestAcceptStackedSuggestionsShrink(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8"

	first := NewSuggestion("alice", 2, 4, "merge", "line2\nline3\nline4", "merged")
	second := NewSuggestion("bob", 6, 7, "rewrite", "line6\nline7", "SIX\nSEVEN")
	commentBelow := NewComment("carol", 8, "below both")
	threads := []*Comment{first, second, commentBelow}

	// Accept the first suggestion: replaces 3 lines with 1 (delta -2)
	content = acceptAndRecalc(t, content, threads, first)

	if second.StartLine != 4 || second.EndLine != 5 {
		t.Fatalf("second suggestion range = %d-%d, want 4-5", second.StartLine, second.EndLine)
	}
	if commentBelow.Line != 6 {
		t.Errorf("commentBelow line = %d, want 6", commentBelow.Line)
	}

	content = acceptAndRecalc(t, content, threads, second)

	expected := "line1\nmerged\nline5\nSIX\nSEVEN\nline8"
	if content != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, content)
	}
}

func TestAcceptSuggestionAfterDeletionShiftsNext(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"

	// Pure deletion: ProposedText empty removes lines 2-3 (delta -2)
	first := NewSuggestion("alice", 2, 3, "delete", "line2\nline3", "")
	second := NewSuggestion("bob", 5, 5, "fix", "line5", "LINE5")
	threads := []*Comment{first, second}

	content = acceptAndRecalc(t, content, threads, first)

	if second.StartLine != 3 || second.EndLine != 3 {
		t.Fatalf("second suggestion range = %d-%d, want 3-3", second.StartLine, second.EndLine)
	}

	content = acceptAndRecalc(t, content, threads, second)

	expected := "line1\nline4\nLINE5"
	if content != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, content)
	}
}

func TestAcceptSuggestionShiftsNestedReplies(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6"

	first := NewSuggestion("alice", 2, 2, "expand", "line2", "line2a\nline2b\nline2c")
	thread := NewComment("carol", 5, "root below edit")
	reply := NewReply("dave", "reply", thread)
	nested := NewReply("erin", "nested reply", reply)
	reply.Replies = append(reply.Replies, nested)
	thread.Replies = append(thread.Replies, reply)
	threads := []*Comment{first, thread}

	acceptAndRecalc(t, content, threads, first)

	if thread.Line != 7 {
		t.Errorf("thread line = %d, want 7", thread.Line)
	}
	if reply.Line != 7 {
		t.Errorf("reply line = %d, want 7", reply.Line)
	}
	if nested.Line != 7 {
		t.Errorf("nested reply line = %d, want 7", nested.Line)
	}
}

func TestSuggestionLinesAdded(t *testing.T) {
	cases := []struct {
		proposed string
		want     int
	}{
		{"", 0},
		{"one line", 1},
		{"a\nb", 2},
		{"a\nb\nc", 3},
	}

	for _, tc := range cases {
		s := &Comment{IsSuggestion: true, ProposedText: tc.proposed}
		if got := SuggestionLinesAdded(s); got != tc.want {
			t.Errorf("SuggestionLinesAdded(%q) = %d, want %d", tc.proposed, got, tc.want)
		}
	}
}

func TestApplyAllSuggestionsTopDownOrder(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6"

	// Ascending order: the first apply grows the document by 2 lines, so
	// the second suggestion is only correct if ApplyAllSuggestions
	// recalculates its range between applications.
	first := NewSuggestion("alice", 2, 2, "expand", "line2", "line2a\nline2b\nline2c")
	second := NewSuggestion("bob", 5, 5, "fix", "line5", "LINE5")

	result, err := ApplyAllSuggestions(content, []*Comment{first, second})
	if err != nil {
		t.Fatalf("ApplyAllSuggestions failed: %v", err)
	}

	expected := "line1\nline2a\nline2b\nline2c\nline3\nline4\nLINE5\nline6"
	if result != expected {
		t.Errorf("Result mismatch.\nExpected:\n%s\nGot:\n%s", expected, result)
	}

	// The second suggestion's range was updated to match the new content
	if second.StartLine != 7 || second.EndLine != 7 {
		t.Errorf("second suggestion range = %d-%d, want 7-7", second.StartLine, second.EndLine)
	}
}

func TestApplyAllSuggestionsBottomUpStillWorks(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6"

	lower := NewSuggestion("bob", 5, 5, "fix", "line5", "LINE5")
	upper := NewSuggestion("alice", 2, 2, "expand", "line2", "line2a\nline2b\nline2c")

	// Old contract: bottom-to-top ordering must keep working
	result, err := ApplyAllSuggestions(content, []*Comment{lower, upper})
	if err != nil {
		t.Fatalf("ApplyAllSuggestions failed: %v", err)
	}

	expected := "line1\nline2a\nline2b\nline2c\nline3\nline4\nLINE5\nline6"
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
