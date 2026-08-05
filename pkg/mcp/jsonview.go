package mcp

import (
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// commentJSON is the canonical snake_case wire shape for comments over MCP,
// matching the CLI's `list --format json` field naming (schema unification, v2.1).
type commentJSON struct {
	ID               string        `json:"id"`
	Author           string        `json:"author"`
	Line             int           `json:"line"`
	Timestamp        string        `json:"timestamp"`
	Text             string        `json:"text"`
	Type             string        `json:"type,omitempty"`
	Status           string        `json:"status"`
	Priority         string        `json:"priority"`
	Blocking         bool          `json:"blocking"`
	Resolved         bool          `json:"resolved"`
	ReplyCount       int           `json:"reply_count"`
	SectionPath      string        `json:"section_path,omitempty"`
	OrphanedReason   string        `json:"orphaned_reason,omitempty"`
	AnchorConfidence string        `json:"anchor_confidence,omitempty"`
	IsSuggestion     bool          `json:"is_suggestion,omitempty"`
	StartLine        int           `json:"start_line,omitempty"`
	EndLine          int           `json:"end_line,omitempty"`
	OriginalText     string        `json:"original_text,omitempty"`
	ProposedText     string        `json:"proposed_text,omitempty"`
	Accepted         *bool         `json:"accepted,omitempty"`
	Replies          []commentJSON `json:"replies,omitempty"`
}

func toCommentJSON(c *comment.Comment) commentJSON {
	out := commentJSON{
		ID:               c.ID,
		Author:           c.Author,
		Line:             c.Line,
		Timestamp:        c.Timestamp.Format(time.RFC3339),
		Text:             c.Text,
		Type:             c.Type,
		Status:           c.GetStatus(),
		Priority:         c.GetPriority(),
		Blocking:         c.Blocking,
		Resolved:         c.Resolved,
		ReplyCount:       c.CountReplies(),
		SectionPath:      c.SectionPath,
		OrphanedReason:   c.OrphanedReason,
		AnchorConfidence: c.AnchorConfidence,
		IsSuggestion:     c.IsSuggestion,
		StartLine:        c.StartLine,
		EndLine:          c.EndLine,
		OriginalText:     c.OriginalText,
		ProposedText:     c.ProposedText,
		Accepted:         c.Accepted,
	}
	for _, r := range c.Replies {
		out.Replies = append(out.Replies, toCommentJSON(r))
	}
	return out
}

func toCommentJSONList(comments []*comment.Comment) []commentJSON {
	out := make([]commentJSON, 0, len(comments))
	for _, c := range comments {
		out = append(out, toCommentJSON(c))
	}
	return out
}
