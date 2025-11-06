package comment

import "time"

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

	// State
	Resolved bool // Whether the comment/thread has been resolved

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

// IsRoot returns true if this is a root comment (has no parent)
// In v2.0, all top-level comments in the threads array are roots
func (c *Comment) IsRoot() bool {
	return true // In v2.0, we only store root comments in threads array
}

// IsReply returns true if this is a reply to another comment
// In v2.0, replies are found in the Replies array
func (c *Comment) IsReply() bool {
	return false // This method is context-dependent; caller knows based on array location
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

// IsActive returns true if the comment is active (attached to valid position)
func (c *Comment) IsActive() bool {
	status := c.GetStatus()
	return status == "active"
}

// IsOrphaned returns true if the comment's position is no longer valid
func (c *Comment) IsOrphaned() bool {
	return c.Status == "orphaned"
}

// IsCompleted returns true if the comment/task has been marked as completed
func (c *Comment) IsCompleted() bool {
	return c.Status == "completed"
}

// GetStatus returns the comment status with default "active" for backward compatibility
func (c *Comment) GetStatus() string {
	if c.Status == "" {
		return "active"
	}
	return c.Status
}

// GetPriority returns the comment priority with default "medium"
func (c *Comment) GetPriority() string {
	if c.Priority == "" {
		return "medium"
	}
	return c.Priority
}

// Position represents the location of a comment in a document (v2.0 simplified)
type Position struct {
	Line int // Current line number (may change as doc is edited)
}

// DocumentWithComments represents a parsed document with comment threads (v2.0)
type DocumentWithComments struct {
	Content      string     // Raw markdown content without comment markup
	Threads      []*Comment // Root comment threads (each may contain nested replies)
	DocumentHash string     // SHA-256 hash of content for staleness detection
	LastValidated time.Time  // Last time sidecar was validated against document
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

// FindCommentByID finds any comment by ID (root or reply)
func (d *DocumentWithComments) FindCommentByID(id string) *Comment {
	for _, thread := range d.Threads {
		if thread.ID == id {
			return thread
		}
		if found := findInReplies(thread.Replies, id); found != nil {
			return found
		}
	}
	return nil
}

// findInReplies recursively searches for a comment in replies
func findInReplies(replies []*Comment, id string) *Comment {
	for _, reply := range replies {
		if reply.ID == id {
			return reply
		}
		if found := findInReplies(reply.Replies, id); found != nil {
			return found
		}
	}
	return nil
}

// GetActiveComments returns all comments with status "active"
func (d *DocumentWithComments) GetActiveComments() []*Comment {
	return d.GetCommentsByStatus("active")
}

// GetOrphanedComments returns all comments with status "orphaned"
func (d *DocumentWithComments) GetOrphanedComments() []*Comment {
	return d.GetCommentsByStatus("orphaned")
}

// GetCompletedComments returns all comments with status "completed"
func (d *DocumentWithComments) GetCompletedComments() []*Comment {
	return d.GetCommentsByStatus("completed")
}

// GetCommentsByStatus returns all comments (roots + replies) with the given status
func (d *DocumentWithComments) GetCommentsByStatus(status string) []*Comment {
	filtered := []*Comment{}
	for _, comment := range d.GetAllComments() {
		if comment.GetStatus() == status {
			filtered = append(filtered, comment)
		}
	}
	return filtered
}

// GetCommentsByPriority returns all comments with the given priority
func (d *DocumentWithComments) GetCommentsByPriority(priority string) []*Comment {
	filtered := []*Comment{}
	for _, comment := range d.GetAllComments() {
		if comment.GetPriority() == priority {
			filtered = append(filtered, comment)
		}
	}
	return filtered
}

// Migration helpers for upgrading old sidecars to new format

// MigrateComment updates a comment to the new format with status/priority fields
// This is called when loading old sidecars that don't have the new fields
func MigrateComment(c *Comment) {
	// Set default status if empty
	if c.Status == "" {
		c.Status = "active"
	}

	// Set default priority if empty
	if c.Priority == "" {
		c.Priority = "medium"
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
