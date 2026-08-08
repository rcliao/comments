package comment

// Move is a caller-declared anchor migration: the editor knows how its edits
// displaced text, so it names the comment and where the comment now belongs.
// Line wins when both Line and Section are set.
type Move struct {
	CommentID string `json:"comment_id"`
	Line      int    `json:"line,omitempty"`
	Section   string `json:"section,omitempty"`
}

// MoveResult reports the outcome of one Move. Failures are per-move rather
// than fatal so one bad ID cannot discard a whole batch of good migrations.
type MoveResult struct {
	CommentID   string `json:"comment_id"`
	Moved       bool   `json:"moved"`
	Line        int    `json:"line,omitempty"`
	SectionPath string `json:"section_path,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ApplyMoves migrates comment anchors after a document edit. Caller-declared
// moves are ground truth: the stored anchor is dropped and re-captured at the
// new position, and a successful migration un-orphans the comment.
//
// The caller is responsible for persisting doc afterwards.
func ApplyMoves(doc *DocumentWithComments, moves []Move) []MoveResult {
	results := make([]MoveResult, 0, len(moves))
	for _, move := range moves {
		c := doc.FindCommentByID(move.CommentID)
		if c == nil {
			results = append(results, MoveResult{
				CommentID: move.CommentID, Error: "comment not found",
			})
			continue
		}

		line := move.Line
		if move.Section != "" && line == 0 {
			startLine, _, err := ResolveSectionToLines(doc.Content, move.Section, false)
			if err != nil {
				results = append(results, MoveResult{
					CommentID: move.CommentID, Error: err.Error(),
				})
				continue
			}
			line = startLine
		}
		if line < 1 {
			results = append(results, MoveResult{
				CommentID: move.CommentID, Error: "move needs a line or section",
			})
			continue
		}

		if c.OriginalLine == 0 {
			c.OriginalLine = c.Line
		}
		c.Line = line
		// Re-capture the anchor at the new position; declared moves are ground truth
		c.Anchor = nil
		c.AnchorConfidence = ""
		UpdateCommentSection(c, doc.Content)
		// A successful migration un-orphans the comment
		if c.IsOrphaned() {
			c.Status = "active"
			c.OrphanedReason = ""
			c.OrphanedAt = nil
		}
		results = append(results, MoveResult{
			CommentID: move.CommentID, Moved: true, Line: line, SectionPath: c.SectionPath,
		})
	}
	return results
}
