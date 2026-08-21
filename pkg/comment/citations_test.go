package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// citationRepo builds a throwaway project with a duplicated basename, the
// shape that makes a bare reference unpeekable.
func citationRepo(t *testing.T) (root, docPath string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/x\n")
	write("pkg/lib/gate.go", strings.Repeat("line\n", 100))
	write("cmd/app/gate.go", strings.Repeat("line\n", 20))
	write("pkg/lib/only.go", strings.Repeat("line\n", 50))
	write("docs/r.md", "placeholder\n")
	return root, filepath.Join(root, "docs", "r.md")
}

func citationRules(v []Violation) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, x.Rule)
	}
	return out
}

func TestCitationPathQualifiedResolves(t *testing.T) {
	_, doc := citationRepo(t)
	if v := ValidateCitations("See pkg/lib/gate.go:42-50 for the cascade.\n", doc); len(v) != 0 {
		t.Errorf("a path-qualified, in-range citation must pass, got %v", citationRules(v))
	}
}

func TestCitationBareBasenameAmbiguous(t *testing.T) {
	_, doc := citationRepo(t)
	v := ValidateCitations("See gate.go:15 for the decision.\n", doc)
	if len(v) != 1 || v[0].Rule != "ambiguous_citation" {
		t.Fatalf("a basename matching two files must be ambiguous, got %v", citationRules(v))
	}
	// The message has to name the candidates, or the writer cannot fix it.
	for _, want := range []string{"pkg/lib/gate.go", "cmd/app/gate.go"} {
		if !strings.Contains(v[0].Message, want) {
			t.Errorf("message must list %s: %s", want, v[0].Message)
		}
	}
}

func TestCitationBareBasenameUniqueIsFine(t *testing.T) {
	_, doc := citationRepo(t)
	if v := ValidateCitations("See only.go:10 here.\n", doc); len(v) != 0 {
		t.Errorf("an unambiguous basename is peekable, got %v", citationRules(v))
	}
}

func TestCitationMissingFile(t *testing.T) {
	_, doc := citationRepo(t)
	v := ValidateCitations("See pkg/lib/ghost.go:3 here.\n", doc)
	if len(v) != 1 || v[0].Rule != "unresolvable_citation" {
		t.Fatalf("a citation to a nonexistent file must fail, got %v", citationRules(v))
	}
}

func TestCitationLinePastEOF(t *testing.T) {
	_, doc := citationRepo(t)
	v := ValidateCitations("See cmd/app/gate.go:19-25 here.\n", doc) // file has 20 lines
	if len(v) != 1 || v[0].Rule != "unresolvable_citation" {
		t.Fatalf("a range past EOF must fail, got %v", citationRules(v))
	}
	if !strings.Contains(v[0].Message, "20 lines") {
		t.Errorf("message should state the real length: %s", v[0].Message)
	}
}

func TestCitationReversedRange(t *testing.T) {
	_, doc := citationRepo(t)
	v := ValidateCitations("See pkg/lib/only.go:20-10 here.\n", doc)
	if len(v) != 1 || !strings.Contains(v[0].Message, "reversed") {
		t.Fatalf("reversed range must fail clearly, got %+v", v)
	}
}

// Fenced code is illustration, but comment trails are evidence: schema
// templates put their load-bearing citations beside fields inside fences.
func TestCitationFenceCommentTrailsOnly(t *testing.T) {
	_, doc := citationRepo(t)
	content := "Prose.\n\n```go\nload(\"nowhere/at/all.go:999\")\n// see pkg/lib/only.go:10\n```\n\nMore prose.\n"
	if v := ValidateCitations(content, doc); len(v) != 0 {
		t.Errorf("only the valid comment-trail citation should be checked, got %v", citationRules(v))
	}
}

func TestCitationBrokenFenceCommentTrailFails(t *testing.T) {
	_, doc := citationRepo(t)
	content := "```dbml\nfield int // nowhere/at/all.go:999\n```\n"
	v := ValidateCitations(content, doc)
	if len(v) != 1 || v[0].Rule != "unresolvable_citation" {
		t.Fatalf("broken evidence in a fence comment trail must fail, got %+v", v)
	}
}

func TestCitationOutsideRepoIsNoOp(t *testing.T) {
	// No .git or go.mod above it: nothing to resolve against, so stay silent
	// rather than reporting every reference as broken.
	dir := t.TempDir()
	doc := filepath.Join(dir, "r.md")
	if err := os.WriteFile(doc, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v := ValidateCitations("See pkg/lib/gate.go:1 here.\n", doc); v != nil {
		t.Errorf("outside a project there is nothing to check, got %v", citationRules(v))
	}
}

// Version strings and ordinary prose must not be mistaken for citations.
func TestCitationIgnoresNonCitations(t *testing.T) {
	_, doc := citationRepo(t)
	content := "Released v2.1:100 users. Ratio 16:9. See RFC 7231:2014.\n"
	if v := ValidateCitations(content, doc); len(v) != 0 {
		t.Errorf("prose must not be parsed as citations, got %v (%+v)", citationRules(v), v)
	}
}
