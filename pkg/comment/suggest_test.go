package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A suggestion is usually accepted AFTER the agent has kept editing the
// document. Re-anchoring moved the suggestion's Line but left its range
// behind, so an accept replaced whatever now sat on the old lines — and with
// no original text to verify against, it did so silently.
func TestSuggestionRangeTravelsWithItsAnchor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	content := "# Doc\n\n## Design\n\nKeep this line.\nEntries expire after sixty seconds.\nKeep this too.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, _, err := LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	// Deliberately no original text: nothing but the range protects the accept.
	result, err := AddSuggestion(doc, SuggestionSpec{
		Author: "claude", Text: "configurable", Anchor: "Entries expire after sixty seconds.",
		ProposedText: "Entries expire after a per-route TTL.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StartLine != 6 || result.SectionPath != "Doc > Design" {
		t.Fatalf("suggestion = %+v, want line 6 in Doc > Design", result)
	}
	if err := SaveToSidecar(path, doc); err != nil {
		t.Fatal(err)
	}

	// The agent inserts two lines above the suggested range.
	edited := strings.Replace(content, "## Design\n", "## Design\n\nInserted one.\nInserted two.\n", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, _, err = LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	s := doc.FindCommentByID(result.SuggestionID)
	if s.Line != 9 || s.StartLine != 9 || s.EndLine != 9 {
		t.Fatalf("after the edit: line %d, range %d-%d; want all 9", s.Line, s.StartLine, s.EndLine)
	}
	applied, err := ApplySuggestion(doc.Content, s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(applied, "Entries expire after a per-route TTL.") || !strings.Contains(applied, "Inserted two.") || strings.Contains(applied, "sixty seconds") {
		t.Errorf("accept replaced the wrong lines:\n%s", applied)
	}
}
