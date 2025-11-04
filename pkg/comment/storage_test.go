package comment

import (
	"os"
	"path/filepath"
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
	loaded, err := LoadFromSidecar(mdPath)
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
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	loaded, err := LoadFromSidecar(mdPath)
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
	doc, err := LoadFromSidecar(mdPath)
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
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Should exist now
	if !SidecarExists(mdPath) {
		t.Error("SidecarExists returned false for existing sidecar")
	}
}

func TestDeleteSidecar(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	// Create sidecar
	doc := &DocumentWithComments{
		Content: "test",
		Threads: []*Comment{},
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Verify it exists
	if !SidecarExists(mdPath) {
		t.Fatal("Sidecar was not created")
	}

	// Delete
	if err := DeleteSidecar(mdPath); err != nil {
		t.Fatalf("DeleteSidecar failed: %v", err)
	}

	// Verify it's gone
	if SidecarExists(mdPath) {
		t.Error("Sidecar still exists after deletion")
	}

	// Delete again should not error
	if err := DeleteSidecar(mdPath); err != nil {
		t.Errorf("DeleteSidecar on non-existent file returned error: %v", err)
	}
}

func TestListSidecars(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple markdown files with sidecars
	files := []string{"doc1.md", "doc2.md", "doc3.md"}
	for _, file := range files {
		mdPath := filepath.Join(tmpDir, file)
		doc := &DocumentWithComments{
			Content: "content",
			Threads: []*Comment{},
		}
		if err := SaveToSidecar(mdPath, doc); err != nil {
			t.Fatalf("SaveToSidecar failed for %s: %v", file, err)
		}
	}

	// Create a non-sidecar JSON file (should be ignored)
	otherJSON := filepath.Join(tmpDir, "other.json")
	if err := os.WriteFile(otherJSON, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create other.json: %v", err)
	}

	// List sidecars
	sidecars, err := ListSidecars(tmpDir)
	if err != nil {
		t.Fatalf("ListSidecars failed: %v", err)
	}

	// Should find exactly 3 sidecar files
	if len(sidecars) != 3 {
		t.Errorf("Expected 3 sidecars, got %d: %v", len(sidecars), sidecars)
	}

	// Verify they all end with .comments.json
	for _, sidecar := range sidecars {
		if filepath.Ext(sidecar) != ".json" {
			t.Errorf("Sidecar has wrong extension: %s", sidecar)
		}
		name := filepath.Base(sidecar)
		if len(name) <= len(".comments.json") || name[len(name)-len(".comments.json"):] != ".comments.json" {
			t.Errorf("Sidecar doesn't end with .comments.json: %s", sidecar)
		}
	}
}

func TestComputeDocumentHash(t *testing.T) {
	content1 := "# Test\n\nContent"
	content2 := "# Test\n\nContent" // Same as content1
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
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Load
	loaded, err := LoadFromSidecar(mdPath)
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

	// Save
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatalf("SaveToSidecar failed: %v", err)
	}

	// Read markdown file
	writtenContent, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	if string(writtenContent) != content {
		t.Errorf("Markdown content mismatch.\nExpected: %q\nGot: %q", content, string(writtenContent))
	}
}
