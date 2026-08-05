package mcp


// Tool request/response types for MCP operations

// ListCommentsRequest represents a request to list/filter comments
type ListCommentsRequest struct {
	FilePath     string   `json:"filepath" jsonschema:"Path to the markdown file"`
	Author       string   `json:"author,omitempty" jsonschema:"Filter by author name"`
	Type         string   `json:"type,omitempty" jsonschema:"Filter by comment type (Q S B T E)"`
	Section      string   `json:"section,omitempty" jsonschema:"Filter by section path (e.g. 'Introduction > Overview')"`
	Search       string   `json:"search,omitempty" jsonschema:"Search in comment text"`
	Status       string   `json:"status,omitempty" jsonschema:"Filter by status (active orphaned resolved completed)"`
	Priority     string   `json:"priority,omitempty" jsonschema:"Filter by priority (low medium high)"`
	Resolved     *bool    `json:"resolved,omitempty" jsonschema:"Filter by resolved state"`
	LineStart    int      `json:"line_start,omitempty" jsonschema:"Filter comments from this line"`
	LineEnd      int      `json:"line_end,omitempty" jsonschema:"Filter comments up to this line"`
	WithContext  bool     `json:"with_context,omitempty" jsonschema:"Include surrounding context for each comment"`
}

// GetCommentRequest represents a request to get a specific comment with context
type GetCommentRequest struct {
	FilePath  string `json:"filepath" jsonschema:"Path to the markdown file"`
	CommentID string `json:"comment_id" jsonschema:"ID of the comment to retrieve"`
}

// StatusRequest represents a request to get document status
type StatusRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to the markdown file"`
}

// AddCommentRequest represents a request to add a root comment
type AddCommentRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to the markdown file"`
	Author   string `json:"author" jsonschema:"Author of the comment"`
	Text     string `json:"text" jsonschema:"Comment text"`
	Type     string `json:"type,omitempty" jsonschema:"Comment type: Q (Question) S (Suggestion) B (Bug) T (TODO) E (Enhancement)"`
	Line     int    `json:"line,omitempty" jsonschema:"Line number (use this OR section not both)"`
	Section  string `json:"section,omitempty" jsonschema:"Section path (e.g. 'Introduction > Overview')"`
	Status   string `json:"status,omitempty" jsonschema:"Status: active (default) resolved completed"`
	Priority string `json:"priority,omitempty" jsonschema:"Priority: low medium (default) high"`
	Blocking bool   `json:"blocking,omitempty" jsonschema:"If true this comment must be resolved before the review gate passes"`
}

// ReplyRequest represents a request to reply to a thread
type ReplyRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to the markdown file"`
	ThreadID string `json:"thread_id" jsonschema:"ID of the thread to reply to"`
	Author   string `json:"author" jsonschema:"Author of the reply"`
	Text     string `json:"text" jsonschema:"Reply text"`
}

// ResolveRequest represents a request to resolve/unresolve a thread
type ResolveRequest struct {
	FilePath   string `json:"filepath" jsonschema:"Path to the markdown file"`
	ThreadID   string `json:"thread_id" jsonschema:"ID of the thread to resolve"`
	Unresolve  bool   `json:"unresolve,omitempty" jsonschema:"Set to true to unresolve the thread"`
}

// SuggestRequest represents a request to create an edit suggestion
type SuggestRequest struct {
	FilePath     string `json:"filepath" jsonschema:"Path to the markdown file"`
	Author       string `json:"author" jsonschema:"Author of the suggestion"`
	Text         string `json:"text" jsonschema:"Description of the suggestion"`
	StartLine    int    `json:"start_line" jsonschema:"Start line of the edit"`
	EndLine      int    `json:"end_line" jsonschema:"End line of the edit"`
	OriginalText string `json:"original_text,omitempty" jsonschema:"Original text being replaced (optional for verification)"`
	ProposedText string `json:"proposed_text" jsonschema:"Proposed replacement text"`
}

// AcceptSuggestionRequest represents a request to accept a suggestion
type AcceptSuggestionRequest struct {
	FilePath     string `json:"filepath" jsonschema:"Path to the markdown file"`
	SuggestionID string `json:"suggestion_id" jsonschema:"ID of the suggestion to accept"`
	Preview      bool   `json:"preview,omitempty" jsonschema:"If true preview the change without applying it"`
}

// RejectSuggestionRequest represents a request to reject a suggestion
type RejectSuggestionRequest struct {
	FilePath     string `json:"filepath" jsonschema:"Path to the markdown file"`
	SuggestionID string `json:"suggestion_id" jsonschema:"ID of the suggestion to reject"`
}

// BatchAddRequest represents a request to add multiple comments
type BatchAddRequest struct {
	FilePath string                 `json:"filepath" jsonschema:"Path to the markdown file"`
	Comments []BatchCommentData     `json:"comments" jsonschema:"Array of comment objects to add"`
}

