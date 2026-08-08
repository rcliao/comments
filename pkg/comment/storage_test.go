package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	// Create test document
	content := "# Test Document\n\nSome content here.\nMore content.\n"
	timestamp := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{
			{
				ID:        "c1",
				Author:    "alice",
				Line:      3,
				Timestamp: timestamp,
				Text:      "This is a question",
				Type:      "Q",
				Resolved:  false,
				Replies: []*Comment{
					{
						ID:        "c2",
						Author:    "bob",
						Line:      3,
						Timestamp: timestamp.Add(5 * time.Minute),
						Text:      "Here's an answer",
						Type:      "",
						Resolved:  false,
						Replies:   []*Comment{},
					},
				},
			},
		},
	}

	// Save
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Verify markdown file exists
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Fatal("Markdown file was not created")
	}

	// Verify sidecar file exists
	sidecarPath := GetSidecarPath(mdPath)
	if _, err := os.Stat(sidecarPath); os.IsNotExist(err) {
		t.Fatal("Sidecar file was not created")
	}

	// Load
	loaded, _, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatalf("LoadFromSidecar failed: %v", err)
	}

	// Verify content
	if loaded.Content != content {
		t.Errorf("Content mismatch.\nExpected: %q\nGot: %q", content, loaded.Content)
	}

	// Verify threads count
	if len(loaded.Threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(loaded.Threads))
	}

	// Verify root comment
	c1 := loaded.Threads[0]
	if c1.ID != "c1" || c1.Author != "alice" || c1.Text != "This is a question" {
		t.Errorf("Root comment mismatch: %+v", c1)
	}

	// Verify reply
	if len(c1.Replies) != 1 {
		t.Fatalf("Expected 1 reply, got %d", len(c1.Replies))
	}
	c2 := c1.Replies[0]
	if c2.ID != "c2" || c2.Author != "bob" {
		t.Errorf("Reply mismatch: %+v", c2)
	}

	// Verify document hash was computed
	if loaded.DocumentHash == "" {
		t.Error("DocumentHash should not be empty")
	}
}

func TestSaveAndLoadWithSuggestion(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	content := "# Test\n\nOriginal line here\n"
	timestamp := time.Now()

	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{
			{
				ID:           "s1",
				Author:       "claude",
				Line:         3,
				Timestamp:    timestamp,
				Text:         "Suggest changing this line",
				Type:         "S",
				Resolved:     false,
				Replies:      []*Comment{},
				IsSuggestion: true,
				StartLine:    3,
				EndLine:      3,
				OriginalText: "Original line here",
				ProposedText: "Improved line here",
				Accepted:     nil, // Pending
			},
		},
	}

	// Save and load
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	loaded, _, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatalf("LoadFromSidecar failed: %v", err)
	}

	// Verify suggestion fields
	if len(loaded.Threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(loaded.Threads))
	}

	s1 := loaded.Threads[0]
	if !s1.IsSuggestion {
		t.Error("Expected IsSuggestion to be true")
	}
	if s1.ProposedText != "Improved line here" {
		t.Errorf("ProposedText mismatch: got %q", s1.ProposedText)
	}
	if s1.OriginalText != "Original line here" {
		t.Errorf("OriginalText mismatch: got %q", s1.OriginalText)
	}
	if !s1.IsPending() {
		t.Error("Suggestion should be pending")
	}
	if s1.StartLine != 3 {
		t.Errorf("StartLine = %d, want 3", s1.StartLine)
	}
	if s1.EndLine != 3 {
		t.Errorf("EndLine = %d, want 3", s1.EndLine)
	}
}

func TestLoadNonExistentSidecar(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	// Create markdown file but no sidecar
	content := "# Test\n\nContent\n"
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load should succeed with empty threads
	doc, _, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatalf("LoadFromSidecar failed: %v", err)
	}

	if doc.Content != content {
		t.Errorf("Content mismatch")
	}
	if len(doc.Threads) != 0 {
		t.Errorf("Expected 0 threads, got %d", len(doc.Threads))
	}
}

