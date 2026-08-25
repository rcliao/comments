package comment

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func TestBaselineUpdatesOnVerdictsOnly(t *testing.T) {
	if !BaselineUpdatesOn(DecisionApproved) || !BaselineUpdatesOn(DecisionChangesRequested) {
		t.Fatal("verdicts must update the baseline")
	}
	// A reply-pass hands the turn back without judging the doc: the marks keep
	// reading "since your last VERDICT" (plan-landscape-improvements Phase 2)
	if BaselineUpdatesOn(DecisionCommented) || BaselineUpdatesOn("") {
		t.Fatal("commented / empty decisions must leave the baseline untouched")
	}
}

func TestBaselinePathIsDirLocalAndSanitized(t *testing.T) {
	got := BaselinePath("/proj/docs/plan.md", "claude/agent 1")
	want := filepath.Join("/proj/docs", ".comments", "baselines", "plan.md.claude_agent_1.md")
	if got != want {
		t.Errorf("BaselinePath = %q, want %q", got, want)
	}
	if !strings.HasSuffix(BaselinePath("/p/d.md", ""), "d.md._.md") {
		t.Error("empty author must still yield a usable filename")
	}
}

func TestSaveAndLoadReviewBaseline(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	if _, ok := LoadReviewBaseline(doc, "eric"); ok {
		t.Fatal("no baseline should exist before save")
	}
	if err := SaveReviewBaseline(doc, "eric", "a\nb\n"); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadReviewBaseline(doc, "eric")
	if !ok || got != "a\nb\n" {
		t.Fatalf("LoadReviewBaseline = %q, %v", got, ok)
	}
	// Latest only: a second save replaces, never appends
	if err := SaveReviewBaseline(doc, "eric", "c\n"); err != nil {
		t.Fatal(err)
	}
	if got, _ := LoadReviewBaseline(doc, "eric"); got != "c\n" {
		t.Errorf("second save should replace, got %q", got)
	}
	// Per reviewer: another author's baseline is independent
	if _, ok := LoadReviewBaseline(doc, "claude"); ok {
		t.Error("baselines must be per reviewer")
	}
	if _, err := os.Stat(filepath.Join(dir, ".comments", "baselines")); err != nil {
		t.Errorf("baseline dir not created: %v", err)
	}
}

func TestChangedLinesMarksAddedAndEditedOnly(t *testing.T) {
	base := "# T\n\nline one\nline two\nline three\n"
	cur := "# T\n\nline one\nline two EDITED\ninserted\nline three\n"
	got := sortedKeys(ChangedLines(base, cur))
	if want := []int{4, 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("ChangedLines = %v, want %v", got, want)
	}
}

func TestDiffLinesIdenticalAndDeletionOnly(t *testing.T) {
	c, d := DiffLines("a\nb\n", "a\nb\n")
	if len(c)+len(d) != 0 {
		t.Errorf("identical content marked %v / %v", c, d)
	}
	// A pure deletion has no edited line, but the line BEFORE the gap carries
	// the mark so the removal is still visible
	c, d = DiffLines("a\nb\nc\n", "a\nc\n")
	if len(c) != 0 || !reflect.DeepEqual(sortedKeys(d), []int{1}) {
		t.Errorf("deletion-only: changed=%v anchors=%v, want none / [1]", c, d)
	}
	// Removed at the end: the last current line marks it
	c, d = DiffLines("a\nb\nc", "a")
	if len(c) != 0 || !reflect.DeepEqual(sortedKeys(d), []int{1}) {
		t.Errorf("tail deletion: changed=%v anchors=%v, want none / [1]", c, d)
	}
	// Removed at the very start: clamps to line 1
	_, d = DiffLines("X\na\n", "a\n")
	if !reflect.DeepEqual(sortedKeys(d), []int{1}) {
		t.Errorf("head deletion: anchors=%v, want [1]", d)
	}
	// Two separate gaps → two marks; one contiguous run → one mark
	_, d = DiffLines("a\nX\nb\nY\nZ\nc\n", "a\nb\nc\n")
	if !reflect.DeepEqual(sortedKeys(d), []int{1, 2}) {
		t.Errorf("two gaps: anchors=%v, want [1 2]", d)
	}
	// A replacement (old line gone, new line in its place) is ONE change,
	// not a change plus a deletion
	c, d = DiffLines("a\nold\nc\n", "a\nnew one\nnew two\nc\n")
	if !reflect.DeepEqual(sortedKeys(c), []int{2, 3}) || len(d) != 0 {
		t.Errorf("replacement: changed=%v deletedBefore=%v, want [2 3] / none", c, d)
	}
	cs := ChangeSet{Lines: map[int]bool{4: true}, DeletionAnchors: map[int]bool{2: true, 4: true}}
	if got := sortedKeys(cs.Marked()); !reflect.DeepEqual(got, []int{2, 4}) || cs.Deletions() != 2 {
		t.Errorf("Marked = %v, Deletions = %d", got, cs.Deletions())
	}
}

