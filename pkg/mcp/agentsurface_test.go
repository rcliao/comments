package mcp

// Phase 2 of docs/plan-agent-surface.md: list returns thread roots with
// replies nested (no flattened reply items), and the write path accepts any
// ID it ever handed out — the external agent's failing sequence
// (list -> reply to a reply ID -> "thread not found") must succeed.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

func agentSurfaceDoc(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	content := "# T\n\nAlpha line.\nBeta line.\n"
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := &comment.DocumentWithComments{Content: content, Threads: []*comment.Comment{
		{ID: "root1", Author: "claude", Line: 3, Text: "root question", Replies: []*comment.Comment{
			{ID: "reply1", Author: "rcliao", Line: 3, Text: "human answer", Replies: []*comment.Comment{
				{ID: "reply2", Author: "claude", Line: 3, Text: "nested follow-up"},
			}},
		}},
		{ID: "root2", Author: "rcliao", Line: 4, Text: "second thread"},
	}}
	if err := comment.SaveToSidecar(mdPath, doc); err != nil {
		t.Fatal(err)
	}
	return mdPath
}

func TestListIsRootsNestedWithParentThreadID(t *testing.T) {
	mdPath := agentSurfaceDoc(t)
	doc, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	views := comment.NewCommentViews(doc.Threads)
	if len(views) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(views))
	}
	root := views[0]
	if root.ParentThreadID != "" {
		t.Errorf("roots carry no parent_thread_id, got %q", root.ParentThreadID)
	}
	if len(root.Replies) != 1 || root.Replies[0].Text != "human answer" {
		t.Fatalf("reply text must be nested in the view: %+v", root.Replies)
	}
	if got := root.Replies[0].ParentThreadID; got != "root1" {
		t.Errorf("reply parent_thread_id = %q, want root1", got)
	}
	// Depth 2: still the ROOT id, not the immediate parent — it is the ID the
	// write path addresses
	if got := root.Replies[0].Replies[0].ParentThreadID; got != "root1" {
		t.Errorf("nested reply parent_thread_id = %q, want root1", got)
	}
}

func TestWritePathAcceptsReplyIDs(t *testing.T) {
	mdPath := agentSurfaceDoc(t)
	doc, _, err := comment.LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}

	// The external agent's dead end: reply addressed by a reply's own ID
	if err := comment.AddReplyToThread(doc.Threads, "reply1", "claude", "lands on the thread"); err != nil {
		t.Fatalf("reply by reply ID must resolve to its thread: %v", err)
	}
	if n := len(doc.Threads[0].Replies); n != 2 {
		t.Fatalf("reply should append to root1's thread, got %d replies", n)
	}

	// Resolve by nested reply ID
	if err := comment.ResolveThread(doc.Threads, "reply2"); err != nil {
		t.Fatalf("resolve by nested reply ID: %v", err)
	}
	if !doc.Threads[0].Resolved {
		t.Error("resolving by reply ID should resolve the root thread")
	}
	if err := comment.UnresolveThread(doc.Threads, "reply2"); err != nil || doc.Threads[0].Resolved {
		t.Errorf("unresolve by reply ID: %v", err)
	}

	// Unknown IDs still fail loudly
	if err := comment.AddReplyToThread(doc.Threads, "nope", "x", "y"); err == nil {
		t.Error("unknown ID must still error")
	}
}
