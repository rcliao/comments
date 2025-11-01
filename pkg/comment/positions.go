package comment

import (
	"strings"
)

// RecalculatePositions updates all comment positions after a document edit
// This is necessary when applying suggestions or making other edits to the content
// oldContent: the content before the change
// newContent: the content after the change
// positions: the position map to update (modified in place)
func RecalculatePositions(oldContent, newContent string, positions map[string]Position) {
	if oldContent == newContent {
		return // No changes, positions remain valid
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Build a mapping of line changes using a simple diff algorithm
	lineMapping := buildLineMapping(oldLines, newLines)

	// Update each position based on the line mapping
	for id, pos := range positions {
		newLine, exists := lineMapping[pos.Line]
		if exists {
			positions[id] = Position{
				Line:       newLine,
				Column:     pos.Column,
				ByteOffset: CalculateByteOffset(newContent, newLine, pos.Column),
			}
		} else {
			// Line was deleted, map to nearest available line
			nearestLine := findNearestLine(pos.Line, lineMapping)
			positions[id] = Position{
				Line:       nearestLine,
				Column:     0,
				ByteOffset: CalculateByteOffset(newContent, nearestLine, 0),
			}
		}
	}
}

// RecalculatePositionsAfterEdit updates positions after a specific edit operation
// This is more efficient than full recalculation when we know exactly what changed
func RecalculatePositionsAfterEdit(editStartLine, editEndLine, newLineCount int, positions map[string]Position) {
	linesDeleted := editEndLine - editStartLine + 1
	linesAdded := newLineCount
	delta := linesAdded - linesDeleted

	// Update positions for comments after the edit
	for id, pos := range positions {
		if pos.Line > editEndLine {
			// Comment is after the edit - shift by delta
			positions[id] = Position{
				Line:       pos.Line + delta,
				Column:     pos.Column,
				ByteOffset: 0, // Will need recalculation from content
			}
		} else if pos.Line >= editStartLine && pos.Line <= editEndLine {
			// Comment is within the edited range
			// Map to the start of the new content
			positions[id] = Position{
				Line:       editStartLine,
				Column:     0,
				ByteOffset: 0,
			}
		}
		// Comments before the edit remain unchanged
	}
}

// buildLineMapping creates a mapping from old line numbers to new line numbers
// Uses a simple longest common subsequence (LCS) approach
func buildLineMapping(oldLines, newLines []string) map[int]int {
	mapping := make(map[int]int)

	// Simple approach: match identical lines in order
	oldIdx := 0
	newIdx := 0

	for oldIdx < len(oldLines) && newIdx < len(newLines) {
		if oldLines[oldIdx] == newLines[newIdx] {
			// Lines match - create mapping (1-indexed)
			mapping[oldIdx+1] = newIdx + 1
			oldIdx++
			newIdx++
		} else {
			// Lines don't match - try to find next match
			// Look ahead in new lines for current old line
			foundInNew := -1
			for i := newIdx + 1; i < min(newIdx+5, len(newLines)); i++ {
				if oldLines[oldIdx] == newLines[i] {
					foundInNew = i
					break
				}
			}

			// Look ahead in old lines for current new line
			foundInOld := -1
			for i := oldIdx + 1; i < min(oldIdx+5, len(oldLines)); i++ {
				if newLines[newIdx] == oldLines[i] {
					foundInOld = i
					break
				}
			}

			if foundInNew >= 0 {
				// New content was inserted - skip new lines until match
				newIdx = foundInNew
			} else if foundInOld >= 0 {
				// Old content was deleted - skip old lines until match
				oldIdx = foundInOld
			} else {
				// No match found nearby - advance both
				oldIdx++
				newIdx++
			}
		}
	}

	return mapping
}

// findNearestLine finds the nearest valid line for a deleted line
func findNearestLine(deletedLine int, mapping map[int]int) int {
	// Try lines before the deleted line
	for line := deletedLine - 1; line > 0; line-- {
		if newLine, exists := mapping[line]; exists {
			return newLine + 1 // Return next line after the mapped line
		}
	}

	// Try lines after the deleted line
	for line := deletedLine + 1; line < deletedLine+100; line++ {
		if newLine, exists := mapping[line]; exists {
			return newLine
		}
	}

	// Default to line 1 if no mapping found
	return 1
}

// CalculateByteOffset calculates the byte offset for a given line and column
func CalculateByteOffset(content string, line, column int) int {
	if line < 1 {
		return 0
	}

	lines := strings.Split(content, "\n")
	if line > len(lines) {
		return len(content)
	}

	offset := 0
	// Add bytes for all lines before target line
	for i := 0; i < line-1; i++ {
		offset += len(lines[i]) + 1 // +1 for newline
	}

	// Add column offset within target line
	if column > 0 && line <= len(lines) {
		lineContent := lines[line-1]
		if column <= len(lineContent) {
			offset += column
		} else {
			offset += len(lineContent)
		}
	}

	return offset
}

// UpdatePositionsByteOffsets recalculates byte offsets for all positions
// based on the current content. This should be called after any position changes.
func UpdatePositionsByteOffsets(content string, positions map[string]Position) {
	for id, pos := range positions {
		byteOffset := CalculateByteOffset(content, pos.Line, pos.Column)
		positions[id] = Position{
			Line:       pos.Line,
			Column:     pos.Column,
			ByteOffset: byteOffset,
		}
	}
}

// GetAffectedComments returns a list of comment IDs that would be affected by an edit
// This is useful for warning users about potential conflicts
func GetAffectedComments(positions map[string]Position, editStartLine, editEndLine int) []string {
	affected := []string{}

	for id, pos := range positions {
		if pos.Line >= editStartLine && pos.Line <= editEndLine {
			affected = append(affected, id)
		}
	}

	return affected
}

// SortSuggestionsByPosition sorts suggestions by their position (bottom to top)
// This order is optimal for applying multiple suggestions without position drift
func SortSuggestionsByPosition(suggestions []*Comment) {
	// Sort in descending order by line number (bottom to top)
	// This ensures that applying suggestions doesn't invalidate positions of later suggestions
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			// Compare by line, or by byte offset if on same line
			iLine := suggestions[i].Line
			jLine := suggestions[j].Line

			if suggestions[i].Selection != nil {
				iLine = suggestions[i].Selection.StartLine
			}
			if suggestions[j].Selection != nil {
				jLine = suggestions[j].Selection.StartLine
			}

			if jLine > iLine {
				// Swap to put higher line number first
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			} else if jLine == iLine {
				// Same line - sort by byte offset if available
				iOffset := 0
				jOffset := 0
				if suggestions[i].Selection != nil {
					iOffset = suggestions[i].Selection.ByteOffset
				}
				if suggestions[j].Selection != nil {
					jOffset = suggestions[j].Selection.ByteOffset
				}

				if jOffset > iOffset {
					suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
				}
			}
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ConflictType represents the type of conflict between suggestions
type ConflictType string

const (
	ConflictNone        ConflictType = ""           // No conflict
	ConflictOverlap     ConflictType = "overlap"    // Suggestions overlap
	ConflictAdjacent    ConflictType = "adjacent"   // Suggestions are adjacent (may affect each other)
	ConflictNested      ConflictType = "nested"     // One suggestion contains another
	ConflictSameLine    ConflictType = "same-line"  // Multiple suggestions on same line
)

// Conflict represents a detected conflict between two suggestions
type Conflict struct {
	Type         ConflictType
	Suggestion1  *Comment
	Suggestion2  *Comment
	Description  string
}

// DetectConflicts identifies conflicts between pending suggestions
// Returns a list of conflicts that need user attention
func DetectConflicts(suggestions []*Comment) []Conflict {
	conflicts := []Conflict{}

	// Compare each pair of suggestions
	for i := 0; i < len(suggestions); i++ {
		s1 := suggestions[i]
		if !s1.IsSuggestion() || !s1.IsPending() {
			continue
		}

		for j := i + 1; j < len(suggestions); j++ {
			s2 := suggestions[j]
			if !s2.IsSuggestion() || !s2.IsPending() {
				continue
			}

			conflict := detectConflictBetween(s1, s2)
			if conflict.Type != ConflictNone {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// detectConflictBetween checks if two suggestions conflict
func detectConflictBetween(s1, s2 *Comment) Conflict {
	if s1.Selection == nil || s2.Selection == nil {
		return Conflict{Type: ConflictNone}
	}

	sel1 := s1.Selection
	sel2 := s2.Selection

	// For character-range suggestions, check byte offset overlap
	if s1.SuggestionType == SuggestionCharRange && s2.SuggestionType == SuggestionCharRange {
		s1Start := sel1.ByteOffset
		s1End := sel1.ByteOffset + sel1.Length
		s2Start := sel2.ByteOffset
		s2End := sel2.ByteOffset + sel2.Length

		// Check for overlap
		if (s1Start < s2End && s1End > s2Start) {
			return Conflict{
				Type:        ConflictOverlap,
				Suggestion1: s1,
				Suggestion2: s2,
				Description: "Character ranges overlap",
			}
		}

		// Check for adjacent (within 10 bytes)
		if abs(s1End-s2Start) <= 10 || abs(s2End-s1Start) <= 10 {
			return Conflict{
				Type:        ConflictAdjacent,
				Suggestion1: s1,
				Suggestion2: s2,
				Description: "Character ranges are very close",
			}
		}

		return Conflict{Type: ConflictNone}
	}

	// For line-based suggestions, check line number overlap
	s1Start := sel1.StartLine
	s1End := sel1.EndLine
	if s1End == 0 {
		s1End = s1Start
	}

	s2Start := sel2.StartLine
	s2End := sel2.EndLine
	if s2End == 0 {
		s2End = s2Start
	}

	// Check for overlap
	if s1Start <= s2End && s1End >= s2Start {
		// Check if same line first (before nested check)
		if s1Start == s2Start && s1End == s2End {
			return Conflict{
				Type:        ConflictSameLine,
				Suggestion1: s1,
				Suggestion2: s2,
				Description: "Multiple suggestions on same line(s)",
			}
		}

		// Check if one is nested within the other
		if (s1Start <= s2Start && s1End >= s2End) || (s2Start <= s1Start && s2End >= s1End) {
			return Conflict{
				Type:        ConflictNested,
				Suggestion1: s1,
				Suggestion2: s2,
				Description: "One suggestion contains the other",
			}
		}

		return Conflict{
			Type:        ConflictOverlap,
			Suggestion1: s1,
			Suggestion2: s2,
			Description: "Line ranges overlap",
		}
	}

	// Check for adjacent lines (may affect each other)
	if abs(s1End-s2Start) <= 1 || abs(s2End-s1Start) <= 1 {
		return Conflict{
			Type:        ConflictAdjacent,
			Suggestion1: s1,
			Suggestion2: s2,
			Description: "Suggestions are on adjacent lines",
		}
	}

	return Conflict{Type: ConflictNone}
}

// HasConflicts returns true if any conflicts exist in the list
func HasConflicts(conflicts []Conflict) bool {
	for _, c := range conflicts {
		if c.Type == ConflictOverlap || c.Type == ConflictNested || c.Type == ConflictSameLine {
			return true
		}
	}
	return false
}

// GetConflictingSuggestions returns a map of suggestion IDs that have conflicts
func GetConflictingSuggestions(conflicts []Conflict) map[string]bool {
	conflicting := make(map[string]bool)

	for _, c := range conflicts {
		if c.Type == ConflictOverlap || c.Type == ConflictNested || c.Type == ConflictSameLine {
			if c.Suggestion1 != nil {
				conflicting[c.Suggestion1.ID] = true
			}
			if c.Suggestion2 != nil {
				conflicting[c.Suggestion2.ID] = true
			}
		}
	}

	return conflicting
}

// FilterNonConflicting removes suggestions that conflict with each other
// Keeps the first suggestion in each conflicting pair
func FilterNonConflicting(suggestions []*Comment) []*Comment {
	conflicts := DetectConflicts(suggestions)
	if len(conflicts) == 0 {
		return suggestions
	}

	// Build set of suggestions to exclude (later suggestions in conflicts)
	exclude := make(map[string]bool)
	for _, conflict := range conflicts {
		if conflict.Type == ConflictOverlap || conflict.Type == ConflictNested || conflict.Type == ConflictSameLine {
			// Keep first, exclude second
			if conflict.Suggestion2 != nil {
				exclude[conflict.Suggestion2.ID] = true
			}
		}
	}

	// Filter suggestions
	filtered := []*Comment{}
	for _, s := range suggestions {
		if !exclude[s.ID] {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