func TestGetSidecarPath(t *testing.T) {
	tests := []struct {
		mdPath   string
		expected string
	}{
		{"doc.md", "doc.md.comments.json"},
		{"/path/to/doc.md", "/path/to/doc.md.comments.json"},
		{"README.md", "README.md.comments.json"},
	}

	for _, tt := range tests {
		got := GetSidecarPath(tt.mdPath)
		if got != tt.expected {
			t.Errorf("GetSidecarPath(%q) = %q, want %q", tt.mdPath, got, tt.expected)
		}
	}
}

func TestSidecarExists(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	// Should not exist initially
	if SidecarExists(mdPath) {
		t.Error("SidecarExists returned true for non-existent sidecar")
	}

	// Create sidecar
	doc := &DocumentWithComments{
		Content: "test",
		Threads: []*Comment{},
	}
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Should exist now
	if !SidecarExists(mdPath) {
		t.Error("SidecarExists returned false for existing sidecar")
	}
}

func TestComputeDocumentHash(t *testing.T) {
	content1 := "# Test\n\nContent"
	content2 := "# Test\n\nContent"   // Same as content1
	content3 := "# Test\n\nDifferent" // Different

	hash1 := ComputeDocumentHash(content1)
	hash2 := ComputeDocumentHash(content2)
	hash3 := ComputeDocumentHash(content3)

	// Same content should produce same hash
	if hash1 != hash2 {
		t.Errorf("Same content produced different hashes: %s vs %s", hash1, hash2)
	}

	// Different content should produce different hash
	if hash1 == hash3 {
		t.Error("Different content produced same hash")
	}

	// Hash should be hex string
	if len(hash1) != 64 { // SHA-256 produces 64 hex chars
		t.Errorf("Hash length = %d, want 64", len(hash1))
	}
}

func TestSaveUpdatesDocumentHash(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	content := "# Test\n\nContent\n"
	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{},
	}

	// Save
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Verify hash was computed
	if doc.DocumentHash == "" {
		t.Error("DocumentHash should be set after save")
	}

	expectedHash := ComputeDocumentHash(content)
	if doc.DocumentHash != expectedHash {
		t.Errorf("DocumentHash = %s, want %s", doc.DocumentHash, expectedHash)
	}
}

func TestNestedRepliesSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	// Create deeply nested thread structure
	doc := &DocumentWithComments{
		Content: "# Test\n\nContent\n",
		Threads: []*Comment{
			{
				ID:     "c1",
				Author: "alice",
				Line:   1,
				Text:   "Root comment",
				Replies: []*Comment{
					{
						ID:     "c2",
						Author: "bob",
						Line:   1,
						Text:   "Reply 1",
						Replies: []*Comment{
							{
								ID:      "c3",
								Author:  "charlie",
								Line:    1,
								Text:    "Nested reply",
								Replies: []*Comment{},
							},
						},
					},
					{
						ID:      "c4",
						Author:  "dave",
						Line:    1,
						Text:    "Reply 2",
						Replies: []*Comment{},
					},
				},
			},
		},
	}

	// Save
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Load
	loaded, _, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatalf("LoadFromSidecar failed: %v", err)
	}

	// Verify structure
	if len(loaded.Threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(loaded.Threads))
	}

	root := loaded.Threads[0]
	if root.ID != "c1" {
		t.Errorf("Root ID = %s, want c1", root.ID)
	}

	if len(root.Replies) != 2 {
		t.Fatalf("Expected 2 replies to root, got %d", len(root.Replies))
	}

	if root.Replies[0].ID != "c2" {
		t.Errorf("First reply ID = %s, want c2", root.Replies[0].ID)
	}

	if len(root.Replies[0].Replies) != 1 {
		t.Fatalf("Expected 1 nested reply, got %d", len(root.Replies[0].Replies))
	}

	if root.Replies[0].Replies[0].ID != "c3" {
		t.Errorf("Nested reply ID = %s, want c3", root.Replies[0].Replies[0].ID)
	}

	if root.Replies[1].ID != "c4" {
		t.Errorf("Second reply ID = %s, want c4", root.Replies[1].ID)
	}
}

