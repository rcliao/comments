package comment

// Anchor-by-text comment creation (docs/plan-agent-surface.md Phase 1):
// agents locate their target by quoting the line rather than shipping a
// grepped — and often stale — line number. Resolution reuses the same
// exact-then-normalized matching the re-anchor cascade trusts (anchor.go),
// but NOT its auto-pick: at creation time an ambiguous anchor is an error
// listing the candidates, never a guess.

import (
	"fmt"
	"sort"
	"strings"
)

// AmbiguousAnchorError reports an anchor that matches more than one line.
// Candidates are 1-based line numbers, ascending.
type AmbiguousAnchorError struct {
	Anchor     string
	Candidates []int
}

func (e *AmbiguousAnchorError) Error() string {
	nums := make([]string, len(e.Candidates))
	for i, n := range e.Candidates {
		nums[i] = fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("anchor matches %d lines (%s) — quote more of the line or pass line explicitly",
		len(e.Candidates), strings.Join(nums, ", "))
}

// ResolveAnchorText maps anchor text to its 1-based line number in content.
// The anchor may be a whole line or a unique substring of one. Exact
// (substring) matching first; whitespace/case-normalized matching as the
// fallback. Zero matches or more than one match is an error — creation never
// guesses (that is the re-anchor cascade's job, after the fact).
func ResolveAnchorText(content, anchor string) (int, error) {
	needle := strings.TrimSpace(anchor)
	if needle == "" {
		return 0, fmt.Errorf("anchor text is empty")
	}
	lines := strings.Split(content, "\n")

	match := func(pred func(line string) bool) []int {
		var out []int
		for i, l := range lines {
			if pred(l) {
				out = append(out, i+1)
			}
		}
		return out
	}

	candidates := match(func(l string) bool { return strings.Contains(l, needle) })
	if len(candidates) == 0 {
		norm := normalize(needle)
		candidates = match(func(l string) bool { return strings.Contains(normalize(l), norm) })
	}

	switch len(candidates) {
	case 0:
		return 0, fmt.Errorf("anchor %q not found in document", truncateAnchor(needle))
	case 1:
		return candidates[0], nil
	default:
		sort.Ints(candidates)
		return 0, &AmbiguousAnchorError{Anchor: needle, Candidates: candidates}
	}
}

func truncateAnchor(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}
