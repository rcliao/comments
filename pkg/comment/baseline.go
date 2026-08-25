package comment

// Review baselines: the document content a reviewer last gave a verdict on.
//
// Thread-level round memory (NEW badges) already tells a reviewer which
// CONVERSATIONS moved since their pass; nothing told them which DOCUMENT
// LINES moved. Both signoff writers (TUI verdict, `comments signoff`) store
// the reviewed content here, and readers diff the current document against
// it to mark changed lines and name the sections they fall in.
//
// Exactly one baseline per document per reviewer — the latest verdict — under
// the document's `.comments/baselines/` directory (the same gitignored
// sidecar-state directory the TUI uses for view state). Decisions `approved`
// and `changes_requested` replace it; a `commented` reply-pass does not, so
// the marks keep reading "since your last VERDICT" until you judge the doc.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
)

// baselineMaxLines caps the LCS diff: beyond this the quadratic table is not
// worth a redraw, and a document that size has outgrown line marks anyway.
const baselineMaxLines = 20000

// BaselineUpdatesOn reports whether a signoff decision replaces the reviewer's
// baseline. Verdicts do; a reply-pass (`commented`) leaves the marks accumulating.
func BaselineUpdatesOn(decision string) bool {
	return decision == DecisionApproved || decision == DecisionChangesRequested
}

// BaselinePath returns where a reviewer's baseline for a document lives:
// <docdir>/.comments/baselines/<docbase>.<author>.md. The author component is
// sanitized because MCP authors are arbitrary strings.
func BaselinePath(docPath, author string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		}
		return '_'
	}, author)
	if safe == "" {
		safe = "_"
	}
	return filepath.Join(filepath.Dir(docPath), ".comments", "baselines",
		filepath.Base(docPath)+"."+safe+".md")
}

// SaveReviewBaseline records content as the reviewer's last-verdict baseline
// for docPath. Callers decide WHEN (see BaselineUpdatesOn); this only writes.
func SaveReviewBaseline(docPath, author, content string) error {
	path := BaselinePath(docPath, author)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating baseline dir: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// LoadReviewBaseline returns the reviewer's stored baseline for docPath, and
// false when none has been recorded.
func LoadReviewBaseline(docPath, author string) (string, bool) {
	data, err := os.ReadFile(BaselinePath(docPath, author))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// ChangeSet is what moved in a document relative to a reviewer's baseline.
type ChangeSet struct {
	Lines           map[int]bool // current line numbers (1-based) that are new or changed
	DeletionAnchors map[int]bool // current lines AFTER WHICH baseline content was removed (line 1 for a removal at the very start)
	Sections        []string     // section paths containing a changed line or a deletion anchor, in document order
}

// Count returns the number of added or edited lines.
func (cs ChangeSet) Count() int { return len(cs.Lines) }

// Deletions returns the number of places where baseline content was removed
// without replacement (a removal beside an edit is counted once, as the edit).
func (cs ChangeSet) Deletions() int { return len(cs.DeletionAnchors) }

// Marked returns every current line that carries a mark: edited lines plus
// the anchor of each deletion, so a pure removal is still visible.
func (cs ChangeSet) Marked() map[int]bool {
	out := make(map[int]bool, len(cs.Lines)+len(cs.DeletionAnchors))
	for l := range cs.Lines {
		out[l] = true
	}
	for l := range cs.DeletionAnchors {
		out[l] = true
	}
	return out
}

// ChangedSince diffs the current content of docPath against author's baseline.
// ok is false when no baseline exists — nothing to mark, zero cost.
func ChangedSince(docPath, author, content string) (cs ChangeSet, ok bool) {
	base, ok := LoadReviewBaseline(docPath, author)
	if !ok {
		return ChangeSet{}, false
	}
	lines, deleted := DiffLines(base, content)
	cs = ChangeSet{Lines: lines, DeletionAnchors: deleted}
	cs.Sections = ChangedSections(content, cs.Marked())
	return cs, true
}

// ChangedLines returns the 1-based line numbers in current that were added
// or edited relative to baseline (see DiffLines).
func ChangedLines(baseline, current string) map[int]bool {
	changed, _ := DiffLines(baseline, current)
	return changed
}

// DiffLines aligns current against baseline with a longest common subsequence
// and returns (changed, deletionAnchors): changed holds the 1-based current
// lines that are new or edited; deletionAnchors holds, for every run of
// baseline lines that disappeared WITHOUT replacement, the current line that
// PRECEDES the gap (line 1 when the removal was at the very start). Anchoring
// on the preceding line keeps the mark inside the section that lost content —
// the following line is often the next heading — so a pure deletion still has
// a line to mark in the right place, while an edit counts once. Over
// baselineMaxLines the diff is skipped.
func DiffLines(baseline, current string) (changed, deletionAnchors map[int]bool) {
	a := strings.Split(baseline, "\n")
	b := strings.Split(current, "\n")
	changed, deletionAnchors = map[int]bool{}, map[int]bool{}
	if baseline == current {
		return changed, deletionAnchors
	}
	if len(a) > baselineMaxLines || len(b) > baselineMaxLines {
		return changed, deletionAnchors
	}

	// Trim the common prefix and suffix first: most edits are local, and the
	// LCS table only needs to cover the middle.
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	am, bm := a[pre:len(a)-suf], b[pre:len(b)-suf]

	// LCS lengths over the middle; then walk back marking b-lines that are
	// not matched to an a-line.
	n, m := len(am), len(bm)
	dp := make([][]int32, n+1)
	for i := range dp {
		dp[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if am[i] == bm[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	// The current line a gap at middle offset j is anchored to: the line
	// before it (clamped to 1 when the removal was at the very start).
	anchor := func(j int) int { return max(1, min(pre+j, len(b))) }
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case am[i] == bm[j]:
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			deletionAnchors[anchor(j)] = true // deleted from baseline
			i++
		default:
			changed[pre+j+1] = true // added/changed in current
			j++
		}
	}
	for ; j < m; j++ {
		changed[pre+j+1] = true
	}
	if i < n {
		deletionAnchors[anchor(j)] = true
	}
	// A gap that touches an added/edited line is a REPLACEMENT, already
	// visible through the changed line; only gaps with nothing new beside
	// them are pure deletions worth their own mark.
	for line := range deletionAnchors {
		if changed[line] || changed[line+1] {
			delete(deletionAnchors, line)
		}
	}
	return changed, deletionAnchors
}

// ChangedSections names the sections that contain at least one marked line,
// by hierarchical path, in document order. A changed line is attributed to
// the innermost section enclosing it; lines before the first heading belong
// to no section and are not reported.
func ChangedSections(content string, changed map[int]bool) []string {
	if len(changed) == 0 {
		return nil
	}
	structure := markdown.ParseDocument(content)
	seen := map[string]int{} // path -> first changed line (for ordering)
	for line := range changed {
		path := structure.GetSectionPath(line)
		if path == "" {
			continue
		}
		if first, ok := seen[path]; !ok || line < first {
			seen[path] = line
		}
	}
	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(x, y int) bool { return seen[paths[x]] < seen[paths[y]] })
	return paths
}
