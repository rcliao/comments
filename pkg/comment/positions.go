package comment

// Package positions provides utilities for position tracking in v2.0
// In v2.0, we only track line numbers (no column/byte offset complexity)

// RecalculateCommentLines updates comment positions after a document edit.
// editStartLine: first line affected by edit (1-indexed)
// editEndLine: last line affected by edit (inclusive, 1-indexed)
// linesAdded: number of lines the edit inserted in place of the range
// (0 for a pure deletion; use SuggestionLinesAdded when the edit is an
// accepted suggestion)
//
// For every comment (recursing into replies):
//   - Line below the edited range shifts by the net delta
//   - Line inside the edited range snaps to editStartLine
//   - a suggestion range entirely below the edited range shifts by the
//     delta (both StartLine and EndLine), so pending suggestions still
//     target the right lines after an earlier suggestion above them is
//     accepted
//   - a suggestion range overlapping the edited range is left untouched:
//     it is now stale, and OriginalText verification in ApplySuggestion is
//     the guard against applying it to the wrong text
func RecalculateCommentLines(comments []*Comment, editStartLine, editEndLine, linesAdded int) {
	linesDeleted := editEndLine - editStartLine + 1
	delta := linesAdded - linesDeleted

	for _, comment := range comments {
		if comment.IsSuggestion && comment.StartLine > editEndLine {
			// Suggestion entirely below the edit - shift the whole range
			comment.StartLine += delta
			comment.EndLine += delta
		}

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
