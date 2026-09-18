package comment

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeDoc(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func watchOnce(t *testing.T, path string, opts WatchOptions) []WatchEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var events []WatchEvent
	_ = Watch(ctx, path, opts, func(e WatchEvent) bool {
		events = append(events, e)
		return false
	})
	return events
}

// The hand-off race: the agent says "please review", the human is fast, and
// the verdict lands BEFORE the agent's watch takes its first look. A watch only
// reports changes after that look, so without Since the agent waits forever on
// a review that already happened.
func TestWatchSinceReturnsAVerdictRecordedBeforeTheWatchStarted(t *testing.T) {
	path := writeDoc(t, "# Doc\n\nBody.\n")
	handoff := time.Now().Add(-time.Second)
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RecordVerdict(path, doc, "eric", DecisionChangesRequested, "pin the prompt"); err != nil {
		t.Fatal(err)
	}

	events := watchOnce(t, path, WatchOptions{Interval: 10 * time.Millisecond, Until: "signoff", Since: handoff})
	if len(events) != 1 || events[0].Event != "signoff" || events[0].Author != "eric" ||
		events[0].Decision != DecisionChangesRequested || events[0].Note != "pin the prompt" {
		t.Fatalf("want the already-recorded verdict at once, got %+v", events)
	}

	// A verdict from BEFORE the hand-off is an old round, not an answer.
	stale := watchOnce(t, path, WatchOptions{Interval: 10 * time.Millisecond, Until: "signoff", Since: time.Now().Add(time.Hour)})
	if len(stale) != 0 {
		t.Errorf("a verdict older than since must not be reported, got %+v", stale)
	}
}

// A document with no sidecar yet is an empty baseline. It used to be skipped,
// which made the review that CREATES the sidecar the first sighting — so its
// comments and its verdict were swallowed as the baseline.
func TestWatchSeesTheReviewThatCreatesTheSidecar(t *testing.T) {
	path := writeDoc(t, "# Doc\n\nBody.\n")
	go func() {
		time.Sleep(150 * time.Millisecond)
		doc, _, err := LoadDocument(path)
		if err != nil {
			return
		}
		_, _ = AddComments(doc, []NewCommentSpec{{Line: 3, Author: "eric", Text: "tighten this"}})
		_, _ = RecordVerdict(path, doc, "eric", DecisionChangesRequested, "")
	}()
	events := watchOnce(t, path, WatchOptions{Interval: 10 * time.Millisecond, Until: "signoff"})
	kinds := map[string]bool{}
	for _, e := range events {
		kinds[e.Event] = true
	}
	if !kinds["comment_added"] || !kinds["signoff"] {
		t.Errorf("want comment_added and signoff from the first review, got %+v", events)
	}
}

// The guard fails CLOSED. A template name that will not load leaves the zones
// unknown; treating that as "no zones" let a frontmatter typo make every
// human-decision thread agent-resolvable, with no signal.
func TestZoneGuardFailsClosedWhenTheTemplateWillNotLoad(t *testing.T) {
	path := writeDoc(t, "---\ncomments:\n  template: no-such-template\n---\n\n# Doc\n\n## Problem\n\nIt is slow.\n")
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	added, err := AddComments(doc, []NewCommentSpec{{Anchor: "It is slow.", Author: "eric", Text: "why?"}})
	if err != nil {
		t.Fatal(err)
	}
	id := added.Added[0].ID

	err = GuardZoneResolve(doc, path, id, ActorAgent)
	if err == nil || !strings.Contains(err.Error(), "could not be loaded") {
		t.Errorf("an agent resolve must be refused when zones are unknown, got %v", err)
	}
	if err := GuardZoneResolve(doc, path, id, ActorHuman); err != nil {
		t.Errorf("the human is never blocked: %v", err)
	}
}

