package comment

import "strings"

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
// One thing a declared move may not do: carry a thread OUT of a template's
// `zone: human` section when an agent is calling. The resolve guard keys on
// where a thread sits, so an agent that could choose that could move a
// human-decision thread into its own zone and then close it.
//
// The caller is responsible for persisting doc afterwards.
func ApplyMoves(doc *DocumentWithComments, absPath string, moves []Move, actor Actor) []MoveResult {
	var (
		zones    *Template
		zonesErr error
	)
	if actor == ActorAgent {
		zones, _, zonesErr = ResolveTemplateForDocument(absPath, doc.Content, "", doc.Template)
	}
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

		if actor == ActorAgent {
			if reason := refuseZoneExit(doc, zones, zonesErr, c, line); reason != "" {
				results = append(results, MoveResult{CommentID: move.CommentID, Error: reason})
				continue
			}
		}

		if c.OriginalLine == 0 {
			c.OriginalLine = c.Line
		}
		// A suggestion's range travels with it, keeping its length: left
		// behind, the human's accept would replace whatever now sits on the
		// old lines.
		if c.IsSuggestion && c.StartLine > 0 {
			c.EndLine = line + (c.EndLine - c.StartLine)
			c.StartLine = line
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

// refuseZoneExit returns why an agent may not make this move, or "".
func refuseZoneExit(doc *DocumentWithComments, zones *Template, zonesErr error, c *Comment, newLine int) string {
	if zonesErr != nil {
		// Fail closed, like the resolve guard: unknown zones are not "no zones".
		return "the document's template could not be loaded, so its human-decision zones are unknown; leave this thread for the human"
	}
	if zones == nil || !inHumanZone(doc.Content, zones, c) {
		return ""
	}
	if SectionZone(doc.Content, zones, newLine) == ZoneHuman {
		return ""
	}
	return "this thread is in a human-decision zone and cannot be moved out of it; reply with where it belongs and leave the move to the human"
}

// inHumanZone reports whether a comment belongs to a human-decision section, by
// its current line OR its recorded section path. The path matters after an
// edit: the line may already point somewhere else, and the agent made the edit.
func inHumanZone(content string, t *Template, c *Comment) bool {
	if SectionZone(content, t, c.Line) == ZoneHuman {
		return true
	}
	for _, part := range strings.Split(c.SectionPath, " > ") {
		for _, ts := range t.Sections {
			if ts.Zone == ZoneHuman && strings.EqualFold(strings.TrimSpace(part), ts.Heading) {
				return true
			}
		}
	}
	return false
}
