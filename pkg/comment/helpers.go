package comment

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

// generateID generates a short random comment ID (v2.1: 4-char base36, e.g. "c7f3k").
// Collisions within a document are resolved at save time by EnsureUniqueIDs.
func generateID() string {
	return "c" + randomBase36(4)
}

func randomBase36(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// Fall back to timestamp-derived digit; practically unreachable
			idx = big.NewInt(time.Now().UnixNano() % int64(len(idAlphabet)))
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return string(b)
}

// EnsureUniqueIDs regenerates any duplicate comment IDs within a document,
// growing the ID length on repeated collision. Existing (long) IDs are untouched
// unless duplicated. Called from SaveToSidecar so every write path is covered.
func EnsureUniqueIDs(doc *DocumentWithComments) {
	seen := map[string]bool{}
	for _, c := range doc.GetAllComments() {
		if !seen[c.ID] {
			seen[c.ID] = true
			continue
		}
		length := 4
		for attempt := 0; ; attempt++ {
			if attempt > 0 && attempt%3 == 0 {
				length++
			}
			candidate := "c" + randomBase36(length)
			if !seen[candidate] {
				c.ID = candidate
				seen[candidate] = true
				break
			}
		}
	}
}

// NewComment creates a new root comment (v2.0)
func NewComment(author string, line int, text string) *Comment {
	return &Comment{
		ID:        generateID(),
		Author:    author,
		Timestamp: time.Now(),
		Text:      text,
		Line:      line,
		Resolved:  false,
		Replies:   []*Comment{},
	}
}

// NewCommentWithType creates a new root comment with a type prefix (v2.0)
func NewCommentWithType(author string, line int, text string, commentType string) *Comment {
	comment := NewComment(author, line, text)
	comment.Type = commentType
	return comment
}

// NewReply creates a reply to an existing comment
func NewReply(author string, text string, parentComment *Comment) *Comment {
	return &Comment{
		ID:          generateID(),
		Author:      author,
		Timestamp:   time.Now(),
		Text:        text,
		Line:        parentComment.Line, // Inherit line from parent
		SectionID:   parentComment.SectionID,
		SectionPath: parentComment.SectionPath,
		Resolved:    false,
		Replies:     []*Comment{},
	}
}

// NewSuggestion creates a new suggestion comment (v2.0)
func NewSuggestion(author string, startLine, endLine int, text, originalText, proposedText string) *Comment {
	return &Comment{
		ID:           generateID(),
		Author:       author,
		Timestamp:    time.Now(),
		Text:         text,
		Line:         startLine,
		Resolved:     false,
		Replies:      []*Comment{},
		IsSuggestion: true,
		StartLine:    startLine,
		EndLine:      endLine,
		OriginalText: originalText,
		ProposedText: proposedText,
		Accepted:     nil, // Pending
	}
}

// GetVisibleComments returns comments that should be displayed based on resolved filter
// In v2.0, this operates on thread roots (since threads are already nested)
func GetVisibleComments(threads []*Comment, showResolved bool) []*Comment {
	if showResolved {
		// Show all threads
		return threads
	}

	// Filter to only unresolved threads
	visible := []*Comment{}
	for _, thread := range threads {
		if !thread.Resolved {
			visible = append(visible, thread)
		}
	}

	return visible
}

// GroupCommentsByLine groups comments by their line number for display
// In v2.0, this flattens all comments (roots + replies) and groups by line
func GroupCommentsByLine(threads []*Comment) map[int][]*Comment {
	grouped := make(map[int][]*Comment)

	// Get all comments (roots and replies)
	allComments := []*Comment{}
	for _, thread := range threads {
		allComments = append(allComments, thread)
		allComments = append(allComments, flattenReplies(thread.Replies)...)
	}

	// Group by line
	for _, comment := range allComments {
		grouped[comment.Line] = append(grouped[comment.Line], comment)
	}

	return grouped
}

// AddReplyToThread adds a reply to a thread
// Returns error if thread not found
func AddReplyToThread(threads []*Comment, threadID, author, text string) error {
	thread := findThreadByID(threads, threadID)
	if thread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	reply := NewReply(author, text, thread)
	thread.Replies = append(thread.Replies, reply)

	return nil
}

// findThreadByID finds a thread by ID (helper function)
func findThreadByID(threads []*Comment, id string) *Comment {
	for _, thread := range threads {
		if thread.ID == id {
			return thread
		}
	}
	return nil
}

// ResolveThread marks a thread as resolved
func ResolveThread(threads []*Comment, threadID string) error {
	thread := findThreadByID(threads, threadID)
	if thread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	thread.Resolved = true
	return nil
}

// UnresolveThread marks a thread as unresolved
func UnresolveThread(threads []*Comment, threadID string) error {
	thread := findThreadByID(threads, threadID)
	if thread == nil {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	thread.Resolved = false
	return nil
}

// GetPendingSuggestions returns all pending suggestions from threads
func GetPendingSuggestions(threads []*Comment) []*Comment {
	suggestions := []*Comment{}

	allComments := []*Comment{}
	for _, thread := range threads {
		allComments = append(allComments, thread)
		allComments = append(allComments, flattenReplies(thread.Replies)...)
	}

	for _, comment := range allComments {
		if comment.IsSuggestion && comment.IsPending() {
			suggestions = append(suggestions, comment)
		}
	}

	return suggestions
}

// GetSuggestionsByAuthor returns all pending suggestions by a specific author
func GetSuggestionsByAuthor(threads []*Comment, author string) []*Comment {
	suggestions := GetPendingSuggestions(threads)

	filtered := []*Comment{}
	for _, s := range suggestions {
		if s.Author == author {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// AcceptSuggestion marks a suggestion as accepted
func AcceptSuggestion(threads []*Comment, suggestionID string) error {
	suggestion := findCommentByID(threads, suggestionID)
	if suggestion == nil {
		return fmt.Errorf("suggestion not found: %s", suggestionID)
	}

	if !suggestion.IsSuggestion {
		return fmt.Errorf("comment is not a suggestion: %s", suggestionID)
	}

	accepted := true
	suggestion.Accepted = &accepted
	return nil
}

// RejectSuggestion marks a suggestion as rejected
func RejectSuggestion(threads []*Comment, suggestionID string) error {
	suggestion := findCommentByID(threads, suggestionID)
	if suggestion == nil {
		return fmt.Errorf("suggestion not found: %s", suggestionID)
	}

	if !suggestion.IsSuggestion {
		return fmt.Errorf("comment is not a suggestion: %s", suggestionID)
	}

	rejected := false
	suggestion.Accepted = &rejected
	return nil
}

// findCommentByID recursively finds a comment by ID in threads
func findCommentByID(threads []*Comment, id string) *Comment {
	for _, thread := range threads {
		if thread.ID == id {
			return thread
		}
		if found := findInRepliesHelper(thread.Replies, id); found != nil {
			return found
		}
	}
	return nil
}

// findInRepliesHelper recursively searches replies
func findInRepliesHelper(replies []*Comment, id string) *Comment {
	for _, reply := range replies {
		if reply.ID == id {
			return reply
		}
		if found := findInRepliesHelper(reply.Replies, id); found != nil {
			return found
		}
	}
	return nil
}
