package mcp

import "github.com/rcliao/comments/pkg/comment"

// Tool request/response types for MCP operations

// GetRequest looks up one comment, or every thread when comment_id is empty.
type GetRequest struct {
	FilePath       string `json:"filepath,omitempty" jsonschema:"Path to the markdown file (omit when using cite)"`
	CommentID      string `json:"comment_id,omitempty" jsonschema:"Comment or reply ID; omit to list every thread in the document"`
	UnresolvedOnly bool   `json:"unresolved_only,omitempty" jsonschema:"When listing: leave out resolved threads"`
	Cite           string `json:"cite,omitempty" jsonschema:"A thread: citation exactly as written in a document, e.g. thread:research.md#c1abc"`
	From           string `json:"from,omitempty" jsonschema:"The document the citation was read in; relative cite paths resolve against it"`
}

// AddRequest adds one or many root comments. One comment is a batch of one.
type AddRequest struct {
	FilePath string                   `json:"filepath" jsonschema:"Path to the markdown file"`
	Comments []comment.NewCommentSpec `json:"comments" jsonschema:"Comments to add, atomically: if any one is invalid none are added and the error names it"`
}

// ReplyRequest replies to — and optionally resolves — one or many threads.
type ReplyRequest struct {
	FilePath string              `json:"filepath" jsonschema:"Path to the markdown file"`
	Replies  []comment.ReplySpec `json:"replies" jsonschema:"Replies to post, atomically"`
}

// SuggestRequest represents a request to create an edit suggestion
type SuggestRequest struct {
	FilePath     string `json:"filepath" jsonschema:"Path to the markdown file"`
	Author       string `json:"author" jsonschema:"Author of the suggestion"`
	Text         string `json:"text" jsonschema:"Description of the suggestion"`
	StartLine    int    `json:"start_line,omitempty" jsonschema:"Start line of the edit (or use anchor)"`
	EndLine      int    `json:"end_line,omitempty" jsonschema:"End line of the edit (or use anchor)"`
	Anchor       string `json:"anchor,omitempty" jsonschema:"Quote of the range's FIRST line (or unique substring); original_text's line count sets the end"`
	Section      string `json:"section,omitempty" jsonschema:"Replace a whole section, by its full path (alternative to a line range or an anchor)"`
	OriginalText string `json:"original_text,omitempty" jsonschema:"Original text being replaced (optional for verification)"`
	ProposedText string `json:"proposed_text" jsonschema:"Proposed replacement text"`
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
	Template string `json:"template,omitempty" jsonschema:"Template name (defaults to frontmatter, legacy sidecar, or bundle)"`
}

// AnalyzeRequest asks for deterministic artifact coverage. It is advisory and
// never mutates the review gate.
type AnalyzeRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to the markdown artifact"`
	Against  string `json:"against,omitempty" jsonschema:"Research document to check plan coverage against"`
	Template string `json:"template,omitempty" jsonschema:"Template name (defaults to frontmatter, legacy sidecar, or bundle)"`
}

// NewDocumentRequest creates a concept through the project's OKF bundle.
type NewDocumentRequest struct {
	Name        string `json:"name" jsonschema:"Lowercase document slug without .md"`
	Template    string `json:"template" jsonschema:"Template that selects the bundle collection and document shape"`
	Title       string `json:"title,omitempty" jsonschema:"Document title; defaults to the slug"`
	Description string `json:"description,omitempty" jsonschema:"One-sentence concept description"`
	From        string `json:"from,omitempty" jsonschema:"Related source document to record as informed_by"`
	BundlePath  string `json:"bundle_path,omitempty" jsonschema:"Path used to discover or initialize .comments/bundle.yaml; defaults to current directory"`
}

// ContextRequest retrieves the relevant OKF neighborhood for an agent role.
type ContextRequest struct {
	FilePath       string `json:"filepath" jsonschema:"Path to the current OKF concept"`
	For            string `json:"for,omitempty" jsonschema:"Role mode: drafting, review, coverage-scout, evidence-verifier, human-review, or implementation"`
	IncludeBody    bool   `json:"include_body,omitempty" jsonschema:"Include document bodies"`
	IncludeThreads bool   `json:"include_threads,omitempty" jsonschema:"Include review comment threads"`
}