func TestSaveToSidecarWritesMarkdownContent(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	content := "# Test Document\n\nThis is the content.\n"
	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{},
	}

	// Disk holds NEWER content than this doc's memory (an agent edited it)
	onDisk := "# Test Document\n\nEdited on disk after this session loaded.\n"
	if err := os.WriteFile(mdPath, []byte(onDisk), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// SaveToSidecar must NOT have touched the markdown: a stale in-memory
	// Content overwriting disk was a live lost-update bug (a TUI signoff
	// reverted agent edits). Content writes are SaveDocumentContent's job.
	writtenContent, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}
	if string(writtenContent) != onDisk {
		t.Errorf("SaveToSidecar overwrote the markdown from memory.\nDisk had: %q\nNow: %q", onDisk, string(writtenContent))
	}

	// SaveDocumentContent is the explicit content writer
	if err := SaveDocumentContent(mdPath, doc); err != nil {
		t.Fatalf("SaveDocumentContent failed: %v", err)
	}
	writtenContent, _ = os.ReadFile(mdPath)
	if string(writtenContent) != content {
		t.Errorf("SaveDocumentContent mismatch.\nExpected: %q\nGot: %q", content, string(writtenContent))
	}
}

// TestLoadFromSidecarIsReadOnly verifies that loading never writes files, even
// when the document changed out-of-band: the migration is reported instead.
func TestLoadFromSidecarIsReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	content := "# Title\n\nAnchored line here.\n"
	c := &Comment{ID: "c1", Author: "alice", Line: 3, Text: "note", Replies: []*Comment{}}
	c.Anchor = CaptureAnchor(content, 3)
	doc := &DocumentWithComments{
		Content: content,
		Threads: []*Comment{c},
	}
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Edit the markdown out-of-band so the load must re-anchor
	newContent := "# Title\n\nInserted line.\n\nAnchored line here.\n"
	if err := os.WriteFile(mdPath, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	sidecarPath := GetSidecarPath(mdPath)
	before, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatal(err)
	}

	loaded, report, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatalf("LoadFromSidecar failed: %v", err)
	}

	after, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("LoadFromSidecar modified the sidecar on disk; loads must be read-only")
	}

	if !report.Stale {
		t.Error("expected report.Stale after out-of-band edit")
	}
	if !report.Dirty {
		t.Error("expected report.Dirty when the load re-anchored a comment")
	}
	if loaded.Threads[0].Line != 5 {
		t.Errorf("expected comment re-anchored to line 5 in memory, got %d", loaded.Threads[0].Line)
	}
}

// TestSaveSkipsUnchangedMarkdown verifies the markdown file is rewritten only
// when doc.Content actually differs from what is on disk.
func TestSaveSkipsUnchangedMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	doc := &DocumentWithComments{
		Content: "# Title\n\nBody.\n",
		Threads: []*Comment{},
	}
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()

	// Identical content must not touch the markdown file (mtime preserved)
	time.Sleep(10 * time.Millisecond)
	if err := SaveDocumentContent(mdPath, doc); err != nil {
		t.Fatalf("SaveDocumentContent failed: %v", err)
	}
	if info, err = os.Stat(mdPath); err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Error("markdown file was rewritten although the content did not change")
	}

	// A content change must be written
	doc.Content = "# Title\n\nBody changed.\n"
	if err := SaveDocumentContent(mdPath, doc); err != nil {
		t.Fatalf("SaveDocumentContent failed: %v", err)
	}
	written, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != doc.Content {
		t.Errorf("markdown not updated on content change: %q", string(written))
	}
}


func TestLoadFromSidecarMissingMarkdown(t *testing.T) {
	_, _, err := LoadFromSidecar(filepath.Join(t.TempDir(), "nope.md"))
	if err == nil {
		t.Fatal("expected error for missing markdown file")
	}
	if !strings.Contains(err.Error(), "failed to read markdown file") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLoadFromSidecarCorruptJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "doc.md")
	if err := os.WriteFile(mdPath, []byte("# Doc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GetSidecarPath(mdPath), []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadFromSidecar(mdPath)
	if err == nil {
		t.Fatal("expected error for corrupt sidecar JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse sidecar JSON") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestLoadFromSidecarVersionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "doc.md")
	if err := os.WriteFile(mdPath, []byte("# Doc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sidecar := `{"version":"1.0","documentHash":"x","threads":[]}`
	if err := os.WriteFile(GetSidecarPath(mdPath), []byte(sidecar), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadFromSidecar(mdPath)
	if err == nil {
		t.Fatal("expected error for version mismatch")
	}
	if !strings.Contains(err.Error(), "unsupported storage version: 1.0") {
		t.Errorf("wrong error: %v", err)
	}
}
