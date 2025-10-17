package comment

import (
	"strings"
	"testing"
	"time"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantComments  int
		wantContent   string
		wantError     bool
	}{
		{
			name:         "no comments",
			input:        "# Hello World\n\nThis is a test.",
			wantComments: 0,
			wantContent:  "# Hello World\n\nThis is a test.",
		},
		{
			name:         "single comment with metadata",
			input:        "This is a test. {>>[@user:c1:1:2025-01-15T10:30:00Z] Great point! <<}",
			wantComments: 1,
			wantContent:  "This is a test.",
		},
		{
			name:         "single comment without metadata",
			input:        "This is a test. {>> This needs work <<}",
			wantComments: 1,
			wantContent:  "This is a test.",
		},
		{
			name: "multiple comments on different lines",
			input: `# Title {>>[@alice:c1:1:2025-01-15T10:00:00Z] Fix this <<}

Content here {>>[@bob:c2:3:2025-01-15T11:00:00Z] Add more detail <<}`,
			wantComments: 2,
			wantContent: `# Title

Content here`,
		},
		{
			name:         "multiple comments on same line",
			input:        "Text {>>[@user:c1:1:2025-01-15T10:00:00Z] First <<} more {>>[@user:c2:1:2025-01-15T10:01:00Z] Second <<}",
			wantComments: 2,
			wantContent:  "Text more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			doc, err := parser.Parse(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("Parse() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil {
				return
			}

			if len(doc.Comments) != tt.wantComments {
				t.Errorf("Parse() got %d comments, want %d", len(doc.Comments), tt.wantComments)
			}

			if strings.TrimSpace(doc.Content) != strings.TrimSpace(tt.wantContent) {
				t.Errorf("Parse() content = %q, want %q", doc.Content, tt.wantContent)
			}

			// Verify positions are tracked
			for _, comment := range doc.Comments {
				if _, ok := doc.Positions[comment.ID]; !ok {
					t.Errorf("Position not tracked for comment %s", comment.ID)
				}
			}
		})
	}
}

func TestParser_ParseCommentMetadata(t *testing.T) {
	parser := NewParser()

	input := "Test {>>[@alice:c123:5:2025-01-15T10:30:00Z] This is a comment <<}"
	doc, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(doc.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(doc.Comments))
	}

	c := doc.Comments[0]

	if c.Author != "alice" {
		t.Errorf("Author = %s, want alice", c.Author)
	}

	if c.ID != "c123" {
		t.Errorf("ID = %s, want c123", c.ID)
	}

	if c.Text != "This is a comment" {
		t.Errorf("Text = %s, want 'This is a comment'", c.Text)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	if !c.Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp = %v, want %v", c.Timestamp, expectedTime)
	}
}

func TestParser_RoundTrip(t *testing.T) {
	// Test that parse -> serialize -> parse produces the same result
	original := `# Document

This is content. {>>[@user:c1:3:2025-01-15T10:00:00Z] Comment one <<}

More content {>>[@alice:c2:5:2025-01-15T11:00:00Z] Comment two <<} here.`

	parser := NewParser()
	serializer := NewSerializer()

	// First parse
	doc1, err := parser.Parse(original)
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Serialize
	serialized, err := serializer.Serialize(doc1.Content, doc1.Comments, doc1.Positions)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Parse again
	doc2, err := parser.Parse(serialized)
	if err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}

	// Verify same number of comments
	if len(doc1.Comments) != len(doc2.Comments) {
		t.Errorf("Comment count mismatch: %d vs %d", len(doc1.Comments), len(doc2.Comments))
	}

	// Verify content is preserved
	if strings.TrimSpace(doc1.Content) != strings.TrimSpace(doc2.Content) {
		t.Errorf("Content mismatch:\n%s\nvs\n%s", doc1.Content, doc2.Content)
	}

	// Verify comment details
	for i := range doc1.Comments {
		c1 := doc1.Comments[i]
		c2 := doc2.Comments[i]

		if c1.ID != c2.ID {
			t.Errorf("Comment %d: ID mismatch: %s vs %s", i, c1.ID, c2.ID)
		}

		if c1.Author != c2.Author {
			t.Errorf("Comment %d: Author mismatch: %s vs %s", i, c1.Author, c2.Author)
		}

		if c1.Text != c2.Text {
			t.Errorf("Comment %d: Text mismatch: %s vs %s", i, c1.Text, c2.Text)
		}
	}
}
