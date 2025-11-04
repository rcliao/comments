package comment

import (
	"testing"
	"time"
)

func TestGetVisibleComments(t *testing.T) {
	threads := []*Comment{
		{ID: "c1", Resolved: false, Replies: []*Comment{}}, // Root, not resolved
		{ID: "c2", Resolved: true, Replies: []*Comment{}},  // Root, resolved
		{ID: "c3", Resolved: false, Replies: []*Comment{}}, // Root, not resolved
	}

	// Without showing resolved
	visible := GetVisibleComments(threads, false)
	if len(visible) != 2 {
		t.Errorf("Visible comments (hide resolved) = %d, want 2", len(visible))
	}

	// With showing resolved
	visibleAll := GetVisibleComments(threads, true)
	if len(visibleAll) != 3 {
		t.Errorf("Visible comments (show resolved) = %d, want 3", len(visibleAll))
	}
}

func TestAddReplyToThread(t *testing.T) {
	thread := &Comment{
		ID:      "c1",
		Text:    "Root",
		Line:    5,
		Replies: []*Comment{},
	}

	threads := []*Comment{thread}

	err := AddReplyToThread(threads, "c1", "user", "Reply text")
	if err != nil {
		t.Fatalf("AddReplyToThread failed: %v", err)
	}

	if len(thread.Replies) != 1 {
		t.Errorf("Thread replies = %d, want 1", len(thread.Replies))
	}

	reply := thread.Replies[0]
	if reply.Author != "user" {
		t.Errorf("Reply author = %s, want user", reply.Author)
	}

	if reply.Text != "Reply text" {
		t.Errorf("Reply text = %s, want 'Reply text'", reply.Text)
	}

	if reply.Line != thread.Line {
		t.Errorf("Reply line = %d, want %d (inherited from parent)", reply.Line, thread.Line)
	}

	// Test adding reply to non-existent thread
	err = AddReplyToThread(threads, "nonexistent", "user", "Reply")
	if err == nil {
		t.Error("Expected error when adding reply to non-existent thread")
	}
}

func TestResolveThread(t *testing.T) {
	threads := []*Comment{
		{ID: "c1", Resolved: false, Replies: []*Comment{}},
		{ID: "c2", Resolved: false, Replies: []*Comment{}},
	}

	err := ResolveThread(threads, "c1")
	if err != nil {
		t.Fatalf("ResolveThread failed: %v", err)
	}

	// Thread c1 should be resolved
	if !threads[0].Resolved {
		t.Error("Thread c1 should be resolved")
	}

	// Thread c2 should not be resolved
	if threads[1].Resolved {
		t.Error("Thread c2 should not be resolved")
	}

	// Test resolving non-existent thread
	err = ResolveThread(threads, "nonexistent")
	if err == nil {
		t.Error("Expected error when resolving non-existent thread")
	}
}

func TestUnresolveThread(t *testing.T) {
	threads := []*Comment{
		{ID: "c1", Resolved: true, Replies: []*Comment{}},
	}

	err := UnresolveThread(threads, "c1")
	if err != nil {
		t.Fatalf("UnresolveThread failed: %v", err)
	}

	if threads[0].Resolved {
		t.Error("Thread c1 should be unresolved")
	}
}

func TestNewComment(t *testing.T) {
	comment := NewComment("user", 5, "Test comment")

	if comment.Author != "user" {
		t.Errorf("Author = %s, want user", comment.Author)
	}

	if comment.Line != 5 {
		t.Errorf("Line = %d, want 5", comment.Line)
	}

	if comment.Text != "Test comment" {
		t.Errorf("Text = %s, want 'Test comment'", comment.Text)
	}

	if comment.ID == "" {
		t.Error("ID should not be empty")
	}

	if comment.Resolved {
		t.Error("New comment should not be resolved")
	}

	if comment.Replies == nil {
		t.Error("Replies array should be initialized")
	}

	if len(comment.Replies) != 0 {
		t.Error("Replies array should be empty")
	}
}

