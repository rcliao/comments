package comment

import (
	"os"
	"path/filepath"
	"testing"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatchSnapshotDiff(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "d.md")
	content := "# T\n\nBody line.\n"
	must(t, os.WriteFile(mdPath, []byte(content), 0644))

	doc := &DocumentWithComments{Content: content, Threads: []*Comment{
		{ID: "c1", Author: "rcliao", Line: 3, Text: "note", Blocking: true},
	}}
	must(t, SaveToSidecar(mdPath, doc))
	base := TakeSnapshot(mdPath)
	if !base.valid || base.gate != DecisionChangesRequested {
		t.Fatalf("bad base snapshot: %+v", base)
	}

	// reply + resolve + signoff
	must(t, AddReplyToThread(doc.Threads, "c1", "claude", "done"))
	doc.Threads[0].Resolved = true
	AddReviewRecord(doc, "rcliao", "", "", false)
	doc.Threads = append(doc.Threads, &Comment{ID: "c2", Author: "rcliao", Line: 3, Text: "new one"})
	must(t, SaveToSidecar(mdPath, doc))

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

// An agent waiting on `watch --until signoff` must get the reviewer's message
// in the event itself, not just the decision — the note is the whole point of
// signoff --note and the TUI verdict note.
func TestSignoffEventCarriesAuthorAndNote(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "d.md")
	content := "# T\n\nBody line.\n"
	must(t, os.WriteFile(mdPath, []byte(content), 0644))

	doc := &DocumentWithComments{Content: content, Threads: []*Comment{
		{ID: "c1", Author: "rcliao", Line: 3, Text: "note"},
	}}
	must(t, SaveToSidecar(mdPath, doc))
	base := TakeSnapshot(mdPath)

	AddReviewRecord(doc, "rcliao", DecisionChangesRequested, "pin the prompt, don't hash it", false)
	must(t, SaveToSidecar(mdPath, doc))

	var signoff *WatchEvent
	for _, e := range DiffSnapshots("d.md", base, TakeSnapshot(mdPath)) {
		if e.Event == "signoff" {
			signoff = &e
		}
	}
	if signoff == nil {
		t.Fatal("no signoff event emitted")
	}
	if signoff.Decision != DecisionChangesRequested {
		t.Errorf("decision = %q, want %q", signoff.Decision, DecisionChangesRequested)
	}
	if signoff.Author != "rcliao" {
		t.Errorf("author = %q, want rcliao", signoff.Author)
	}
	if signoff.Note != "pin the prompt, don't hash it" {
		t.Errorf("note = %q, want the reviewer's note", signoff.Note)
	}
}

func TestMatchesUntil(t *testing.T) {
	cases := []struct {
		event, spec string
		want        bool
	}{
		{"signoff", "signoff", true},
		{"signoff", "gate_changed", false},
		{"signoff", "gate_changed,signoff", true},
		{"gate_changed", "gate_changed,signoff", true},
		{"reply_added", "gate_changed,signoff", false},
		{"signoff", " signoff , gate_changed ", true}, // whitespace tolerated
		{"signoff", "", false},                        // empty spec never matches
		{"", "signoff", false},
		{"signoff", "sign", false}, // no prefix matching
	}
	for _, c := range cases {
		if got := MatchesUntil(c.event, c.spec); got != c.want {
			t.Errorf("MatchesUntil(%q, %q) = %v, want %v", c.event, c.spec, got, c.want)
		}
	}
}

func TestWatchInitialLoadEmitsNothing(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "d.md")
	must(t, os.WriteFile(mdPath, []byte("# T\n"), 0644))
	doc := &DocumentWithComments{Content: "# T\n", Threads: []*Comment{{ID: "c1", Author: "a", Line: 1, Text: "x"}}}
	must(t, SaveToSidecar(mdPath, doc))

	events := DiffSnapshots("d.md", WatchSnapshot{}, TakeSnapshot(mdPath))
	if len(events) != 0 {
		t.Errorf("initial load should emit no events, got %v", events)
	}
}
