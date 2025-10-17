package comment

import (
	"testing"
	"time"
)

func TestBuildThreads(t *testing.T) {
	now := time.Now()

	// Create test comments with threading
	rootComment1 := &Comment{
		ID:        "c1",
		ThreadID:  "c1",
		ParentID:  "",
		Author:    "user",
		Line:      5,
		Timestamp: now,
		Text:      "Root comment 1",
		Resolved:  false,
	}

	reply1 := &Comment{
		ID:        "c2",
		ThreadID:  "c1",
		ParentID:  "c1",
		Author:    "claude",
		Line:      0,
		Timestamp: now.Add(time.Hour),
		Text:      "Reply to c1",
		Resolved:  false,
	}

	reply2 := &Comment{
		ID:        "c3",
		ThreadID:  "c1",
		ParentID:  "c1",
		Author:    "user",
		Line:      0,
		Timestamp: now.Add(2 * time.Hour),
		Text:      "Another reply to c1",
		Resolved:  false,
	}

	rootComment2 := &Comment{
		ID:        "c4",
		ThreadID:  "c4",
		ParentID:  "",
		Author:    "user",
		Line:      10,
		Timestamp: now.Add(3 * time.Hour),
		Text:      "Root comment 2",
		Resolved:  false,
	}

	comments := []*Comment{rootComment1, reply1, reply2, rootComment2}

	threads := BuildThreads(comments)

	// Should have 2 threads
	if len(threads) != 2 {
		t.Errorf("Expected 2 threads, got %d", len(threads))
	}

	// Check thread 1
	thread1, ok := threads["c1"]
	if !ok {
		t.Fatal("Thread c1 not found")
	}

	if thread1.RootComment.ID != "c1" {
		t.Errorf("Thread 1 root comment ID = %s, want c1", thread1.RootComment.ID)
	}

	if len(thread1.Replies) != 2 {
		t.Errorf("Thread 1 replies = %d, want 2", len(thread1.Replies))
	}

	// Replies should be sorted by timestamp
	if thread1.Replies[0].ID != "c2" {
		t.Errorf("First reply ID = %s, want c2", thread1.Replies[0].ID)
	}

	if thread1.Replies[1].ID != "c3" {
		t.Errorf("Second reply ID = %s, want c3", thread1.Replies[1].ID)
	}

	// Check thread 2
	thread2, ok := threads["c4"]
	if !ok {
		t.Fatal("Thread c4 not found")
	}

	if thread2.RootComment.ID != "c4" {
		t.Errorf("Thread 2 root comment ID = %s, want c4", thread2.RootComment.ID)
	}

	if len(thread2.Replies) != 0 {
		t.Errorf("Thread 2 replies = %d, want 0", len(thread2.Replies))
	}
}

func TestGetVisibleComments(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", ThreadID: "c1", ParentID: "", Resolved: false}, // Root, not resolved
		{ID: "c2", ThreadID: "c1", ParentID: "c1", Resolved: false}, // Reply
		{ID: "c3", ThreadID: "c3", ParentID: "", Resolved: true},  // Root, resolved
		{ID: "c4", ThreadID: "c4", ParentID: "", Resolved: false}, // Root, not resolved
	}

	// Without showing resolved
	visible := GetVisibleComments(comments, false)
	if len(visible) != 2 {
		t.Errorf("Visible comments (hide resolved) = %d, want 2", len(visible))
	}

	// With showing resolved
	visibleAll := GetVisibleComments(comments, true)
	if len(visibleAll) != 3 {
		t.Errorf("Visible comments (show resolved) = %d, want 3", len(visibleAll))
	}
}

func TestAddReply(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", ThreadID: "c1", ParentID: "", Text: "Root"},
	}

	reply := &Comment{
		ID:     "c2",
		Author: "user",
		Text:   "Reply",
	}

	updated := AddReply(comments, "c1", reply)

	if len(updated) != 2 {
		t.Errorf("Comments length = %d, want 2", len(updated))
	}

	if reply.ThreadID != "c1" {
		t.Errorf("Reply ThreadID = %s, want c1", reply.ThreadID)
	}

	if reply.ParentID != "c1" {
		t.Errorf("Reply ParentID = %s, want c1", reply.ParentID)
	}
}

func TestResolveThread(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", ThreadID: "c1", ParentID: "", Resolved: false},
		{ID: "c2", ThreadID: "c1", ParentID: "c1", Resolved: false},
		{ID: "c3", ThreadID: "c3", ParentID: "", Resolved: false},
	}

	ResolveThread(comments, "c1")

	// Thread c1 should be resolved
	if !comments[0].Resolved {
		t.Error("Root comment c1 should be resolved")
	}

	if !comments[1].Resolved {
		t.Error("Reply c2 should be resolved")
	}

	// Thread c3 should not be resolved
	if comments[2].Resolved {
		t.Error("Comment c3 should not be resolved")
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

	if comment.ThreadID != comment.ID {
		t.Errorf("ThreadID = %s, want %s (should equal ID for root)", comment.ThreadID, comment.ID)
	}

	if comment.ParentID != "" {
		t.Error("ParentID should be empty for root comment")
	}

	if !comment.IsRoot() {
		t.Error("Comment should be root")
	}
}

func TestNewReply(t *testing.T) {
	reply := NewReply("claude", "c1", "Test reply")

	if reply.Author != "claude" {
		t.Errorf("Author = %s, want claude", reply.Author)
	}

	if reply.ThreadID != "c1" {
		t.Errorf("ThreadID = %s, want c1", reply.ThreadID)
	}

	if reply.ParentID != "c1" {
		t.Errorf("ParentID = %s, want c1", reply.ParentID)
	}

	if reply.Text != "Test reply" {
		t.Errorf("Text = %s, want 'Test reply'", reply.Text)
	}

	if reply.IsRoot() {
		t.Error("Reply should not be root")
	}

	if !reply.IsReply() {
		t.Error("Comment should be a reply")
	}
}

func TestThreadCount(t *testing.T) {
	thread := &Thread{
		ID:          "c1",
		RootComment: &Comment{ID: "c1"},
		Replies:     []*Comment{{ID: "c2"}, {ID: "c3"}},
	}

	if thread.Count() != 3 {
		t.Errorf("Count = %d, want 3", thread.Count())
	}
}

func TestGroupCommentsByLine(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", ThreadID: "c1", ParentID: "", Line: 5},
		{ID: "c2", ThreadID: "c2", ParentID: "", Line: 5},
		{ID: "c3", ThreadID: "c1", ParentID: "c1", Line: 0}, // Reply
		{ID: "c4", ThreadID: "c4", ParentID: "", Line: 10},
	}

	byLine := GroupCommentsByLine(comments)

	if len(byLine) != 2 {
		t.Errorf("GroupCommentsByLine length = %d, want 2", len(byLine))
	}

	if len(byLine[5]) != 2 {
		t.Errorf("Line 5 comments = %d, want 2", len(byLine[5]))
	}

	if len(byLine[10]) != 1 {
		t.Errorf("Line 10 comments = %d, want 1", len(byLine[10]))
	}
}
