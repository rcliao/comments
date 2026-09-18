package comment

// Behaviours that used to be pinned through MCP tools the lean surface removed
// (comments_gate, comments_status, comments_list filters). The behaviour is
// still there — the CLI, the TUI and comments_inbox all reach it — so the pins
// moved down to the core functions every surface now shares.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const surfaceFixture = "# Fixture\n\n## Problem\n\nIt is slow.\n\n## Notes\n\nSome notes.\n"

func writeSurfaceFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// mutateSurfaceDoc loads, applies fn and saves — the same prelude every
// surface's mutating handler runs. The gate and inbox builders re-read disk,
// so each step has to be persisted before the next report.
func mutateSurfaceDoc(t *testing.T, path string, fn func(doc *DocumentWithComments)) {
	t.Helper()
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	fn(doc)
	if err := SaveToSidecar(path, doc); err != nil {
		t.Fatal(err)
	}
}

// An open blocking thread fails the gate; reply+resolve — the one respond path
// an agent has — clears it.
func TestGateReportBlockingLifecycle(t *testing.T) {
	path := writeSurfaceFixture(t, surfaceFixture)
	var id string
	mutateSurfaceDoc(t, path, func(doc *DocumentWithComments) {
		added, err := AddComments(doc, []NewCommentSpec{{Author: "eric", Text: "must fix", Line: 5, Blocking: true}})
		if err != nil {
			t.Fatal(err)
		}
		id = added.Added[0].ID
	})

	report, err := BuildGateReport(path, false, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionChangesRequested || report.Summary.Blocking != 1 {
		t.Errorf("expected changes_requested with open blocking comment, got %s (%+v)", report.Decision, report.Summary)
	}
	// No template anywhere: the pass/fail above is comment state only, and the
	// report has to say so rather than read as a structural pass
	if !report.Files[0].StructureUnchecked {
		t.Error("untemplated doc with threads must report structure_unchecked")
	}

	mutateSurfaceDoc(t, path, func(doc *DocumentWithComments) {
		if _, err := ReplyToThreads(doc, path, []ReplySpec{{ThreadID: id, Author: "claude", Text: "fixed", Resolve: true}}, ActorAgent); err != nil {
			t.Fatal(err)
		}
	})

	report, err = BuildGateReport(path, false, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if report.Decision != DecisionApproved || report.Summary.Blocking != 0 {
		t.Errorf("expected approved after resolve, got %s (%+v)", report.Decision, report.Summary)
	}
}

// Was comments_status: root threads and nested replies are counted apart, so a
// busy thread does not read as many threads.
func TestListThreadsCountsRootsNotReplies(t *testing.T) {
	path := writeSurfaceFixture(t, surfaceFixture)
	mutateSurfaceDoc(t, path, func(doc *DocumentWithComments) {
		added, err := AddComments(doc, []NewCommentSpec{
			{Author: "eric", Text: "first thread", Line: 5},
			{Author: "eric", Text: "second thread", Line: 9},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ReplyToThreads(doc, path, []ReplySpec{
			{ThreadID: added.Added[0].ID, Author: "claude", Text: "a reply"},
			{ThreadID: added.Added[1].ID, Resolve: true},
		}, ActorAgent); err != nil {
			t.Fatal(err)
		}
	})

	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	all := ListThreads(doc, path, false)
	if all.Total != 2 || len(all.Comments) != 2 {
		t.Fatalf("expected 2 root threads, got %d", all.Total)
	}
	if all.Comments[0].ReplyCount != 1 || len(all.Comments[0].Replies) != 1 {
		t.Errorf("reply should nest under its root, got %+v", all.Comments[0])
	}
	if got := len(doc.GetAllComments()); got != 3 {
		t.Errorf("expected 3 total comments (2 roots + 1 reply), got %d", got)
	}
	open := ListThreads(doc, path, true)
	if open.Total != 1 || open.Comments[0].Text != "first thread" {
		t.Errorf("expected 1 unresolved root thread, got %+v", open.Comments)
	}
}

// Was comments_status changed_since, now files[].changes on the inbox: lines,
// deletions and the innermost sections touched since a reviewer's verdict
// baseline — and absent entirely for a reviewer who never signed off, because
// "no key" means no baseline, not "unchanged".
func TestInboxReportsChangedSinceReviewerVerdict(t *testing.T) {
	path := writeSurfaceFixture(t, surfaceFixture)
	// The inbox only covers documents with a sidecar
	mutateSurfaceDoc(t, path, func(doc *DocumentWithComments) {
		if _, err := AddComments(doc, []NewCommentSpec{{Author: "eric", Text: "note", Line: 5}}); err != nil {
			t.Fatal(err)
		}
	})

	before, err := BuildInbox(path, InboxOptions{Reviewer: "eric"})
	if err != nil {
		t.Fatal(err)
	}
	if before.Files[0].Changes != nil {
		t.Fatalf("no baseline → changes must be absent, got %+v", before.Files[0].Changes)
	}

	if err := SaveReviewBaseline(path, "eric", surfaceFixture); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(surfaceFixture+"\n## Added\n\nNew section body.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inbox, err := BuildInbox(path, InboxOptions{Reviewer: "eric"})
	if err != nil {
		t.Fatal(err)
	}
	cs := inbox.Files[0].Changes
	if cs == nil {
		t.Fatalf("changes missing: %+v", inbox.Files[0])
	}
	if cs.Reviewer != "eric" || cs.Lines < 3 || cs.Deleted != 0 {
		t.Errorf("changes = %+v", cs)
	}
	if len(cs.Sections) == 0 || !strings.HasSuffix(cs.Sections[len(cs.Sections)-1], "Added") {
		t.Errorf("changed_sections should name the added section, got %v", cs.Sections)
	}
	// The thread survived the out-of-band edit with its anchor intact
	if inbox.Files[0].Orphaned != 0 || inbox.Count != 1 {
		t.Errorf("appending a section must not orphan the thread: orphaned %d, count %d", inbox.Files[0].Orphaned, inbox.Count)
	}
}

// With no reviewer named, the inbox diffs against whoever signed off last —
// the agent rarely knows the reviewer's name, and must still see what moved.
func TestInboxChangesDefaultToLatestReviewer(t *testing.T) {
	path := writeSurfaceFixture(t, surfaceFixture)
	mutateSurfaceDoc(t, path, func(doc *DocumentWithComments) {
		if _, err := AddComments(doc, []NewCommentSpec{{Author: "eric", Text: "note", Line: 5}}); err != nil {
			t.Fatal(err)
		}
		AddReviewRecord(doc, "eric", "", "first pass", false)
	})
	if err := SaveReviewBaseline(path, "eric", surfaceFixture); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(surfaceFixture+"\nOne more line.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inbox, err := BuildInbox(path, InboxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	file := inbox.Files[0]
	if file.LastReview == nil || file.LastReview.Author != "eric" {
		t.Fatalf("last_review missing: %+v", file)
	}
	if file.Changes == nil || file.Changes.Reviewer != "eric" || file.Changes.Lines == 0 {
		t.Errorf("changes should default to the latest reviewer, got %+v", file.Changes)
	}
}

// Was the comments_list section filter. TestGetCommentsInSection covers own +
// descendant + unknown; what it cannot show (its fixture has no such pair) is
// that a sibling whose title merely shares a string prefix is excluded.
func TestGetCommentsInSectionExcludesPrefixSibling(t *testing.T) {
	doc := &DocumentWithComments{
		Content: "# Doc\n\n## Problem\n\nProblem body.\n\n### Symptoms\n\nSymptom body.\n\n## Problem Details\n\nDetails body.\n",
		Threads: []*Comment{
			{ID: "problem", Line: 5},
			{ID: "symptoms", Line: 9},
			{ID: "details", Line: 13},
		},
	}
	ComputeSectionsForComments(doc)

	ids := map[string]bool{}
	for _, c := range GetCommentsInSection(doc, "Doc > Problem") {
		ids[c.ID] = true
	}
	if len(ids) != 2 || !ids["problem"] || !ids["symptoms"] {
		t.Errorf("expected section + descendant comments, got %v", ids)
	}
	if ids["details"] {
		t.Error("sibling section 'Problem Details' must not match filter 'Doc > Problem'")
	}
	if got := GetCommentsInSection(doc, "Doc"); len(got) != 3 {
		t.Errorf("expected all 3 comments under root section, got %d", len(got))
	}
}
