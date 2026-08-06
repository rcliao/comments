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

// PreviewSuggestion shows what the suggestion would change without applying it
// Returns a simple diff-like output
func PreviewSuggestion(content string, suggestion *Comment) (string, error) {
	if !suggestion.IsSuggestion {
		return "", fmt.Errorf("comment is not a suggestion")
	}

	lines := strings.Split(content, "\n")

	if suggestion.StartLine < 1 || suggestion.StartLine > len(lines) {
		return "", fmt.Errorf("invalid line range")
	}

	if suggestion.EndLine > len(lines) {
		return "", fmt.Errorf("invalid line range")
	}

	var preview strings.Builder

	preview.WriteString("=== Suggestion Preview ===\n\n")
	fmt.Fprintf(&preview, "Lines %d-%d\n\n", suggestion.StartLine, suggestion.EndLine)

	// Show original
	preview.WriteString("--- Original\n")
	for i := suggestion.StartLine - 1; i < suggestion.EndLine; i++ {
		fmt.Fprintf(&preview, "- %s\n", lines[i])
	}

	// Show proposed
	preview.WriteString("\n+++ Proposed\n")
	proposedLines := strings.Split(suggestion.ProposedText, "\n")
	for _, line := range proposedLines {
		fmt.Fprintf(&preview, "+ %s\n", line)
	}

	return preview.String(), nil
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
