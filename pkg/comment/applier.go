package comment

import (
	"bytes"
	"fmt"
	"strings"
)

// ApplySuggestion applies a suggestion to the document content
// Returns the modified content or an error if the suggestion cannot be applied
func ApplySuggestion(content string, suggestion *Comment) (string, error) {
	if !suggestion.IsSuggestion() {
		return "", fmt.Errorf("comment is not a suggestion")
	}

	if suggestion.Selection == nil {
		return "", fmt.Errorf("suggestion has no selection")
	}

	switch suggestion.SuggestionType {
	case SuggestionLine:
		return applyLineSuggestion(content, suggestion)
	case SuggestionCharRange:
		return applyCharRangeSuggestion(content, suggestion)
	case SuggestionMultiLine:
		return applyMultiLineSuggestion(content, suggestion)
	case SuggestionDiffHunk:
		return applyDiffHunkSuggestion(content, suggestion)
	default:
		return "", fmt.Errorf("unknown suggestion type: %s", suggestion.SuggestionType)
	}
}

// applyLineSuggestion replaces entire line(s)
func applyLineSuggestion(content string, suggestion *Comment) (string, error) {
	lines := strings.Split(content, "\n")
	sel := suggestion.Selection

	if sel.StartLine < 1 || sel.StartLine > len(lines) {
		return "", fmt.Errorf("start line %d out of range (1-%d)", sel.StartLine, len(lines))
	}

	// For line suggestions, EndLine defaults to StartLine if not set
	endLine := sel.EndLine
	if endLine == 0 {
		endLine = sel.StartLine
	}

	if endLine < sel.StartLine || endLine > len(lines) {
		return "", fmt.Errorf("end line %d out of range (%d-%d)", endLine, sel.StartLine, len(lines))
	}

	// Verify original text matches (for safety)
	if sel.Original != "" {
		var originalLines []string
		for i := sel.StartLine - 1; i < endLine; i++ {
			originalLines = append(originalLines, lines[i])
		}
		originalText := strings.Join(originalLines, "\n")
		if originalText != sel.Original {
			return "", fmt.Errorf("original text mismatch: expected %q, got %q", sel.Original, originalText)
		}
	}

	// Build new content
	var result []string

	// Lines before the change
	result = append(result, lines[:sel.StartLine-1]...)

	// Insert proposed text (may be multiple lines)
	if suggestion.ProposedText != "" {
		proposedLines := strings.Split(suggestion.ProposedText, "\n")
		result = append(result, proposedLines...)
	}

	// Lines after the change
	if endLine < len(lines) {
		result = append(result, lines[endLine:]...)
	}

	return strings.Join(result, "\n"), nil
}

// applyCharRangeSuggestion replaces a character range using byte offsets
func applyCharRangeSuggestion(content string, suggestion *Comment) (string, error) {
	sel := suggestion.Selection
	contentBytes := []byte(content)

	if sel.ByteOffset < 0 || sel.ByteOffset > len(contentBytes) {
		return "", fmt.Errorf("byte offset %d out of range (0-%d)", sel.ByteOffset, len(contentBytes))
	}

	endOffset := sel.ByteOffset + sel.Length
	if endOffset > len(contentBytes) {
		return "", fmt.Errorf("end offset %d out of range (0-%d)", endOffset, len(contentBytes))
	}

	// Verify original text matches
	if sel.Original != "" {
		originalText := string(contentBytes[sel.ByteOffset:endOffset])
		if originalText != sel.Original {
			return "", fmt.Errorf("original text mismatch at offset %d", sel.ByteOffset)
		}
	}

	// Build new content
	var result bytes.Buffer
	result.Write(contentBytes[:sel.ByteOffset])
	result.WriteString(suggestion.ProposedText)
	result.Write(contentBytes[endOffset:])

	return result.String(), nil
}

