package comment

import "time"

// CommentView is the canonical snake_case JSON shape for a comment, shared by
// every serializer in the project: MCP tools, MCP resources, CLI
// `list --format json`, and `gate --json`. Timestamps are RFC3339.
//
// The exact field-name set is pinned by TestCommentViewFieldNames — extend
// deliberately, never fork a new shape.
type CommentView struct {
	ID               string `json:"id"`
	Author           string `json:"author"`
	Line             int    `json:"line"`
	Timestamp        string `json:"timestamp"`
	Text             string `json:"text"`
	Type             string `json:"type,omitempty"`
	Status           string `json:"status"`
	Priority         string `json:"priority"`
	Blocking         bool   `json:"blocking"`
	Resolved         bool   `json:"resolved"`
	ReplyCount       int    `json:"reply_count"`
	SectionPath      string `json:"section_path,omitempty"`
	OrphanedReason   string `json:"orphaned_reason,omitempty"`
	AnchorConfidence string `json:"anchor_confidence,omitempty"`
	IsSuggestion     bool   `json:"is_suggestion,omitempty"`
	StartLine        int    `json:"start_line,omitempty"`
	EndLine          int    `json:"end_line,omitempty"`
	OriginalText     string `json:"original_text,omitempty"`
	ProposedText     string `json:"proposed_text,omitempty"`
	Accepted         *bool  `json:"accepted,omitempty"`
	// ParentThreadID is set on nested replies: the ROOT thread's ID, i.e. the
	// ID the write path (reply/resolve) addresses. Lets consumers that flatten
	// the tree keep a route back to the thread.
	ParentThreadID string        `json:"parent_thread_id,omitempty"`
	Replies        []CommentView `json:"replies,omitempty"`
}

// NewCommentView builds the canonical JSON view of a comment, recursing into
// nested replies. Nested replies carry parent_thread_id = the root thread's
// ID (the ID the write path addresses).
func NewCommentView(c *Comment) CommentView {
	return newCommentViewIn(c, "")
}

func newCommentViewIn(c *Comment, threadID string) CommentView {
	out := CommentView{
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
	out.ParentThreadID = threadID
	rootID := threadID
	if rootID == "" {
		rootID = c.ID
	}
	for _, r := range c.Replies {
		out.Replies = append(out.Replies, newCommentViewIn(r, rootID))
	}
	return out
}

// NewCommentViews builds canonical views for a list of comments. The result is
// never nil, so it marshals as [] rather than null.
func NewCommentViews(comments []*Comment) []CommentView {
	out := make([]CommentView, 0, len(comments))
	for _, c := range comments {
		out = append(out, NewCommentView(c))
	}
	return out
}

// DocumentView is the canonical snake_case JSON shape for a document plus its
// comment threads — used by the MCP document resource and any full-doc dumps.
type DocumentView struct {
	Content       string         `json:"content"`
	DocumentHash  string         `json:"document_hash"`
	LastValidated string         `json:"last_validated,omitempty"`
	Template      string         `json:"template,omitempty"`
	Reviews       []ReviewRecord `json:"reviews,omitempty"`
	Threads       []CommentView  `json:"threads"`
}

// NewDocumentView builds the canonical JSON view of a document with comments.
func NewDocumentView(d *DocumentWithComments) DocumentView {
	view := DocumentView{
		Content:      d.Content,
		DocumentHash: d.DocumentHash,
		Template:     d.Template,
		Reviews:      d.Reviews,
		Threads:      NewCommentViews(d.Threads),
	}
	if !d.LastValidated.IsZero() {
		view.LastValidated = d.LastValidated.Format(time.RFC3339)
	}
	return view
}