// A refusal that explains how to get past itself is not a refusal: over MCP the
// error text goes straight to the agent.
func TestZoneRefusalDoesNotNameTheOverride(t *testing.T) {
	path := writeDoc(t, "---\ncomments:\n  template: design-doc\n---\n\n# Doc\n\n## Problem\n\nIt is slow.\n")
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	added, err := AddComments(doc, []NewCommentSpec{{Anchor: "It is slow.", Author: "eric", Text: "why?"}})
	if err != nil {
		t.Fatal(err)
	}
	err = GuardZoneResolve(doc, path, added.Added[0].ID, ActorAgent)
	if err == nil || !strings.Contains(err.Error(), "human-decision zone") {
		t.Fatalf("want a zone refusal, got %v", err)
	}
	if strings.Contains(err.Error(), ActorEnvVar) {
		t.Errorf("the refusal names the override: %v", err)
	}
}

func TestAddCommentsRejectsBadPriorityAndOutOfRangeLine(t *testing.T) {
	path := writeDoc(t, "# Doc\n\nBody.\n")
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, spec := range map[string]NewCommentSpec{
		"invalid priority": {Line: 3, Author: "a", Text: "x", Priority: "urgent"},
		"outside the":      {Line: 99, Author: "a", Text: "x"},
		"outside the doc":  {Line: -1, Author: "a", Text: "x"},
	} {
		if _, err := AddComments(doc, []NewCommentSpec{{Line: 3, Author: "a", Text: "fine"}, spec}); err == nil ||
			!strings.Contains(err.Error(), "comment 2") || !strings.Contains(err.Error(), strings.TrimSuffix(name, " doc")) {
			t.Errorf("%s: want a refusal naming comment 2, got %v", name, err)
		}
	}
	if len(doc.Threads) != 0 {
		t.Errorf("a refused batch must add nothing, got %d thread(s)", len(doc.Threads))
	}
}

const zoneDoc = "---\ncomments:\n  template: design-doc\n---\n\n# Doc\n\n## Pitch\n\nCache it.\n\n## Problem\n\nIt is slow.\n\n## Proposed Design\n\nA cache in front of the store.\nEntries expire after sixty seconds.\n"

// The resolve guard keys on where a thread sits, so an agent that may choose
// that can walk a human-decision thread into its own zone and close it:
// reanchor out of `## Pitch`, then reply --resolve. Found in review of the lean
// surface; both halves were individually "allowed".
func TestAgentCannotReanchorAThreadOutOfAHumanZone(t *testing.T) {
	path := writeDoc(t, zoneDoc)
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	added, err := AddComments(doc, []NewCommentSpec{{Anchor: "Cache it.", Author: "eric", Text: "is this the decision?", Blocking: true}})
	if err != nil {
		t.Fatal(err)
	}
	id := added.Added[0].ID

	out := ApplyMoves(doc, path, []Move{{CommentID: id, Section: "Doc > Proposed Design"}}, ActorAgent)
	if out[0].Moved || !strings.Contains(out[0].Error, "human-decision zone") {
		t.Fatalf("an agent moved a thread out of a human zone: %+v", out[0])
	}
	if _, err := ReplyToThreads(doc, path, []ReplySpec{{ThreadID: id, Resolve: true}}, ActorAgent); err == nil {
		t.Fatal("the thread must still be unresolvable by an agent")
	}

	// Within the zone is fine, and the human may move it anywhere.
	if out := ApplyMoves(doc, path, []Move{{CommentID: id, Section: "Doc > Problem"}}, ActorAgent); !out[0].Moved {
		t.Errorf("a move that stays in a human zone must be allowed: %+v", out[0])
	}
	if out := ApplyMoves(doc, path, []Move{{CommentID: id, Section: "Doc > Proposed Design"}}, ActorHuman); !out[0].Moved {
		t.Errorf("the human may move any thread: %+v", out[0])
	}
}

