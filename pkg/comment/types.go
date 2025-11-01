package comment

import "time"

// SuggestionType represents the type of edit suggestion
type SuggestionType string

const (
	SuggestionNone      SuggestionType = ""            // Not a suggestion
	SuggestionLine      SuggestionType = "line"        // Replace entire line(s)
	SuggestionCharRange SuggestionType = "char-range"  // Replace character range
	SuggestionMultiLine SuggestionType = "multi-line"  // Replace multiple lines
	SuggestionDiffHunk  SuggestionType = "diff-hunk"   // Apply unified diff
)

// AcceptanceState represents the state of a suggestion
type AcceptanceState string

const (
	AcceptancePending  AcceptanceState = "pending"   // Awaiting review
	AcceptanceAccepted AcceptanceState = "accepted"  // Accepted and applied
	AcceptanceRejected AcceptanceState = "rejected"  // Rejected
)

// Selection represents the text range being edited by a suggestion
type Selection struct {
	StartLine  int    // Starting line number (1-indexed)
	EndLine    int    // Ending line number (inclusive, for multi-line)
	ByteOffset int    // Byte offset from start of document (for char-range)
	Length     int    // Length in bytes (for char-range)
	Original   string // Original text being replaced
}

// Comment represents a single comment in a document
type Comment struct {
	ID        string    // Unique identifier for the comment
	ThreadID  string    // Root comment ID (same as ID for root comments)
	ParentID  string    // ID of parent comment (empty for top-level)
	Author    string    // Author of the comment (user or LLM name)
	Line      int       // Original line number where comment was added
	Timestamp time.Time // When the comment was created
	Text      string    // Comment content
	Type      string    // Comment type: Q, S, B, T, E (optional)
	Resolved  bool      // Whether the comment/thread has been resolved

	// Suggestion-specific fields (optional, only for edit suggestions)
	SuggestionType  SuggestionType  // Type of suggestion (empty if not a suggestion)
	Selection       *Selection      // Text selection being edited (nil if not a suggestion)
	ProposedText    string          // Proposed replacement text (empty if not a suggestion)
	AcceptanceState AcceptanceState // Acceptance state (empty if not a suggestion)
}

// IsRoot returns true if this is a root comment (not a reply)
func (c *Comment) IsRoot() bool {
	return c.ParentID == ""
}

// IsReply returns true if this is a reply to another comment
func (c *Comment) IsReply() bool {
	return c.ParentID != ""
}

// IsSuggestion returns true if this comment is an edit suggestion
func (c *Comment) IsSuggestion() bool {
	return c.SuggestionType != SuggestionNone && c.SuggestionType != ""
}

// IsPending returns true if this suggestion is awaiting review
func (c *Comment) IsPending() bool {
	return c.IsSuggestion() && (c.AcceptanceState == AcceptancePending || c.AcceptanceState == "")
}

// IsAccepted returns true if this suggestion has been accepted
func (c *Comment) IsAccepted() bool {
	return c.IsSuggestion() && c.AcceptanceState == AcceptanceAccepted
}

// IsRejected returns true if this suggestion has been rejected
func (c *Comment) IsRejected() bool {
	return c.IsSuggestion() && c.AcceptanceState == AcceptanceRejected
}

// Position represents the location of a comment in a document
type Position struct {
	Line      int // Current line number (may change as doc is edited)
	Column    int // Column offset within the line
	ByteOffset int // Byte offset from start of document
}

// Thread represents a conversation thread
type Thread struct {
	ID          string     // Thread identifier (same as root comment ID)
	RootComment *Comment   // The root comment that started the thread
	Replies     []*Comment // All replies in the thread, ordered by timestamp
	Resolved    bool       // Whether the thread has been resolved
	Line        int        // Line number where the thread is anchored
}

// Count returns the total number of comments in the thread (root + replies)
func (t *Thread) Count() int {
	return 1 + len(t.Replies)
}

// LatestTimestamp returns the timestamp of the most recent comment
func (t *Thread) LatestTimestamp() time.Time {
	if len(t.Replies) == 0 {
		return t.RootComment.Timestamp
	}
	return t.Replies[len(t.Replies)-1].Timestamp
}

// DocumentWithComments represents a parsed document
type DocumentWithComments struct {
	Content  string               // Raw markdown content without comment markup
	Comments []*Comment           // All comments extracted from the document
	Positions map[string]Position // Map comment IDs to their positions
}
