package comment

import "testing"

func TestDocumentAnchorHealthCountsDegradedUnresolvedThreads(t *testing.T) {
	doc := &DocumentWithComments{Threads: []*Comment{
		{ID: "a", AnchorConfidence: ConfidenceExact},
		{ID: "b", AnchorConfidence: ConfidenceFuzzy},
		{ID: "c", AnchorConfidence: ConfidenceFuzzy},
		{ID: "d", AnchorConfidence: ConfidenceSectionLevel},
		{ID: "e", AnchorConfidence: ""},
	}}
	h := DocumentAnchorHealth(doc)
	if h.Fuzzy != 2 {
		t.Errorf("Fuzzy = %d, want 2", h.Fuzzy)
	}
	if h.SectionLevel != 1 {
		t.Errorf("SectionLevel = %d, want 1", h.SectionLevel)
	}
	if h.Total() != 3 {
		t.Errorf("Total() = %d, want 3", h.Total())
	}
}

// A resolved thread's anchor no longer decides anything, so re-checking it is
// work nobody needs done — it must not inflate the rail's count.
func TestDocumentAnchorHealthSkipsResolvedThreads(t *testing.T) {
	doc := &DocumentWithComments{Threads: []*Comment{
		{ID: "a", AnchorConfidence: ConfidenceFuzzy, Resolved: true},
		{ID: "b", AnchorConfidence: ConfidenceSectionLevel, Resolved: true},
		{ID: "c", AnchorConfidence: ConfidenceFuzzy},
	}}
	if h := DocumentAnchorHealth(doc); h.Total() != 1 || h.Fuzzy != 1 {
		t.Errorf("resolved threads counted: %+v, want Fuzzy 1 / Total 1", h)
	}
}

func TestDocumentAnchorHealthNilSafe(t *testing.T) {
	if h := DocumentAnchorHealth(nil); h.Total() != 0 {
		t.Errorf("nil document should report no degraded anchors, got %+v", h)
	}
	doc := &DocumentWithComments{Threads: []*Comment{nil, {ID: "a", AnchorConfidence: ConfidenceFuzzy}}}
	if h := DocumentAnchorHealth(doc); h.Total() != 1 {
		t.Errorf("nil thread should be skipped, got %+v", h)
	}
}