// BatchCommentData represents a single comment in a batch add operation
type BatchCommentData struct {
	Author       string `json:"author" jsonschema:"Author of the comment"`
	Text         string `json:"text" jsonschema:"Comment text"`
	Type         string `json:"type,omitempty" jsonschema:"Comment type (Q S B T E)"`
	Line         int    `json:"line,omitempty" jsonschema:"Line number"`
	Section      string `json:"section,omitempty" jsonschema:"Section path"`
	Status       string `json:"status,omitempty" jsonschema:"Status (active resolved completed)"`
	Priority     string `json:"priority,omitempty" jsonschema:"Priority (low medium high)"`
	Blocking     bool   `json:"blocking,omitempty" jsonschema:"If true this comment must be resolved before the review gate passes"`
	IsSuggestion bool   `json:"is_suggestion,omitempty" jsonschema:"True if this is an edit suggestion"`
	StartLine    int    `json:"start_line,omitempty" jsonschema:"Start line for suggestion"`
	EndLine      int    `json:"end_line,omitempty" jsonschema:"End line for suggestion"`
	OriginalText string `json:"original_text,omitempty" jsonschema:"Original text for suggestion"`
	ProposedText string `json:"proposed_text,omitempty" jsonschema:"Proposed text for suggestion"`
}

// BatchReplyRequest represents a request to add multiple replies
type BatchReplyRequest struct {
	FilePath string            `json:"filepath" jsonschema:"Path to the markdown file"`
	Replies  []BatchReplyData  `json:"replies" jsonschema:"Array of reply objects to add"`
}

// BatchReplyData represents a single reply in a batch reply operation
type BatchReplyData struct {
	ThreadID string `json:"thread_id" jsonschema:"ID of the thread to reply to"`
	Author   string `json:"author" jsonschema:"Author of the reply"`
	Text     string `json:"text" jsonschema:"Reply text"`
}

// GateRequest represents a request to evaluate the review gate
type GateRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to a markdown file or a directory of markdown files"`
	Strict   bool   `json:"strict,omitempty" jsonschema:"If true fail on any unresolved comment or pending suggestion not just blocking ones"`
}

// RequestReviewRequest represents a blocking request for human review
type RequestReviewRequest struct {
	FilePath       string `json:"filepath" jsonschema:"Path to the markdown file to be reviewed"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"Max seconds to wait for the human signoff (default 600)"`
	Strict         bool   `json:"strict,omitempty" jsonschema:"If true evaluate the gate with strict rules after signoff"`
}

// ReanchorMove relocates one comment to its new position after an agent edit
type ReanchorMove struct {
	CommentID string `json:"comment_id" jsonschema:"ID of the comment to move"`
	Line      int    `json:"line,omitempty" jsonschema:"New line number (use line OR section)"`
	Section   string `json:"section,omitempty" jsonschema:"New section path (use line OR section)"`
}

// ReanchorRequest migrates comment anchors after the agent edited the document.
// The editing agent knows how its edits moved text, so it migrates the anchors
// it displaced; the load-time cascade is only the safety net.
type ReanchorRequest struct {
	FilePath string         `json:"filepath" jsonschema:"Path to the markdown file"`
	Moves    []ReanchorMove `json:"moves" jsonschema:"Comments to relocate to their new lines/sections"`
}

// ValidateRequest represents a request to validate a document against a template
type ValidateRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to the markdown file"`
	Template string `json:"template,omitempty" jsonschema:"Template name (defaults to the template recorded in the sidecar)"`
}

// SeedRequest represents a request to seed template review threads
type SeedRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to the markdown file"`
	Template string `json:"template,omitempty" jsonschema:"Template name (defaults to the template recorded in the sidecar)"`
	Author   string `json:"author,omitempty" jsonschema:"Author for seeded threads (default 'template')"`
}

// GetTemplateRequest represents a request to read a template definition
type GetTemplateRequest struct {
	Name string `json:"name,omitempty" jsonschema:"Template name; omit to list available templates"`
}

// DocumentStatus represents the status of a document's comments
type DocumentStatus struct {
	FilePath            string                   `json:"filepath"`
	TotalThreads        int                      `json:"total_threads"`
	ResolvedThreads     int                      `json:"resolved_threads"`
	UnresolvedThreads   int                      `json:"unresolved_threads"`
	PendingSuggestions  int                      `json:"pending_suggestions"`
	OrphanedComments    int                      `json:"orphaned_comments"`
	IsStale             bool                     `json:"is_stale"`
	DocumentHash        string                   `json:"document_hash"`
	LastValidated       string                   `json:"last_validated"`
	SuggestionsByAuthor map[string]int           `json:"suggestions_by_author,omitempty"`
}

// CommentWithContext represents a comment with its surrounding context
type CommentWithContext struct {
	Comment      commentJSON `json:"comment"`
	SectionPath  string      `json:"section_path,omitempty"`
	ContextLines []string    `json:"context_lines,omitempty"`
	IsOrphaned   bool        `json:"is_orphaned,omitempty"`
}
