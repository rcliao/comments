package comment

import (
	"fmt"
	"sort"
	"time"
)

// BuildThreads organizes comments into conversation threads
func BuildThreads(comments []*Comment) map[string]*Thread {
	threads := make(map[string]*Thread)

	// First pass: create threads for root comments
	for _, c := range comments {
		if c.IsRoot() {
			// For root comments, ThreadID should equal ID
			if c.ThreadID == "" {
				c.ThreadID = c.ID
			}

			threads[c.ID] = &Thread{
				ID:          c.ID,
				RootComment: c,
				Replies:     make([]*Comment, 0),
				Resolved:    c.Resolved,
				Line:        c.Line,
			}
		}
	}

	// Second pass: add replies to their threads
	for _, c := range comments {
		if c.IsReply() {
			threadID := c.ThreadID
			if threadID == "" {
				// Fallback: use ParentID to find thread
				threadID = c.ParentID
			}

			if thread, ok := threads[threadID]; ok {
				thread.Replies = append(thread.Replies, c)
			}
		}
	}

	// Sort replies by timestamp within each thread
	for _, thread := range threads {
		sort.Slice(thread.Replies, func(i, j int) bool {
			return thread.Replies[i].Timestamp.Before(thread.Replies[j].Timestamp)
		})
	}

	return threads
}

// GetVisibleComments returns only root comments, optionally filtering resolved ones
func GetVisibleComments(comments []*Comment, showResolved bool) []*Comment {
	visible := make([]*Comment, 0)

	for _, c := range comments {
		if c.IsRoot() {
			if showResolved || !c.Resolved {
				visible = append(visible, c)
			}
		}
	}

	return visible
}

// AddReply adds a reply to a thread
func AddReply(comments []*Comment, threadID string, reply *Comment) []*Comment {
	// Set thread metadata
	reply.ThreadID = threadID
	reply.ParentID = threadID // For now, flat threading (all replies to root)

	// Add to comments list
	return append(comments, reply)
}

// ResolveThread marks a thread as resolved
func ResolveThread(comments []*Comment, threadID string) {
	for _, c := range comments {
		if c.ID == threadID || c.ThreadID == threadID {
			c.Resolved = true
		}
	}
}

// UnresolveThread marks a thread as unresolved
func UnresolveThread(comments []*Comment, threadID string) {
	for _, c := range comments {
		if c.ID == threadID || c.ThreadID == threadID {
			c.Resolved = false
		}
	}
}

// GetThreadAtLine finds the thread at a specific line
func GetThreadAtLine(threads map[string]*Thread, line int) *Thread {
	for _, thread := range threads {
		if thread.Line == line {
			return thread
		}
	}
	return nil
}

// NewComment creates a new root comment
func NewComment(author string, line int, text string) *Comment {
	id := generateCommentID()
	return &Comment{
		ID:        id,
		ThreadID:  id, // Root comment's thread ID is its own ID
		ParentID:  "",
		Author:    author,
		Line:      line,
		Timestamp: time.Now(),
		Text:      text,
		Resolved:  false,
	}
}

// NewReply creates a new reply to a comment
func NewReply(author string, threadID string, text string) *Comment {
	return &Comment{
		ID:        generateCommentID(),
		ThreadID:  threadID,
		ParentID:  threadID,
		Author:    author,
		Line:      0, // Replies don't have their own line
		Timestamp: time.Now(),
		Text:      text,
		Resolved:  false,
	}
}

// generateCommentID generates a unique comment ID
func generateCommentID() string {
	return fmt.Sprintf("c%d", time.Now().UnixNano())
}

// GetThreadByID retrieves a thread by its ID
func GetThreadByID(threads map[string]*Thread, id string) (*Thread, bool) {
	thread, ok := threads[id]
	return thread, ok
}

// GroupCommentsByLine groups comments by their line number
func GroupCommentsByLine(comments []*Comment) map[int][]*Comment {
	byLine := make(map[int][]*Comment)

	for _, c := range comments {
		if c.IsRoot() {
			byLine[c.Line] = append(byLine[c.Line], c)
		}
	}

	return byLine
}
