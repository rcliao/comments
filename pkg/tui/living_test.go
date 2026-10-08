package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

const livingTestDoc = `---
comments:
    template: living
---

# Work

## Now

- State: building.

## Why

Because.
`

// openLiving writes doc to a temp file with a sidecar and opens it the way
// `comments view` does.
func openLiving(t *testing.T, path, content string) Model {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, _, err := comment.LoadFromSidecar(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := comment.SaveToSidecar(path, doc); err != nil {
		t.Fatal(err)
	}
	m := NewModelWithFile(doc, path)
	m.author = "reader"
	m.refreshChangedLines()
	m.width, m.height = 100, 40
	m.handleResize()
	return m
}

// q on a living doc records what you saw and quits: no verdict dialog, no
// review record. The next open tints only what changed since.
func TestLivingQuitRecordsSeenAndTintsOnlyNewEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "work.md")
	m := openLiving(t, path, livingTestDoc)
	next, cmd := m.handleBrowseKeys(keyMsg("q"))
	nm := next.(Model)
	if nm.mode == ModeVerdict || cmd == nil {
		t.Fatalf("q on a living doc should quit, got mode %v", nm.mode)
	}
	if loaded, _, _ := comment.LoadFromSidecar(path); len(loaded.Reviews) != 0 {
		t.Fatalf("q recorded a review: %v", loaded.Reviews)
	}

	edited := strings.Replace(livingTestDoc, "- State: building.", "- State: done.", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, _, err := comment.LoadFromSidecar(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewModelWithFile(doc, path)
	reopened.author = "reader"
	reopened.refreshChangedLines()
	stateLine := 0
	for i, line := range strings.Split(edited, "\n") {
		if strings.Contains(line, "State: done") {
			stateLine = i + 1
		}
	}
	if len(reopened.changedLines) != 1 || !reopened.changedLines[stateLine] {
		t.Fatalf("changed lines = %v, want only line %d", reopened.changedLines, stateLine)
	}
}

// Other docs keep q as the verdict entry.
func TestNonLivingQuitStillOpensVerdict(t *testing.T) {
	m := testModel(nil)
	next, _ := m.handleBrowseKeys(keyMsg("q"))
	if next.(Model).mode != ModeVerdict {
		t.Fatal("q on a review doc should open the verdict dialog")
	}
}

// A view left open picks up the agent's edits, but never while the human is
// typing: the reload waits for browse.
func TestLiveReloadOnlyInReadingModes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "work.md")
	m := openLiving(t, path, livingTestDoc)
	edited := strings.Replace(livingTestDoc, "Because.", "Because, rewritten.", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	m.mode = ModeAddComment
	next, cmd := m.Update(reloadTickMsg{})
	typing := next.(Model)
	if cmd == nil {
		t.Fatal("the tick must reschedule itself")
	}
	if strings.Contains(typing.doc.Content, "rewritten") {
		t.Fatal("reloaded while the human was typing")
	}

	typing.mode = ModeBrowse
	next, _ = typing.Update(reloadTickMsg{})
	if got := next.(Model).doc.Content; !strings.Contains(got, "rewritten") {
		t.Fatalf("browse did not reload the edit:\n%s", got)
	}
}

// A doc with no sidecar (hand-written, opened by `comments view`) still
// reloads when its text changes.
func TestLiveReloadWithoutSidecar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.md")
	if err := os.WriteFile(path, []byte(livingTestDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	m := NewModelWithFile(&comment.DocumentWithComments{Content: livingTestDoc}, path)
	m.width, m.height = 100, 40
	m.handleResize()
	if err := os.WriteFile(path, []byte(livingTestDoc+"\nAdded.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(reloadTickMsg{})
	if !strings.Contains(next.(Model).doc.Content, "Added.") {
		t.Fatal("no-sidecar doc did not reload")
	}
}

// An agent reply lands in the open thread panel without the human moving.
func TestLiveReloadRedrawsOpenThread(t *testing.T) {
	path := filepath.Join(t.TempDir(), "work.md")
	m := openLiving(t, path, livingTestDoc)
	disk, _, _ := comment.LoadFromSidecar(path)
	disk.Threads = []*comment.Comment{{ID: "c1", Line: 9, Author: "reader", Text: "Is this done?"}}
	if err := comment.SaveToSidecar(path, disk); err != nil {
		t.Fatal(err)
	}
	m.refreshDocFromDisk()
	m.diskStamp = diskStamp(path)
	m.selectedThread = m.doc.Threads[0]
	m.mode = ModeThreadView
	m.applyThreadPanel()

	if err := comment.AddReplyToThread(disk.Threads, "c1", "claude", "Yes, shipped in slice two."); err != nil {
		t.Fatal(err)
	}
	if err := comment.SaveToSidecar(path, disk); err != nil {
		t.Fatal(err)
	}
	next, _ := m.Update(reloadTickMsg{})
	if view := next.(Model).threadViewport.View(); !strings.Contains(view, "shipped in slice two") {
		t.Fatalf("open thread panel did not show the new reply:\n%s", view)
	}
}
