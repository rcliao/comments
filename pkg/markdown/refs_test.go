package markdown

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseReferenceRange(t *testing.T) {
	content := "Research spans notes.md:12-27 here.\n"
	refs := ParseReferences(content)
	if len(refs) != 1 {
		t.Fatalf("want one range reference, got %+v", refs)
	}
	if refs[0].Line != 12 || refs[0].EndLine != 27 || refs[0].Raw != "notes.md:12-27" {
		t.Fatalf("range bounds were not preserved: %+v", refs[0])
	}
	if got := content[refs[0].StartCol:refs[0].EndCol]; got != refs[0].Raw {
		t.Fatalf("range span = %q, want %q", got, refs[0].Raw)
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

// Thread citations (plan-compounding-rpi Phase 1): explicit thread: scheme,
// same-doc and cross-doc forms; fences follow the comment-trail rule.
func TestParseThreadReferences(t *testing.T) {
	content := "Vetoed in thread:cz1xk after review.\n" +
		"Inherited from `thread:research.md#c6mv7` (the join decision).\n" +
		"```go\nx := \"thread:cfake\" // but thread:c9real in a trail\n```\n"
	refs := ParseReferences(content)
	if len(refs) != 3 {
		t.Fatalf("want 3 thread refs, got %d: %+v", len(refs), refs)
	}
	if refs[0].ThreadID != "cz1xk" || refs[0].Path != "" || refs[0].LineNum != 1 {
		t.Errorf("same-doc form: %+v", refs[0])
	}
	if refs[1].ThreadID != "c6mv7" || refs[1].Path != "research.md" {
		t.Errorf("cross-doc form: %+v", refs[1])
	}
	if refs[2].ThreadID != "c9real" || refs[2].LineNum != 4 {
		t.Errorf("fence comment-trail form: %+v", refs[2])
	}
}

// StripCitations exempts citation tokens from word counting: the token, its
// wrapping backticks or parens, and any range suffix all go, while markdown
// link text (prose the author wrote) and byte length stay put.
func TestStripCitations(t *testing.T) {
	cases := []struct {
		name, in  string
		wantWords int
	}{
		{"bare", "See gate.go:59 for this.", 3},
		{"backticked", "See `gate.go:59` now.", 2},
		{"parenthesized", "It fails (gate.go:59) here.", 3},
		{"range", "Rule at gate.go:11-44 applies.", 3},
		{"thread", "Vetoed in thread:cz1xk today.", 3},
		{"cross-doc thread", "Decided in `thread:research.md#c6mv7` earlier.", 3},
		{"link text kept", "Read [the plan](docs/plan.md) first.", 4},
		{"no citations", "Plain prose only.", 3},
	}
	for _, c := range cases {
		got := StripCitations(c.in)
		if n := len(strings.Fields(got)); n != c.wantWords {
			t.Errorf("%s: %q -> %q counts %d words, want %d", c.name, c.in, got, n, c.wantWords)
		}
		if len(got) != len(c.in) {
			t.Errorf("%s: length changed %d -> %d (offsets must survive)", c.name, len(c.in), len(got))
		}
	}
}

func TestStripCitationsMultiplePerLine(t *testing.T) {
	in := "Both gate.go:59 and `cmd/gate.go:114` decide it."
	got := StripCitations(in)
	if strings.Contains(got, "gate.go") {
		t.Errorf("both citations should be blanked, got %q", got)
	}
	if len(strings.Fields(got)) != 4 { // Both, and, decide, it.
		t.Errorf("want 4 remaining words, got %d in %q", len(strings.Fields(got)), got)
	}
}
