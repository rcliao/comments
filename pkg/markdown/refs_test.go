package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func refByPath(refs []Reference, path string) *Reference {
	for i := range refs {
		if refs[i].Path == path {
			return &refs[i]
		}
	}
	return nil
}

func TestParseReferencesForms(t *testing.T) {
	content := `# Plan

Per research.md:42 the cascade works.
See [the findings](./research/notes.md#cascade) for detail.
Deep path pkg/tui/model.go:123 is cited too.
A ratio 3:5 and a time 12:30 are not references.
Nor is word:12 or https://example.com/a.md:1 a local citation.
[external](https://example.com/doc.md) links are ignored.
[anchor only](#local-section) is not a file reference.
`
	refs := ParseReferences(content)

	r := refByPath(refs, "research.md")
	if r == nil || r.Line != 42 || r.LineNum != 3 {
		t.Fatalf("research.md:42 should parse as unit on line 3, got %+v", refs)
	}
	if got := content[posOf(t, content, 3, r.StartCol):posOf(t, content, 3, r.EndCol)]; got != "research.md:42" {
		t.Errorf("span should cover the raw token, got %q", got)
	}

	link := refByPath(refs, "./research/notes.md")
	if link == nil || link.Heading != "cascade" || link.Line != 0 {
		t.Fatalf("md link with #heading not parsed: %+v", refs)
	}

	deep := refByPath(refs, "pkg/tui/model.go")
	if deep == nil || deep.Line != 123 {
		t.Fatalf("slash path file:line not parsed: %+v", refs)
	}

	for _, banned := range []string{"3", "12", "word", "https", "example.com", "https://example.com/doc.md"} {
		if refByPath(refs, banned) != nil {
			t.Errorf("false positive parsed: %q", banned)
		}
	}
	if len(refs) != 3 {
		t.Errorf("expected exactly 3 references, got %d: %+v", len(refs), refs)
	}
}

// posOf converts (line, col) to a content offset for span verification
func posOf(t *testing.T, content string, line, col int) int {
	t.Helper()
	offset := 0
	for l := 1; l < line; l++ {
		idx := indexByteFrom(content, offset, '\n')
		if idx < 0 {
			t.Fatal("line out of range")
		}
		offset = idx + 1
	}
	return offset + col
}

func indexByteFrom(s string, from int, b byte) int {
	for i := from; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestParseReferencesSkipsCodeFences(t *testing.T) {
	// Non-comment code in a fence is not a citation; a COMMENT trail in a
	// fence is (schema templates put their file:line evidence there).
	content := "# D\n\n```go\nload(\"parser.go:10\")\n// see parser.go:10\n```\nreal.md:5 after fence.\n"
	refs := ParseReferences(content)
	if len(refs) != 2 {
		t.Fatalf("want fence-comment + after-fence refs only, got %+v", refs)
	}
	if refs[0].Path != "parser.go" || refs[0].LineNum != 5 {
		t.Errorf("fence comment-trail citation should parse: %+v", refs[0])
	}
	if refs[1].Path != "real.md" {
		t.Errorf("real citation after fence kept: %+v", refs[1])
	}
}

func TestResolveReference(t *testing.T) {
	root := t.TempDir()
	docDir := filepath.Join(root, "docs", "plans")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sibling := filepath.Join(docDir, "research.md")
	if err := os.WriteFile(sibling, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	atRoot := filepath.Join(root, "shared.md")
	if err := os.WriteFile(atRoot, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, ok := ResolveReference(docDir, "research.md"); !ok || got != sibling {
		t.Errorf("doc-relative resolution failed: %q %v", got, ok)
	}
	// walk-up: shared.md is two levels above the doc
	if got, ok := ResolveReference(docDir, "shared.md"); !ok || got != atRoot {
		t.Errorf("walk-up resolution failed: %q %v", got, ok)
	}
	if _, ok := ResolveReference(docDir, "missing.md"); ok {
		t.Error("missing file must not resolve")
	}
	// directories are not references
	if _, ok := ResolveReference(docDir, "plans"); ok {
		t.Error("directory must not resolve as a file reference")
	}
}

func TestHeadingLine(t *testing.T) {
	content := "# Doc\n\n## Goals / Non-Goals\n\nText.\n\n## The Cascade\n\nMore.\n"
	if got := HeadingLine(content, "the-cascade"); got != 7 {
		t.Errorf("slug anchor should find heading line 7, got %d", got)
	}
	if got := HeadingLine(content, "The Cascade"); got != 7 {
		t.Errorf("raw title anchor should find heading line 7, got %d", got)
	}
	if got := HeadingLine(content, "goals--non-goals"); got != 3 {
		t.Errorf("punctuated slug should find line 3, got %d", got)
	}
	if got := HeadingLine(content, "nope"); got != 0 {
		t.Errorf("unknown anchor should return 0, got %d", got)
	}
}

// Schema-notation templates put their file:line evidence in comment trails
// INSIDE fenced blocks (DBML `// scorer.go:41`); those must peek, while
// non-comment code in fences stays excluded.
func TestParseReferencesInFenceCommentTrails(t *testing.T) {
	content := "```dbml\n" +
		"Table check_result {\n" +
		"  max_points int // pkg/comment/gate.go:41 hard-coded here\n" +
		"  raw string\n" +
		"}\n" +
		"```\n" +
		"```go\n" +
		"data := load(\"types.go:12\") // not a citation target: fake.go\n" +
		"```\n"
	refs := ParseReferences(content)
	if len(refs) != 1 {
		t.Fatalf("expected exactly the comment-trail citation, got %d: %+v", len(refs), refs)
	}
	if refs[0].Path != "pkg/comment/gate.go" || refs[0].Line != 41 || refs[0].LineNum != 3 {
		t.Errorf("got %+v, want gate.go:41 at doc line 3", refs[0])
	}
}