// A declared move is the agent's required post-edit step, so it must carry a
// suggestion's range the same way the automatic cascade does.
func TestDeclaredMoveCarriesASuggestionsRange(t *testing.T) {
	path := writeDoc(t, zoneDoc)
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := AddSuggestion(doc, SuggestionSpec{Author: "claude", Text: "rework", Anchor: "A cache in front of the store.",
		OriginalText: "A cache in front of the store.\nEntries expire after sixty seconds.", ProposedText: "A read-through cache."})
	if err != nil {
		t.Fatal(err)
	}
	if res.StartLine != 18 || res.EndLine != 19 {
		t.Fatalf("fixture drifted: %+v", res)
	}
	ApplyMoves(doc, path, []Move{{CommentID: res.SuggestionID, Line: 10}}, ActorHuman)
	s := doc.FindCommentByID(res.SuggestionID)
	if s.Line != 10 || s.StartLine != 10 || s.EndLine != 11 {
		t.Errorf("range did not travel with the move: line %d, range %d-%d; want 10, 10-11", s.Line, s.StartLine, s.EndLine)
	}
}

// When a suggestion's target text is gone there is nothing left to replace. It
// used to fall back to the section heading and stay acceptable, so an accept
// replaced whatever now sat on its old lines.
func TestSuggestionWhoseTextIsGoneCannotBeApplied(t *testing.T) {
	path := writeDoc(t, zoneDoc)
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := AddSuggestion(doc, SuggestionSpec{Author: "claude", Text: "ttl", Anchor: "Entries expire after sixty seconds.", ProposedText: "Per-route TTL."})
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(path, doc); err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(zoneDoc, "Entries expire after sixty seconds.", "Expiry is handled elsewhere now.", 1)
	if err := os.WriteFile(path, []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, _, err = LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	s := doc.FindCommentByID(res.SuggestionID)
	if !s.IsOrphaned() {
		t.Fatalf("want the suggestion orphaned, got status %q confidence %q at line %d", s.Status, s.AnchorConfidence, s.Line)
	}
	if _, err := ApplySuggestion(doc.Content, s); err == nil || !strings.Contains(err.Error(), "can no longer be applied") {
		t.Errorf("an orphaned suggestion must refuse to apply, got %v", err)
	}
}

// Neither human surface resolves a suggestion it rejects, so the inbox dropping
// decided suggestions made the human's "no" invisible to the agent.
func TestInboxShowsARejectedSuggestionUntilTheAgentClosesIt(t *testing.T) {
	path := writeDoc(t, zoneDoc)
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	// In `## Problem`, a human zone: the agent must still be able to close it.
	res, err := AddSuggestion(doc, SuggestionSpec{Author: "claude", Text: "sharper", Anchor: "It is slow.", ProposedText: "Reads take four seconds."})
	if err != nil {
		t.Fatal(err)
	}
	if err := RejectSuggestion(doc.Threads, res.SuggestionID); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(path, doc); err != nil {
		t.Fatal(err)
	}

	inbox, err := BuildInbox(path, InboxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inbox.Count != 1 || inbox.Items[0].Reasons[0] != "suggestion_rejected" || len(inbox.PendingSuggestions) != 0 {
		t.Fatalf("want one suggestion_rejected item and nothing pending, got %+v", inbox)
	}

	doc, _, _ = LoadDocument(path)
	if _, err := ReplyToThreads(doc, path, []ReplySpec{{ThreadID: res.SuggestionID, Author: "claude", Text: "Understood, dropping it.", Resolve: true}}, ActorAgent); err != nil {
		t.Fatalf("the human already decided; the agent must be able to close it: %v", err)
	}
	if err := SaveToSidecar(path, doc); err != nil {
		t.Fatal(err)
	}
	if inbox, _ = BuildInbox(path, InboxOptions{}); inbox.Count != 0 {
		t.Errorf("a closed rejected suggestion must leave the inbox, got %+v", inbox.Items)
	}
}