func TestNewCommentWithType(t *testing.T) {
	comment := NewCommentWithType("user", 10, "Question text", "Q")

	if comment.Type != "Q" {
		t.Errorf("Type = %s, want Q", comment.Type)
	}

	if comment.Text != "Question text" {
		t.Errorf("Text = %s, want 'Question text'", comment.Text)
	}

	if comment.Line != 10 {
		t.Errorf("Line = %d, want 10", comment.Line)
	}
}

func TestNewReply(t *testing.T) {
	parent := &Comment{
		ID:   "c1",
		Line: 5,
		SectionID: "s1",
		SectionPath: "Intro > Overview",
	}

	reply := NewReply("claude", "Test reply", parent)

	if reply.Author != "claude" {
		t.Errorf("Author = %s, want claude", reply.Author)
	}

	if reply.Text != "Test reply" {
		t.Errorf("Text = %s, want 'Test reply'", reply.Text)
	}

	// Reply should inherit line from parent
	if reply.Line != parent.Line {
		t.Errorf("Line = %d, want %d (inherited from parent)", reply.Line, parent.Line)
	}

	// Reply should inherit section metadata
	if reply.SectionID != parent.SectionID {
		t.Errorf("SectionID = %s, want %s", reply.SectionID, parent.SectionID)
	}

	if reply.SectionPath != parent.SectionPath {
		t.Errorf("SectionPath = %s, want %s", reply.SectionPath, parent.SectionPath)
	}

	if reply.ID == "" {
		t.Error("ID should not be empty")
	}

	if reply.Replies == nil {
		t.Error("Replies array should be initialized")
	}
}

func TestCountReplies(t *testing.T) {
	// Create nested reply structure
	thread := &Comment{
		ID: "c1",
		Replies: []*Comment{
			{
				ID: "c2",
				Replies: []*Comment{
					{ID: "c3", Replies: []*Comment{}},
				},
			},
			{ID: "c4", Replies: []*Comment{}},
		},
	}

	// Should count direct replies (2) + nested reply (1) = 3
	if thread.CountReplies() != 3 {
		t.Errorf("CountReplies = %d, want 3", thread.CountReplies())
	}
}

func TestLatestTimestamp(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Minute)
	latest := now.Add(10 * time.Minute)

	thread := &Comment{
		ID:        "c1",
		Timestamp: now,
		Replies: []*Comment{
			{
				ID:        "c2",
				Timestamp: later,
				Replies:   []*Comment{},
			},
			{
				ID:        "c3",
				Timestamp: latest,
				Replies:   []*Comment{},
			},
		},
	}

	latestTime := thread.LatestTimestamp()
	if !latestTime.Equal(latest) {
		t.Errorf("LatestTimestamp = %v, want %v", latestTime, latest)
	}
}

func TestGroupCommentsByLine(t *testing.T) {
	threads := []*Comment{
		{
			ID:   "c1",
			Line: 5,
			Replies: []*Comment{
				{ID: "c2", Line: 5},
			},
		},
		{
			ID:      "c3",
			Line:    5,
			Replies: []*Comment{},
		},
		{
			ID:      "c4",
			Line:    10,
			Replies: []*Comment{},
		},
	}

	byLine := GroupCommentsByLine(threads)

	if len(byLine) != 2 {
		t.Errorf("GroupCommentsByLine length = %d, want 2", len(byLine))
	}

	// Line 5 should have 3 comments (c1, c2, c3)
	if len(byLine[5]) != 3 {
		t.Errorf("Line 5 comments = %d, want 3", len(byLine[5]))
	}

	// Line 10 should have 1 comment
	if len(byLine[10]) != 1 {
		t.Errorf("Line 10 comments = %d, want 1", len(byLine[10]))
	}
}

func TestGetPendingSuggestions(t *testing.T) {
	accepted := true
	rejected := false

	threads := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil, // Pending
			Replies:      []*Comment{},
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     &accepted, // Accepted
			Replies:      []*Comment{},
		},
		{
			ID:           "s3",
			IsSuggestion: true,
			Accepted:     &rejected, // Rejected
			Replies:      []*Comment{},
		},
		{
			ID:           "c1",
			IsSuggestion: false, // Regular comment
			Replies:      []*Comment{},
		},
	}

	pending := GetPendingSuggestions(threads)

	if len(pending) != 1 {
		t.Errorf("GetPendingSuggestions = %d, want 1", len(pending))
	}

	if pending[0].ID != "s1" {
		t.Errorf("Pending suggestion ID = %s, want s1", pending[0].ID)
	}
}

