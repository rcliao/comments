package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const anchorDoc = `# Title

## Alpha

The quick brown fox.

## Beta

Lazy dogs sleep here.
`

func anchoredComment(t *testing.T, line int) *Comment {
	c := NewComment("eric", line, "note")
	UpdateCommentSection(c, anchorDoc)
	if c.Anchor == nil {
		t.Fatalf("anchor not captured for line %d", line)
	}
	return c
}

func runValidation(doc *DocumentWithComments) (int, []ValidationIssue) {
	doc.DocumentHash = "stale-hash-forces-mismatch"
	return ValidateAndUpdateCommentStatus(doc)
}

func TestAnchorKeepsExactPosition(t *testing.T) {
	c := anchoredComment(t, 5) // "The quick brown fox."
	doc := &DocumentWithComments{Content: anchorDoc, Threads: []*Comment{c}}
	runValidation(doc)
	if c.Line != 5 || c.AnchorConfidence != ConfidenceExact {
		t.Errorf("expected line 5 exact, got line %d (%s)", c.Line, c.AnchorConfidence)
	}
}

func TestAnchorFollowsMovedText(t *testing.T) {
	c := anchoredComment(t, 5)
	moved := strings.Replace(anchorDoc, "## Alpha\n\nThe quick brown fox.\n",
		"## Inserted\n\nNew stuff.\n\n## Alpha\n\nThe quick brown fox.\n", 1)
	doc := &DocumentWithComments{Content: moved, Threads: []*Comment{c}}
	runValidation(doc)

	lines := strings.Split(moved, "\n")
	if lines[c.Line-1] != "The quick brown fox." {
		t.Errorf("comment should follow its text, points at %q", lines[c.Line-1])
	}
	if c.AnchorConfidence != ConfidenceExact {
		t.Errorf("expected exact confidence, got %s", c.AnchorConfidence)
	}
	if c.Status == "orphaned" {
		t.Error("moved comment should not be orphaned")
	}
}

func TestAnchorFuzzyMatch(t *testing.T) {
	c := anchoredComment(t, 5)
	edited := strings.Replace(anchorDoc, "The quick brown fox.", "The  QUICK brown fox.", 1)
	doc := &DocumentWithComments{Content: edited, Threads: []*Comment{c}}
	runValidation(doc)
	if c.AnchorConfidence != ConfidenceFuzzy {
		t.Errorf("expected fuzzy confidence, got %s", c.AnchorConfidence)
	}
	if c.Status == "orphaned" {
		t.Error("fuzzy-matched comment should not be orphaned")
	}
}

func TestAnchorSectionFallback(t *testing.T) {
	c := anchoredComment(t, 5)
	edited := strings.Replace(anchorDoc, "The quick brown fox.", "Entirely different sentence now.", 1)
	doc := &DocumentWithComments{Content: edited, Threads: []*Comment{c}}
	runValidation(doc)
	if c.AnchorConfidence != ConfidenceSectionLevel {
		t.Errorf("expected section-level fallback, got %s", c.AnchorConfidence)
	}
	if c.Line != 3 { // "## Alpha" heading
		t.Errorf("expected fallback to heading line 3, got %d", c.Line)
	}
}

func TestAnchorOrphansWhenAllGone(t *testing.T) {
	c := anchoredComment(t, 5)
	edited := strings.Replace(anchorDoc, "## Alpha\n\nThe quick brown fox.\n", "", 1)
	doc := &DocumentWithComments{Content: edited, Threads: []*Comment{c}}
	orphaned, _ := runValidation(doc)
	if orphaned != 1 || c.Status != "orphaned" {
		t.Errorf("expected orphan, got status %q (count %d)", c.Status, orphaned)
	}
}

func TestNoSnapToHeadingOnUnrelatedEdit(t *testing.T) {
	// Regression: body-line comments must keep their line, not jump to the heading
	c := anchoredComment(t, 5)
	edited := strings.Replace(anchorDoc, "Lazy dogs", "Very lazy dogs", 1)
	doc := &DocumentWithComments{Content: edited, Threads: []*Comment{c}}
	runValidation(doc)
	if c.Line != 5 {
		t.Errorf("unrelated edit moved comment from line 5 to %d", c.Line)
	}
}

func TestShortIDFormat(t *testing.T) {
	c := NewComment("a", 1, "x")
	if len(c.ID) != 5 || c.ID[0] != 'c' {
		t.Errorf("expected short ID like c7f3k, got %q", c.ID)
	}
}

func TestEnsureUniqueIDs(t *testing.T) {
	doc := &DocumentWithComments{
		Content: "# T\n",
		Threads: []*Comment{
			{ID: "cdup", Text: "a"},
			{ID: "cdup", Text: "b"},
			{ID: "cdup", Text: "c"},
		},
	}
	EnsureUniqueIDs(doc)
	seen := map[string]bool{}
	for _, c := range doc.Threads {
		if seen[c.ID] {
			t.Errorf("duplicate ID survived: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestAnchorBackfillOnLoad(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "d.md")
	if err := os.WriteFile(mdPath, []byte(anchorDoc), 0644); err != nil {
		t.Fatal(err)
	}
	// Simulate an old sidecar: comment without anchor, hash current
	doc := &DocumentWithComments{
		Content: anchorDoc,
		Threads: []*Comment{{ID: "cold1", Author: "eric", Line: 5, Text: "legacy"}},
	}
	if err := SaveToSidecar(mdPath, doc); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := LoadFromSidecar(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	c := loaded.Threads[0]
	if c.Anchor == nil || c.Anchor.SelectedText != "The quick brown fox." {
		t.Errorf("anchor not backfilled on load: %+v", c.Anchor)
	}
}

func TestPickCandidateContextBeatsProximity(t *testing.T) {
	lines := []string{
		"alpha",    // 1
		"dup line", // 2 (closest to oldLine)
		"beta",     // 3
		"gamma",    // 4
		"dup line", // 5 (context matches here)
		"delta",    // 6
	}
	a := &Anchor{SelectedText: "dup line", ContextBefore: "gamma", ContextAfter: "delta"}

	got := pickCandidate(lines, []int{2, 5}, a, 2)
	if got != 5 {
		t.Errorf("pickCandidate = %d, want 5 (context agreement outranks proximity)", got)
	}
}

func TestPickCandidateProximityTiebreak(t *testing.T) {
	lines := []string{
		"x",        // 1
		"dup line", // 2
		"x",        // 3
		"x",        // 4
		"dup line", // 5
		"x",        // 6
	}
	// Context matches both candidates equally, so the nearest to oldLine wins
	a := &Anchor{SelectedText: "dup line", ContextBefore: "x", ContextAfter: "x"}

	if got := pickCandidate(lines, []int{2, 5}, a, 4); got != 5 {
		t.Errorf("pickCandidate(oldLine=4) = %d, want 5 (closest)", got)
	}
	if got := pickCandidate(lines, []int{2, 5}, a, 2); got != 2 {
		t.Errorf("pickCandidate(oldLine=2) = %d, want 2 (closest)", got)
	}
}

func TestPickCandidateSingleMatch(t *testing.T) {
	lines := []string{"a", "b"}
	a := &Anchor{SelectedText: "b"}
	if got := pickCandidate(lines, []int{2}, a, 99); got != 2 {
		t.Errorf("pickCandidate single = %d, want 2", got)
	}
}
