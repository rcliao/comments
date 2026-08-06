package comment

import (
	"fmt"
	"strings"
)

// ApplySuggestion applies a multi-line suggestion to the document content (v2.0)
// Returns the modified content or an error if the suggestion cannot be applied
func ApplySuggestion(content string, suggestion *Comment) (string, error) {
	if !suggestion.IsSuggestion {
		return "", fmt.Errorf("comment is not a suggestion")
	}

	if suggestion.StartLine < 1 {
		return "", fmt.Errorf("invalid start line: %d", suggestion.StartLine)
	}

	if suggestion.EndLine < suggestion.StartLine {
		return "", fmt.Errorf("end line %d cannot be before start line %d", suggestion.EndLine, suggestion.StartLine)
	}

	lines := strings.Split(content, "\n")

	// Validate line range
	if suggestion.StartLine > len(lines) {
		return "", fmt.Errorf("start line %d out of range (1-%d)", suggestion.StartLine, len(lines))
	}

	if suggestion.EndLine > len(lines) {
		return "", fmt.Errorf("end line %d out of range (1-%d)", suggestion.EndLine, len(lines))
	}

	// Extract original text for verification
	var originalLines []string
	for i := suggestion.StartLine - 1; i < suggestion.EndLine; i++ {
		originalLines = append(originalLines, lines[i])
	}
	actualOriginal := strings.Join(originalLines, "\n")

	// Verify original text matches (if provided)
	if suggestion.OriginalText != "" && actualOriginal != suggestion.OriginalText {
		return "", fmt.Errorf("original text mismatch:\nExpected:\n%s\n\nGot:\n%s",
			suggestion.OriginalText, actualOriginal)
	}

	// Build new content
	var result []string

	// Lines before the change
	result = append(result, lines[:suggestion.StartLine-1]...)

	// Insert proposed text (may be multiple lines)
	if suggestion.ProposedText != "" {
		proposedLines := strings.Split(suggestion.ProposedText, "\n")
		result = append(result, proposedLines...)
	}

	// Lines after the change
	if suggestion.EndLine < len(lines) {
		result = append(result, lines[suggestion.EndLine:]...)
	}

	return strings.Join(result, "\n"), nil
}

// SuggestionLinesAdded returns the number of lines the suggestion's proposed
// text inserts in place of [StartLine, EndLine]. An empty ProposedText is a
// pure deletion and inserts 0 lines. Use this as the linesAdded argument to
// RecalculateCommentLines after applying a suggestion.
func SuggestionLinesAdded(suggestion *Comment) int {
	if suggestion.ProposedText == "" {
		return 0
	}
	return len(strings.Split(suggestion.ProposedText, "\n"))
}

// ApplyAndAcceptSuggestion is the single accept implementation shared by the
// CLI, TUI, and MCP server. It finds the pending suggestion by ID, applies it
// to doc.Content, marks it accepted, shifts every comment and pending
// suggestion below the edited range via RecalculateCommentLines, and refreshes
// section metadata for the changed content.
//
// Returns (true, nil) when the document was modified. Accepting an
// already-accepted suggestion is an idempotent no-op: (false, nil). A missing
// ID, a non-suggestion comment, a previously rejected suggestion, or an
// OriginalText mismatch returns (false, err) with doc untouched.
func ApplyAndAcceptSuggestion(doc *DocumentWithComments, suggestionID string) (bool, error) {
	suggestion := doc.FindCommentByID(suggestionID)
	if suggestion == nil {
		return false, fmt.Errorf("suggestion not found: %s", suggestionID)
	}
	if !suggestion.IsSuggestion {
		return false, fmt.Errorf("comment is not a suggestion: %s", suggestionID)
	}
	if suggestion.IsAccepted() {
		return false, nil // already applied; keep accepts idempotent
	}
	if suggestion.IsRejected() {
		return false, fmt.Errorf("suggestion already rejected: %s", suggestionID)
	}

	newContent, err := ApplySuggestion(doc.Content, suggestion)
	if err != nil {
		return false, err
	}

	if err := AcceptSuggestion(doc.Threads, suggestionID); err != nil {
		return false, err
	}

	doc.Content = newContent
	RecalculateCommentLines(doc.Threads, suggestion.StartLine, suggestion.EndLine,
		SuggestionLinesAdded(suggestion))
	ComputeSectionsForComments(doc)
	return true, nil
}

// ApplyAllSuggestions applies multiple suggestions to the document in the
// order given. After each successful application, the positions of the
// remaining (not yet applied) suggestions are recalculated via
// RecalculateCommentLines, so callers no longer need to pre-sort
// suggestions bottom-to-top. Note that this mutates the StartLine/EndLine
// of the passed suggestions to keep them consistent with the returned
// content.
func ApplyAllSuggestions(content string, suggestions []*Comment) (string, error) {
	result := content
	var err error

	// Apply each suggestion
	for i, suggestion := range suggestions {
		result, err = ApplySuggestion(result, suggestion)
		if err != nil {
			return "", fmt.Errorf("failed to apply suggestion %s: %w", suggestion.ID, err)
		}
		// Shift the ranges of the suggestions not yet applied so they still
		// target the right lines after this edit changed the line count.
		if rest := suggestions[i+1:]; len(rest) > 0 {
			RecalculateCommentLines(rest, suggestion.StartLine, suggestion.EndLine, SuggestionLinesAdded(suggestion))
		}
	}

	return result, nil
}
