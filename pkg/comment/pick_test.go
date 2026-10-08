package comment

import (
	"strings"
	"testing"
)

// A pick is a decision the agent made alone and proceeds on: it never blocks
// the default gate, it survives a save, and the strict end-of-work gate fails
// until the human settles it.
func TestPickProceedsButFailsStrictGate(t *testing.T) {
	path := writeSurfaceFixture(t, surfaceFixture)
	mutateSurfaceDoc(t, path, func(doc *DocumentWithComments) {
		res, err := AddComments(doc, []NewCommentSpec{{Anchor: "It is slow.", Author: "claude", Text: "Cache or index?", Type: "Q", Pick: "index"}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Added[0].Pick != "index" {
			t.Fatalf("view pick = %q", res.Added[0].Pick)
		}
	})
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Threads[0].Pick; got != "index" {
		t.Fatalf("stored pick = %q", got)
	}
	if d := EvaluateGate(doc, false).Decision; d != DecisionApproved {
		t.Fatalf("default gate = %s, want approved", d)
	}
	if d := EvaluateGate(doc, true).Decision; d != DecisionChangesRequested {
		t.Fatalf("strict gate = %s, want changes_requested", d)
	}
}

func TestPickCannotBeBlocking(t *testing.T) {
	doc := &DocumentWithComments{Content: surfaceFixture}
	_, err := AddComments(doc, []NewCommentSpec{{Anchor: "It is slow.", Author: "claude", Text: "x", Pick: "a", Blocking: true}})
	if err == nil || !strings.Contains(err.Error(), "pick") {
		t.Fatalf("err = %v, want a refusal naming pick", err)
	}
}
