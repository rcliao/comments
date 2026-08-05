package comment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWatchSnapshotDiff(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "d.md")
	content := "# T\n\nBody line.\n"
	os.WriteFile(mdPath, []byte(content), 0644)

	doc := &DocumentWithComments{Content: content, Threads: []*Comment{
		{ID: "c1", Author: "rcliao", Line: 3, Text: "note", Blocking: true},
	}}
	SaveToSidecar(mdPath, doc)
	base := TakeSnapshot(mdPath)
	if !base.valid || base.gate != DecisionChangesRequested {
		t.Fatalf("bad base snapshot: %+v", base)
	}

	// reply + resolve + signoff
	AddReplyToThread(doc.Threads, "c1", "claude", "done")
	doc.Threads[0].Resolved = true
	AddReviewRecord(doc, "rcliao", "", "", false)
	doc.Threads = append(doc.Threads, &Comment{ID: "c2", Author: "rcliao", Line: 3, Text: "new one"})
	SaveToSidecar(mdPath, doc)

	events := DiffSnapshots("d.md", base, TakeSnapshot(mdPath))
	got := map[string]bool{}
	for _, e := range events {
		got[e.Event] = true
	}
	for _, want := range []string{"reply_added", "thread_resolved", "signoff", "gate_changed", "comment_added"} {
		if !got[want] {
			t.Errorf("missing event %s in %v", want, events)
		}
	}
}

func TestWatchInitialLoadEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "d.md")
	os.WriteFile(mdPath, []byte("# T\n"), 0644)
	doc := &DocumentWithComments{Content: "# T\n", Threads: []*Comment{{ID: "c1", Author: "a", Line: 1, Text: "x"}}}
	SaveToSidecar(mdPath, doc)

	events := DiffSnapshots("d.md", WatchSnapshot{}, TakeSnapshot(mdPath))
	if len(events) != 0 {
		t.Errorf("initial load should emit no events, got %v", events)
	}
}