// applyMultiLineSuggestion replaces multiple complete lines
func applyMultiLineSuggestion(content string, suggestion *Comment) (string, error) {
	lines := strings.Split(content, "\n")
	sel := suggestion.Selection

	if sel.StartLine < 1 || sel.StartLine > len(lines) {
		return "", fmt.Errorf("start line %d out of range (1-%d)", sel.StartLine, len(lines))
	}

	if sel.EndLine < sel.StartLine || sel.EndLine > len(lines) {
		return "", fmt.Errorf("end line %d out of range (%d-%d)", sel.EndLine, sel.StartLine, len(lines))
	}

	// Verify original text matches
	if sel.Original != "" {
		var originalLines []string
		for i := sel.StartLine - 1; i < sel.EndLine; i++ {
			originalLines = append(originalLines, lines[i])
		}
		originalText := strings.Join(originalLines, "\n")
		if originalText != sel.Original {
			return "", fmt.Errorf("original text mismatch")
		}
	}

	// Build new content
	var result []string

	// Lines before
	result = append(result, lines[:sel.StartLine-1]...)

	// Proposed lines
	if suggestion.ProposedText != "" {
		proposedLines := strings.Split(suggestion.ProposedText, "\n")
		result = append(result, proposedLines...)
	}

	// Lines after
	if sel.EndLine < len(lines) {
		result = append(result, lines[sel.EndLine:]...)
	}

	return strings.Join(result, "\n"), nil
}

// applyDiffHunkSuggestion applies a unified diff patch
func applyDiffHunkSuggestion(content string, suggestion *Comment) (string, error) {
	// For now, parse a simple unified diff format
	// ProposedText should contain the diff hunk
	if suggestion.ProposedText == "" {
		return "", fmt.Errorf("diff hunk is empty")
	}

	// Parse the diff hunk to extract line changes
	// Format: @@ -startLine,count +startLine,count @@
	// Followed by lines prefixed with ' ', '-', or '+'

	lines := strings.Split(content, "\n")
	diffLines := strings.Split(suggestion.ProposedText, "\n")

	if len(diffLines) == 0 {
		return "", fmt.Errorf("invalid diff format")
	}

	// Parse the hunk header
	header := diffLines[0]
	if !strings.HasPrefix(header, "@@") {
		return "", fmt.Errorf("invalid diff header: %s", header)
	}

	// Extract start line from header
	// Simple parser for @@ -X,Y +A,B @@ or @@ -X +A @@
	var oldStart, oldCount, newStart, newCount int

	// Try full format first: @@ -X,Y +A,B @@
	n, err := fmt.Sscanf(header, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
	if err != nil || n < 4 {
		// Try with only start lines: @@ -X +A @@
		n, err = fmt.Sscanf(header, "@@ -%d +%d @@", &oldStart, &newStart)
		if err != nil || n < 2 {
			return "", fmt.Errorf("failed to parse diff header: %s", header)
		}
	}

	// Apply the diff
	var result []string
	lineIdx := 0
	diffIdx := 1 // Skip header

	// Copy lines before the diff
	for lineIdx < oldStart-1 && lineIdx < len(lines) {
		result = append(result, lines[lineIdx])
		lineIdx++
	}

	// Apply the diff hunk
	for diffIdx < len(diffLines) {
		diffLine := diffLines[diffIdx]
		if len(diffLine) == 0 {
			diffIdx++
			continue
		}

		prefix := diffLine[0]
		text := ""
		if len(diffLine) > 1 {
			text = diffLine[1:]
		}

		switch prefix {
		case ' ': // Context line (unchanged)
			if lineIdx < len(lines) {
				result = append(result, lines[lineIdx])
				lineIdx++
			}
		case '-': // Deleted line
			// Skip this line in original
			lineIdx++
		case '+': // Added line
			result = append(result, text)
		}

		diffIdx++
	}

	// Copy remaining lines
	for lineIdx < len(lines) {
		result = append(result, lines[lineIdx])
		lineIdx++
	}

	return strings.Join(result, "\n"), nil
}

// CanApplySuggestion checks if a suggestion can be applied without actually applying it
// Returns true if the suggestion is valid and can be applied
func CanApplySuggestion(content string, suggestion *Comment) error {
	_, err := ApplySuggestion(content, suggestion)
	return err
}

// ApplyMultipleSuggestions applies multiple suggestions in order
// Suggestions should be sorted by position (bottom to top) to avoid position drift
// Returns the modified content and a list of successfully applied suggestion IDs
func ApplyMultipleSuggestions(content string, suggestions []*Comment) (string, []string, error) {
	applied := []string{}
	current := content

	for _, sug := range suggestions {
		if !sug.IsSuggestion() || !sug.IsPending() {
			continue // Skip non-suggestions or non-pending
		}

		newContent, err := ApplySuggestion(current, sug)
		if err != nil {
			return current, applied, fmt.Errorf("failed to apply suggestion %s: %w", sug.ID, err)
		}

		current = newContent
		applied = append(applied, sug.ID)
	}

	return current, applied, nil
}
