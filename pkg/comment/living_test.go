package comment

import (
	"strings"
	"testing"
)

const livingFixture = `---
comments:
    phase: shaping
    template: living
type: Living
---

# Work

## Now

- State: shaping.

## Why

Because.

## Findings

- F1. The rail leads with the verdict. pkg/tui/rail.go:61
- F2. The citation is on the next line,
  pkg/tui/rail.go:89
- F3. A thread backs this one. thread:c1abc

## Plan

- Slice 1.

## Decisions

- None yet.

## Checks

- ` + "`go test ./...`" + ` passes.
`

func livingViolations(t *testing.T, content string) []Violation {
	t.Helper()
	tmpl, err := loadBuiltinTemplate(LivingTemplate)
	if err != nil {
		t.Fatal(err)
	}
	return ValidateTemplate(content, tmpl)
}

// Every finding carries its own citation; one without fails, named by line.
func TestLivingFindingsNeedOwnCitation(t *testing.T) {
	for _, v := range livingViolations(t, livingFixture) {
		if v.Rule == "uncited_item" {
			t.Fatalf("cited findings flagged: %v", v)
		}
	}
	bad := strings.Replace(livingFixture, "- F3. A thread backs this one. thread:c1abc", "- F3. Trust me on this one.", 1)
	var got []Violation
	for _, v := range livingViolations(t, bad) {
		if v.Rule == "uncited_item" {
			got = append(got, v)
		}
	}
	if len(got) != 1 || got[0].Line != 23 {
		t.Fatalf("uncited violations = %v, want one at line 23", got)
	}
	if !IsLivingDoc(livingFixture) {
		t.Fatal("fixture not recognised as a living doc")
	}
}

// The phase is the tool's own field, and only three values are allowed.
func TestLivingPhaseParsedAndChecked(t *testing.T) {
	meta, err := ParseDocumentMetadata(livingFixture)
	if err != nil || meta.Phase != "shaping" {
		t.Fatalf("phase = %q, err %v", meta.Phase, err)
	}
	bad := strings.Replace(livingFixture, "phase: shaping", "phase: shipped", 1)
	found := false
	for _, v := range ValidateOKFMetadata(bad) {
		found = found || v.Rule == "invalid_phase"
	}
	if !found {
		t.Fatal("an unknown phase was accepted")
	}
}

// A glance reads the phase and Now's first line, markup stripped; the inbox
// carries the same, so the mod needs no parser of its own.
func TestReadLivingState(t *testing.T) {
	doc := strings.Replace(livingFixture, "- State: shaping.", "- **State:** shaping the `rail` slice.", 1)
	st, ok := ReadLivingState(doc)
	if !ok || st.Phase != "shaping" || st.Now != "shaping the rail slice." {
		t.Fatalf("state = %+v, ok %v", st, ok)
	}
	if _, ok := ReadLivingState("# plain\n"); ok {
		t.Fatal("a plain doc read as living")
	}
	path := writeSurfaceFixture(t, doc)
	inbox, err := BuildInbox(path, InboxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := inbox.Files[0].Living; got == nil || got.Now != st.Now {
		t.Fatalf("inbox living = %+v", got)
	}
}

func uncitedLines(t *testing.T, findings string) []int {
	t.Helper()
	i := strings.Index(livingFixture, "## Findings\n\n") + len("## Findings\n\n")
	j := strings.Index(livingFixture, "\n## Plan")
	doc := livingFixture[:i] + findings + livingFixture[j:]
	var lines []int
	for _, v := range livingViolations(t, doc) {
		if v.Rule == "uncited_item" {
			lines = append(lines, v.Line)
		}
	}
	return lines
}

// Every list style counts as a finding, fenced examples never do, and
// sub-bullets ride on their parent's citation.
func TestLivingFindingsEveryListStyle(t *testing.T) {
	cases := map[string]int{
		"1. F1. none\n2. F2. none\n":                    2,
		"+ F1. none\n":                                  1,
		"  - F1. none\n  - F2. ok pkg/tui/rail.go:61\n": 1,
		"* F1. none\n":                                  1,
		"Intro.\n\n```\n- not a finding\n```\n\n- F1. ok pkg/tui/rail.go:61\n": 0,
		"- F1. ok pkg/tui/rail.go:61\n  - detail without a cite\n":             0,
	}
	for findings, want := range cases {
		if got := uncitedLines(t, findings); len(got) != want {
			t.Errorf("%q: %d uncited (%v), want %d", findings, len(got), got, want)
		}
	}
}

// Now reads its first item whatever the marker, keeps literal asterisks, and
// a nested "### Now" never wins over the doc's own.
func TestReadLivingStateEdges(t *testing.T) {
	doc := strings.Replace(livingFixture, "- State: shaping.", "* **State:** 2 * 3 is `six`.", 1)
	doc = strings.Replace(doc, "## Plan\n\n- Slice 1.", "## Plan\n\n### Now\n\n- the wrong one", 1)
	st, _ := ReadLivingState(doc)
	if st.Now != "2 * 3 is six." {
		t.Fatalf("now = %q", st.Now)
	}
}
