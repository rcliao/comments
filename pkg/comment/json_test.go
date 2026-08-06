package comment

import (
	"encoding/json"
	"sort"
	"testing"
	"time"
)

// fullComment returns a comment with every serializable field populated so
// omitempty can't hide a field from the name-set assertion.
func fullComment() *Comment {
	accepted := false
	return &Comment{
		ID:               "c1",
		Author:           "alice",
		Timestamp:        time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC),
		Text:             "root",
		Type:             "Q",
		Line:             5,
		SectionID:        "s1",
		SectionPath:      "Intro > Overview",
		Resolved:         true,
		Blocking:         true,
		OrphanedReason:   "line_deleted",
		AnchorConfidence: "fuzzy",
		IsSuggestion:     true,
		StartLine:        5,
		EndLine:          7,
		OriginalText:     "old",
		ProposedText:     "new",
		Accepted:         &accepted,
		Replies: []*Comment{
			{ID: "c2", Author: "bob", Timestamp: time.Date(2026, 8, 6, 11, 0, 0, 0, time.UTC),
				Text: "reply", Line: 5,
				Replies: []*Comment{
					{ID: "c3", Author: "alice", Timestamp: time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
						Text: "nested", Line: 5},
				}},
		},
	}
}

// TestCommentViewFieldNames pins the canonical snake_case field-name set so
// any drift in the shared serializer is caught here, not by a consumer.
func TestCommentViewFieldNames(t *testing.T) {
	data, err := json.Marshal(NewCommentView(fullComment()))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(m))
	for k := range m {
		got = append(got, k)
	}
	sort.Strings(got)
	want := []string{
		"accepted", "anchor_confidence", "author", "blocking", "end_line",
		"id", "is_suggestion", "line", "original_text", "orphaned_reason",
		"priority", "proposed_text", "replies", "reply_count", "resolved",
		"section_path", "start_line", "status", "text", "timestamp", "type",
	}
	if len(got) != len(want) {
		t.Fatalf("field-name set drifted:\n got %v\nwant %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("field-name set drifted at %q:\n got %v\nwant %v", got[i], got, want)
		}
	}
}

func TestCommentViewRoundTrip(t *testing.T) {
	data, err := json.Marshal(NewCommentView(fullComment()))
	if err != nil {
		t.Fatal(err)
	}
	var v CommentView
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	if v.ID != "c1" || !v.Blocking || v.AnchorConfidence != "fuzzy" {
		t.Errorf("root fields lost in round-trip: %+v", v)
	}
	if v.Timestamp != "2026-08-06T10:00:00Z" {
		t.Errorf("timestamp must be RFC3339, got %q", v.Timestamp)
	}
	if !v.IsSuggestion || v.StartLine != 5 || v.EndLine != 7 || v.OriginalText != "old" || v.ProposedText != "new" {
		t.Errorf("suggestion fields lost: %+v", v)
	}
	if v.Accepted == nil || *v.Accepted != false {
		t.Errorf("Accepted=false must survive (pointer, not omitted): %+v", v.Accepted)
	}
	if v.ReplyCount != 2 {
		t.Errorf("reply_count should count nested replies, got %d", v.ReplyCount)
	}
	if len(v.Replies) != 1 || len(v.Replies[0].Replies) != 1 || v.Replies[0].Replies[0].ID != "c3" {
		t.Errorf("nested replies lost in round-trip: %+v", v.Replies)
	}
}

func TestDocumentViewShape(t *testing.T) {
	doc := &DocumentWithComments{
		Content:       "# T\n",
		DocumentHash:  "abc",
		LastValidated: time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC),
		Template:      "design-doc",
		Reviews:       []ReviewRecord{{Author: "eric", Decision: DecisionApproved, Timestamp: time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)}},
		Threads:       []*Comment{fullComment()},
	}
	data, err := json.Marshal(NewDocumentView(doc))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"content", "document_hash", "last_validated", "template", "reviews", "threads"} {
		if _, ok := m[k]; !ok {
			t.Errorf("document view missing %q: have %v", k, m)
		}
	}
	// Threads must be the canonical snake_case shape, not PascalCase structs
	threads := m["threads"].([]any)
	first := threads[0].(map[string]any)
	if _, ok := first["id"]; !ok {
		t.Errorf("document view threads must use canonical snake_case comment view, got keys of first thread: %v", first)
	}
	// Empty threads must marshal as [] not null
	empty, _ := json.Marshal(NewDocumentView(&DocumentWithComments{Content: "x"}))
	var em map[string]any
	_ = json.Unmarshal(empty, &em)
	if _, ok := em["threads"].([]any); !ok {
		t.Errorf("empty threads must be [], got %v", em["threads"])
	}
}
