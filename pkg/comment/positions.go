package comment

// Package positions provides utilities for position tracking in v2.0
// In v2.0, we only track line numbers (no column/byte offset complexity)

// RecalculateCommentLines updates comment line numbers after a document edit
// editStartLine: first line affected by edit (1-indexed)
// editEndLine: last line affected by edit (inclusive, 1-indexed)
// linesAdded: number of lines added by the edit (can be negative for deletions)
func RecalculateCommentLines(comments []*Comment, editStartLine, editEndLine, linesAdded int) {
	linesDeleted := editEndLine - editStartLine + 1
	delta := linesAdded - linesDeleted

	for _, comment := range comments {
		if comment.Line > editEndLine {
			// Comment is after the edit - shift by delta
			comment.Line += delta
		} else if comment.Line >= editStartLine && comment.Line <= editEndLine {
			// Comment is within the edited range - move to start of edit
			comment.Line = editStartLine
		}
		// Comments before the edit remain unchanged

		// Recursively update replies
		if len(comment.Replies) > 0 {
			RecalculateCommentLines(comment.Replies, editStartLine, editEndLine, linesAdded)
		}
	}
}

// SortSuggestionsByLine sorts suggestions by line number in descending order (bottom to top)
// This order is optimal for applying multiple suggestions without position drift
func SortSuggestionsByLine(suggestions []*Comment) {
	// Simple bubble sort for descending order
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].StartLine > suggestions[i].StartLine {
				// Swap to put higher line number first
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
}

// GetAffectedComments returns comments that would be affected by an edit
// This is useful for warning users about potential conflicts
func GetAffectedComments(comments []*Comment, editStartLine, editEndLine int) []*Comment {
	affected := []*Comment{}

	for _, comment := range comments {
		if comment.Line >= editStartLine && comment.Line <= editEndLine {
			affected = append(affected, comment)
		}

		// Check replies recursively
		if len(comment.Replies) > 0 {
			affected = append(affected, GetAffectedComments(comment.Replies, editStartLine, editEndLine)...)
		}
	}

	return affected
}

// ConflictType represents the type of conflict between suggestions
type ConflictType string

const (
	ConflictNone     ConflictType = ""          // No conflict
	ConflictOverlap  ConflictType = "overlap"   // Suggestions overlap
	ConflictAdjacent ConflictType = "adjacent"  // Suggestions are adjacent
	ConflictNested   ConflictType = "nested"    // One suggestion contains another
)

// Conflict represents a detected conflict between two suggestions
type Conflict struct {
	Type        ConflictType
	Suggestion1 *Comment
	Suggestion2 *Comment
	Description string
}

// DetectConflicts identifies conflicts between pending suggestions
// Returns a list of conflicts that need user attention
func DetectConflicts(suggestions []*Comment) []Conflict {
	conflicts := []Conflict{}

	// Compare each pair of suggestions
	for i := 0; i < len(suggestions); i++ {
		s1 := suggestions[i]
		if !s1.IsSuggestion || !s1.IsPending() {
			continue
		}

		for j := i + 1; j < len(suggestions); j++ {
			s2 := suggestions[j]
			if !s2.IsSuggestion || !s2.IsPending() {
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
	s1Start := s1.StartLine
	s1End := s1.EndLine

	s2Start := s2.StartLine
	s2End := s2.EndLine

	// Check for overlap
	if s1Start <= s2End && s1End >= s2Start {
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

// HasConflicts returns true if any serious conflicts exist in the list
func HasConflicts(conflicts []Conflict) bool {
	for _, c := range conflicts {
		if c.Type == ConflictOverlap || c.Type == ConflictNested {
			return true
		}
	}
	return false
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
		if conflict.Type == ConflictOverlap || conflict.Type == ConflictNested {
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
