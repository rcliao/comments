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

// pickDoc holds three picks and a plain thread: one pick nobody answered, one
// its author followed up on, and one the human objected to under a reply.
func pickDoc() *DocumentWithComments {
	followUp := &Comment{ID: "r1", Author: "claude", Text: "Measured: index wins."}
	objection := &Comment{ID: "r2", Author: "claude", Text: "Why not cache?",
		Replies: []*Comment{{ID: "r3", Author: "rcliao", Text: "Cache, please."}}}
	return &DocumentWithComments{Content: surfaceFixture, Threads: []*Comment{
		{ID: "p1", Author: "claude", Text: "Cache or index?", Pick: "index"},
		{ID: "p2", Author: "claude", Text: "Retry or fail?", Pick: "retry", Replies: []*Comment{followUp}},
		{ID: "p3", Author: "claude", Text: "Sync or async?", Pick: "sync", Replies: []*Comment{objection}},
		{ID: "q1", Author: "claude", Text: "Plain question."},
	}}
}

// Approving settles each pick nobody else answered; a pick with any reply from
// someone other than its author stays open, as do threads that are not picks.
func TestSettlePicksOnApprove(t *testing.T) {
	doc := pickDoc()
	AddReviewRecord(doc, "rcliao", DecisionApproved, "", false)
	want := map[string]bool{"p1": true, "p2": true, "p3": false, "q1": false}
	for _, th := range doc.Threads {
		if th.Resolved != want[th.ID] {
			t.Errorf("%s resolved = %v, want %v", th.ID, th.Resolved, want[th.ID])
		}
	}
	replies := doc.Threads[0].Replies
	if len(replies) == 0 {
		t.Fatal("settled pick has no reply saying who accepted it")
	}
	last := replies[len(replies)-1]
	if last.Author != "rcliao" || !strings.Contains(last.Text, "index") {
		t.Errorf("settling reply = %q by %s, want the approver accepting the pick", last.Text, last.Author)
	}
	if n := len(doc.Threads[2].Replies); n != 1 {
		t.Errorf("objected pick gained replies: %d", n)
	}
}

func TestSettlePicksOnlyOnApprove(t *testing.T) {
	for _, decision := range []string{DecisionChangesRequested, DecisionCommented} {
		doc := pickDoc()
		AddReviewRecord(doc, "rcliao", decision, "", false)
		for _, th := range doc.Threads {
			if th.Resolved {
				t.Errorf("%s: %s settled", decision, th.ID)
			}
		}
		if n := len(doc.Threads[0].Replies); n != 0 {
			t.Errorf("%s: pick gained %d replies", decision, n)
		}
	}
}

// With the objection answered, approving leaves the strict gate passing.
func TestSettlePicksPassesStrictGate(t *testing.T) {
	doc := pickDoc()
	doc.Threads = append(doc.Threads[:2], doc.Threads[3])
	doc.Threads[2].Resolved = true
	if d := EvaluateGate(doc, true).Decision; d != DecisionChangesRequested {
		t.Fatalf("strict gate before = %s", d)
	}
	AddReviewRecord(doc, "rcliao", DecisionApproved, "", false)
	if d := EvaluateGate(doc, true).Decision; d != DecisionApproved {
		t.Fatalf("strict gate after approve = %s, want approved", d)
	}
}

// A resolved pick is left alone: no second settling reply.
func TestSettlePicksSkipsResolved(t *testing.T) {
	doc := pickDoc()
	doc.Threads[0].Resolved = true
	AddReviewRecord(doc, "rcliao", DecisionApproved, "", false)
	if n := len(doc.Threads[0].Replies); n != 0 {
		t.Errorf("resolved pick gained %d replies", n)
	}
}

// The agent cannot settle its own pick: a resolve is refused (and posts
// nothing), a plain reply is fine, and the human may still resolve by hand.
func TestSettlePicksNoAgentResolve(t *testing.T) {
	path := writeSurfaceFixture(t, surfaceFixture)
	doc := pickDoc()
	_, err := ReplyToThreads(doc, path, []ReplySpec{{ThreadID: "p1", Author: "claude", Text: "going ahead", Resolve: true}}, ActorAgent)
	if err == nil || !strings.Contains(err.Error(), "pick") {
		t.Fatalf("agent resolve of a pick: err = %v, want a refusal naming pick", err)
	}
	if doc.Threads[0].Resolved || len(doc.Threads[0].Replies) != 0 {
		t.Fatal("a refused resolve changed the pick")
	}
	if _, err := ReplyToThreads(doc, path, []ReplySpec{{ThreadID: "p1", Author: "claude", Text: "measured"}}, ActorAgent); err != nil {
		t.Fatalf("plain reply refused: %v", err)
	}
	if _, err := ReplyToThreads(doc, path, []ReplySpec{{ThreadID: "p1", Resolve: true}}, ActorHuman); err != nil || !doc.Threads[0].Resolved {
		t.Fatalf("human resolve: err = %v, resolved = %v", err, doc.Threads[0].Resolved)
	}
}
