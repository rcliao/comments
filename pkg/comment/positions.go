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
