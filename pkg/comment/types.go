package comment

import "time"

// Status is the lifecycle state of a comment. It is an alias for string (not a
// distinct type) so the JSON wire format and all existing string-based callers
// stay unchanged; use the Status* constants instead of raw literals.
type Status = string

// Comment status values (JSON wire values — do not change).
const (
	StatusActive    Status = "active"
	StatusOrphaned  Status = "orphaned"
	StatusResolved  Status = "resolved"
	StatusCompleted Status = "completed"
)

// Priority is the priority level of a comment. Alias for string for the same
// wire/API-compatibility reasons as Status.
type Priority = string

// Comment priority values (JSON wire values — do not change).
const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Comment represents a single comment or suggestion in a document (v2.0)
// Simplified structure with nested thread support
type Comment struct {
	// Identity
	ID        string    // Unique identifier for the comment
	Author    string    // Author of the comment (user or LLM name)
	Timestamp time.Time // When the comment was created

	// Content
	Text string // Comment content
	Type string // Comment type: Q, S, B, T, E (optional)

	// Position
	Line int // Line number where comment is attached

	// Section metadata (computed from document structure)
	SectionID   string // ID of the section this comment belongs to (e.g., "s1", "s2")
	SectionPath string // Full hierarchical path (e.g., "Intro > Overview > Key Points")

	// Content anchor (v2.1): what the comment points at, for re-anchoring after edits
	Anchor           *Anchor `json:",omitempty"` // Target line text + surrounding context
	AnchorConfidence string  `json:",omitempty"` // exact, fuzzy, or section-level (set by re-anchor cascade)

	// State
	Resolved bool // Whether the comment/thread has been resolved
	Blocking bool // If true, this thread must be resolved before the gate passes

	// Status tracking (for TODO/task management)
	Status         string     // Comment status: "active", "orphaned", "resolved", "completed"
	Priority       string     // Priority level: "low", "medium", "high" (default: "medium")
	OriginalLine   int        // Original line where comment was first attached (preserved for orphaned comments)
	OrphanedReason string     // Explanation of why comment was orphaned (empty if active)
	OrphanedAt     *time.Time // Timestamp when comment was marked as orphaned (nil if never orphaned)

	// Thread structure (nested replies)
	Replies []*Comment // Nested replies to this comment (empty for leaf comments)

	// Suggestion fields (for edit suggestions)
	IsSuggestion bool   // True if this is an edit suggestion
	StartLine    int    // Start line for suggestion (0 if not a suggestion)
	EndLine      int    // End line for suggestion (0 if not a suggestion)
	OriginalText string // Original text being replaced (empty if not a suggestion)
	ProposedText string // Proposed replacement text (empty if not a suggestion)
	Accepted     *bool  // nil=pending, true=accepted, false=rejected (nil if not a suggestion)
}

// IsPending returns true if this suggestion is awaiting review
func (c *Comment) IsPending() bool {
	return c.IsSuggestion && c.Accepted == nil
}

// IsAccepted returns true if this suggestion has been accepted
func (c *Comment) IsAccepted() bool {
	return c.IsSuggestion && c.Accepted != nil && *c.Accepted
}

// IsRejected returns true if this suggestion has been rejected
func (c *Comment) IsRejected() bool {
	return c.IsSuggestion && c.Accepted != nil && !*c.Accepted
}

// CountReplies returns the total number of replies (direct + nested)
func (c *Comment) CountReplies() int {
	count := len(c.Replies)
	for _, reply := range c.Replies {
		count += reply.CountReplies()
	}
	return count
}

// LatestTimestamp returns the timestamp of the most recent comment in thread
func (c *Comment) LatestTimestamp() time.Time {
	latest := c.Timestamp
	for _, reply := range c.Replies {
		replyLatest := reply.LatestTimestamp()
		if replyLatest.After(latest) {
			latest = replyLatest
		}
	}
	return latest
}

// Status management methods

// IsOrphaned returns true if the comment's position is no longer valid
func (c *Comment) IsOrphaned() bool {
	return c.Status == StatusOrphaned
}

// IsCompleted returns true if the comment/task has been marked as completed
func (c *Comment) IsCompleted() bool {
	return c.Status == StatusCompleted
}

// GetStatus returns the comment status with default "active" for backward compatibility
func (c *Comment) GetStatus() Status {
	if c.Status == "" {
		return StatusActive
	}
	return c.Status
}

// GetPriority returns the comment priority with default "medium"
func (c *Comment) GetPriority() Priority {
	if c.Priority == "" {
		return PriorityMedium
	}
	return c.Priority
}

// ReviewRecord captures a completed human review pass over a document
type ReviewRecord struct {
	Author       string    `json:"author"`
	Timestamp    time.Time `json:"timestamp"`
	Decision     string    `json:"decision"` // "approved" or "changes_requested"
	Note         string    `json:"note,omitempty"`
	DocumentHash string    `json:"document_hash,omitempty"`
	IntentHash   string    `json:"intent_hash,omitempty"`
}

// DocumentWithComments represents a parsed document with comment threads (v2.0)
type DocumentWithComments struct {
	Content       string         // Raw markdown content without comment markup
	Threads       []*Comment     // Root comment threads (each may contain nested replies)
	DocumentHash  string         // SHA-256 hash of content for staleness detection
	LastValidated time.Time      // Last time sidecar was validated against document
	Reviews       []ReviewRecord // Completed review passes (signoffs), newest last
	Template      string         // Legacy sidecar template association; frontmatter is preferred
}

// GetAllComments returns a flat list of all comments (roots + replies)
// Useful for filtering and searching operations
func (d *DocumentWithComments) GetAllComments() []*Comment {
	all := []*Comment{}
	for _, thread := range d.Threads {
		all = append(all, thread)
		all = append(all, flattenReplies(thread.Replies)...)
	}
	return all
}

// flattenReplies recursively flattens nested replies into a single array
func flattenReplies(replies []*Comment) []*Comment {
	flat := []*Comment{}
	for _, reply := range replies {
		flat = append(flat, reply)
		flat = append(flat, flattenReplies(reply.Replies)...)
	}
	return flat
}

// FindThreadByID finds a thread by its ID (root comment ID)
func (d *DocumentWithComments) FindThreadByID(id string) *Comment {
	for _, thread := range d.Threads {
		if thread.ID == id {
			return thread
		}
	}
	return nil
}

// FindCommentByID finds any comment by ID (root or reply). It shares the
// single recursive traversal in findCommentByID (helpers.go).
func (d *DocumentWithComments) FindCommentByID(id string) *Comment {
	return findCommentByID(d.Threads, id)
}

// Migration helpers for upgrading old sidecars to new format

// MigrateComment updates a comment to the new format with status/priority fields
// This is called when loading old sidecars that don't have the new fields
func MigrateComment(c *Comment) {
	// Set default status if empty
	if c.Status == "" {
		c.Status = StatusActive
	}

	// Set default priority if empty
	if c.Priority == "" {
		c.Priority = PriorityMedium
	}

	// Preserve original line position
	if c.OriginalLine == 0 && c.Line > 0 {
		c.OriginalLine = c.Line
	}

	// Recursively migrate all replies
	for _, reply := range c.Replies {
		MigrateComment(reply)
	}
}

// MigrateDocument migrates all comments in a document to the new format
func (d *DocumentWithComments) MigrateDocument() {
	for _, thread := range d.Threads {
		MigrateComment(thread)
	}
}