func TestChangedLinesMovedBlockMarksNewPositionOnly(t *testing.T) {
	base := "x\ny\nz\n"
	cur := "z\nx\ny\n"
	got := sortedKeys(ChangedLines(base, cur))
	// LCS keeps x,y aligned; z is unmatched at its new position
	if want := []int{1}; !reflect.DeepEqual(got, want) {
		t.Errorf("ChangedLines = %v, want %v", got, want)
	}
}

func TestChangedLinesSkipsOversizedDocuments(t *testing.T) {
	big := strings.Repeat("l\n", baselineMaxLines+1)
	if n := len(ChangedLines(big, big+"extra\n")); n != 0 {
		t.Errorf("oversized diff should be skipped, marked %d", n)
	}
}

func TestChangedSectionsInnermostInDocumentOrder(t *testing.T) {
	content := strings.Join([]string{
		"preamble",             // 1: no section
		"# Doc",                // 2
		"intro",                // 3
		"## Plan",              // 4
		"### Phase 2",          // 5
		"phase two body",       // 6
		"## Data model",        // 7
		"entity",               // 8
		"## Untouched",         // 9
		"nothing changed here", // 10
	}, "\n")
	changed := map[int]bool{1: true, 8: true, 6: true, 3: true}
	got := ChangedSections(content, changed)
	want := []string{"Doc", "Doc > Plan > Phase 2", "Doc > Data model"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ChangedSections = %v, want %v", got, want)
	}
	if ChangedSections(content, nil) != nil {
		t.Error("no changes should yield nil sections")
	}
}

func TestChangedSinceRequiresBaseline(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "d.md")
	if _, ok := ChangedSince(doc, "eric", "x\n"); ok {
		t.Fatal("no baseline → ok must be false")
	}
	if err := SaveReviewBaseline(doc, "eric", "# A\nold\n"); err != nil {
		t.Fatal(err)
	}
	cs, ok := ChangedSince(doc, "eric", "# A\nnew\n")
	if !ok || cs.Count() != 1 || !cs.Lines[2] || !reflect.DeepEqual(cs.Sections, []string{"A"}) {
		t.Errorf("ChangedSince = %+v, %v", cs, ok)
	}
	// A deletion inside a section names that section even with no edited line
	if err := SaveReviewBaseline(doc, "eric", "# A\nkeep\n# B\ngone\nkeep\n"); err != nil {
		t.Fatal(err)
	}
	cs, _ = ChangedSince(doc, "eric", "# A\nkeep\n# B\nkeep\n")
	if cs.Count() != 0 || cs.Deletions() != 1 || !reflect.DeepEqual(cs.Sections, []string{"B"}) {
		t.Errorf("deletion rollup = %+v", cs)
	}
	// Removing the TAIL of a section must blame that section, not the next
	// heading (which is what "line after the gap" would have named)
	if err := SaveReviewBaseline(doc, "eric", "# A\na1\na2\n# B\nb1\n"); err != nil {
		t.Fatal(err)
	}
	cs, _ = ChangedSince(doc, "eric", "# A\na1\n# B\nb1\n")
	if cs.Deletions() != 1 || !reflect.DeepEqual(cs.Sections, []string{"A"}) {
		t.Errorf("tail-of-section deletion rollup = %+v, want [A]", cs)
	}
}