func TestGetSuggestionsByAuthor(t *testing.T) {
	threads := []*Comment{
		{
			ID:           "s1",
			Author:       "claude",
			IsSuggestion: true,
			Accepted:     nil,
			Replies:      []*Comment{},
		},
		{
			ID:           "s2",
			Author:       "alice",
			IsSuggestion: true,
			Accepted:     nil,
			Replies:      []*Comment{},
		},
		{
			ID:           "s3",
			Author:       "claude",
			IsSuggestion: true,
			Accepted:     nil,
			Replies:      []*Comment{},
		},
	}

	claudeSuggestions := GetSuggestionsByAuthor(threads, "claude")

	if len(claudeSuggestions) != 2 {
		t.Errorf("GetSuggestionsByAuthor(claude) = %d, want 2", len(claudeSuggestions))
	}

	aliceSuggestions := GetSuggestionsByAuthor(threads, "alice")

	if len(aliceSuggestions) != 1 {
		t.Errorf("GetSuggestionsByAuthor(alice) = %d, want 1", len(aliceSuggestions))
	}
}

func TestAcceptSuggestion(t *testing.T) {
	threads := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			Replies:      []*Comment{},
		},
	}

	err := AcceptSuggestion(threads, "s1")
	if err != nil {
		t.Fatalf("AcceptSuggestion failed: %v", err)
	}

	if !threads[0].IsAccepted() {
		t.Error("Suggestion should be accepted")
	}

	// Test accepting non-existent suggestion
	err = AcceptSuggestion(threads, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent suggestion")
	}

	// Test accepting non-suggestion comment
	threads = append(threads, &Comment{
		ID:           "c1",
		IsSuggestion: false,
		Replies:      []*Comment{},
	})
	err = AcceptSuggestion(threads, "c1")
	if err == nil {
		t.Error("Expected error for accepting non-suggestion")
	}
}

func TestRejectSuggestion(t *testing.T) {
	threads := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			Replies:      []*Comment{},
		},
	}

	err := RejectSuggestion(threads, "s1")
	if err != nil {
		t.Fatalf("RejectSuggestion failed: %v", err)
	}

	if !threads[0].IsRejected() {
		t.Error("Suggestion should be rejected")
	}

	// Test rejecting non-existent suggestion
	err = RejectSuggestion(threads, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent suggestion")
	}
}

func TestIsPending(t *testing.T) {
	pending := &Comment{
		IsSuggestion: true,
		Accepted:     nil,
	}

	if !pending.IsPending() {
		t.Error("Should be pending")
	}

	accepted := true
	acceptedSuggestion := &Comment{
		IsSuggestion: true,
		Accepted:     &accepted,
	}

	if acceptedSuggestion.IsPending() {
		t.Error("Should not be pending (accepted)")
	}

	nonSuggestion := &Comment{
		IsSuggestion: false,
	}

	if nonSuggestion.IsPending() {
		t.Error("Non-suggestion should not be pending")
	}
}

func TestIsAccepted(t *testing.T) {
	accepted := true
	acceptedSuggestion := &Comment{
		IsSuggestion: true,
		Accepted:     &accepted,
	}

	if !acceptedSuggestion.IsAccepted() {
		t.Error("Should be accepted")
	}

	rejected := false
	rejectedSuggestion := &Comment{
		IsSuggestion: true,
		Accepted:     &rejected,
	}

	if rejectedSuggestion.IsAccepted() {
		t.Error("Should not be accepted (rejected)")
	}
}

func TestIsRejected(t *testing.T) {
	rejected := false
	rejectedSuggestion := &Comment{
		IsSuggestion: true,
		Accepted:     &rejected,
	}

	if !rejectedSuggestion.IsRejected() {
		t.Error("Should be rejected")
	}

	accepted := true
	acceptedSuggestion := &Comment{
		IsSuggestion: true,
		Accepted:     &accepted,
	}

	if acceptedSuggestion.IsRejected() {
		t.Error("Should not be rejected (accepted)")
	}
}
