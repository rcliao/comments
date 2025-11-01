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
		Comments: []*Comment{
			{
				ID:        "c1",
				ThreadID:  "c1",
				ParentID:  "",
				Author:    "alice",
				Line:      3,
				Timestamp: timestamp,
				Text:      "This is a question",
				Type:      "Q",
				Resolved:  false,
			},
			{
				ID:        "c2",
				ThreadID:  "c1",
				ParentID:  "c1",
				Author:    "bob",
				Line:      3,
				Timestamp: timestamp.Add(5 * time.Minute),
				Text:      "Here's an answer",
				Type:      "",
				Resolved:  false,
			},
		},
		Positions: map[string]Position{
			"c1": {Line: 3, Column: 0, ByteOffset: 25},
			"c2": {Line: 3, Column: 0, ByteOffset: 25},
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

	// Verify comments count
	if len(loaded.Comments) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(loaded.Comments))
	}

	// Verify first comment
	c1 := loaded.Comments[0]
	if c1.ID != "c1" || c1.Author != "alice" || c1.Text != "This is a question" {
		t.Errorf("First comment mismatch: %+v", c1)
	}

	// Verify second comment
	c2 := loaded.Comments[1]
	if c2.ID != "c2" || c2.ParentID != "c1" || c2.ThreadID != "c1" {
		t.Errorf("Second comment mismatch: %+v", c2)
	}

	// Verify positions
	if len(loaded.Positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(loaded.Positions))
	}
}

func TestSaveAndLoadWithSuggestion(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test.md")

	content := "# Test\n\nOriginal line here\n"
	timestamp := time.Now()

	doc := &DocumentWithComments{
		Content: content,
		Comments: []*Comment{
			{
				ID:             "s1",
				ThreadID:       "s1",
				ParentID:       "",
				Author:         "claude",
				Line:           3,
				Timestamp:      timestamp,
				Text:           "Suggest changing this line",
				Type:           "S",
				Resolved:       false,
				SuggestionType: SuggestionLine,
				Selection: &Selection{
					StartLine: 3,
					EndLine:   3,
					Original:  "Original line here",
				},
				ProposedText:    "Improved line here",
				AcceptanceState: AcceptancePending,
			},
		},
		Positions: map[string]Position{
			"s1": {Line: 3, Column: 0, ByteOffset: 9},
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
	if len(loaded.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(loaded.Comments))
	}

	s1 := loaded.Comments[0]
	if s1.SuggestionType != SuggestionLine {
		t.Errorf("Expected SuggestionType 'line', got %q", s1.SuggestionType)
	}
	if s1.ProposedText != "Improved line here" {
		t.Errorf("ProposedText mismatch: got %q", s1.ProposedText)
	}
	if s1.AcceptanceState != AcceptancePending {
		t.Errorf("Expected AcceptanceState 'pending', got %q", s1.AcceptanceState)
	}
	if s1.Selection == nil {
		t.Fatal("Selection is nil")
	}
	if s1.Selection.Original != "Original line here" {
		t.Errorf("Selection.Original mismatch: got %q", s1.Selection.Original)
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

	// Load should succeed with empty comments
	doc, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatalf("LoadFromSidecar failed: %v", err)
	}

	if doc.Content != content {
		t.Errorf("Content mismatch")
	}
	if len(doc.Comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(doc.Comments))
	}
	if len(doc.Positions) != 0 {
		t.Errorf("Expected 0 positions, got %d", len(doc.Positions))
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
		Content:   "test",
		Comments:  []*Comment{},
		Positions: map[string]Position{},
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
		Content:   "test",
		Comments:  []*Comment{},
		Positions: map[string]Position{},
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
			Content:   "content",
			Comments:  []*Comment{},
			Positions: map[string]Position{},
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
