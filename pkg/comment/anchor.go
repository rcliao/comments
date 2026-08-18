package comment

import (
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
)

// Anchor confidence levels, best to worst
const (
	ConfidenceExact        = "exact"
	ConfidenceFuzzy        = "fuzzy"
	ConfidenceSectionLevel = "section-level"
)

// Anchor captures the content a comment points at, so it can be re-found
// after the document changes (v2.1: content anchoring, per design doc)
type Anchor struct {
	SelectedText  string `json:"selected_text"`
	ContextBefore string `json:"context_before,omitempty"`
	ContextAfter  string `json:"context_after,omitempty"`
}

// CaptureAnchor records the target line and one line of context either side.
// Returns nil when the line is out of bounds.
func CaptureAnchor(content string, line int) *Anchor {
	lines := strings.Split(content, "\n")
	if line < 1 || line > len(lines) {
		return nil
	}
	a := &Anchor{SelectedText: lines[line-1]}
	if line >= 2 {
		a.ContextBefore = lines[line-2]
	}
	if line < len(lines) {
		a.ContextAfter = lines[line]
	}
	return a
}

// normalize collapses whitespace and case for fuzzy comparison
func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// reanchorResult is the outcome of the cascade for one comment
type reanchorResult struct {
	line       int
	confidence string
	found      bool
}

// RelocateLine maps a 1-based line in oldContent to its best position in
// newContent, using the same cascade re-anchoring trusts (exact position ->
// exact search -> normalized search, context + proximity tie-breaks). Falls
// back to the original line clamped into range. Used when the TUI refreshes a
// changed document under a live cursor: the cursor follows its TEXT, so a
// comment lands on the line the human was looking at, not a stale number.
func RelocateLine(oldContent, newContent string, line int) int {
	newLines := strings.Split(newContent, "\n")
	clamp := func(n int) int { return max(1, min(n, len(newLines))) }
	a := CaptureAnchor(oldContent, line)
	if a == nil || strings.TrimSpace(a.SelectedText) == "" {
		return clamp(line)
	}
	if r := findAnchor(newLines, a, line); r.found {
		return r.line
	}
	return clamp(line)
}

// findAnchor runs cascade steps 1-3: exact position, exact search, normalized search.
// Among multiple text matches, context agreement wins, then proximity to the old line.
func findAnchor(lines []string, a *Anchor, oldLine int) reanchorResult {
	// Step 1: unchanged at stored position
	if oldLine >= 1 && oldLine <= len(lines) && lines[oldLine-1] == a.SelectedText {
		return reanchorResult{line: oldLine, confidence: ConfidenceExact, found: true}
	}

	match := func(cmp func(string, string) bool) []int {
		candidates := []int{}
		for i, l := range lines {
			if cmp(l, a.SelectedText) {
				candidates = append(candidates, i+1)
			}
		}
		return candidates
	}

	exactEq := func(x, y string) bool { return x == y }
	normEq := func(x, y string) bool { return normalize(x) == normalize(y) }

	// Blank/trivial lines match everywhere — don't chase them beyond step 1
	if strings.TrimSpace(a.SelectedText) == "" {
		return reanchorResult{}
	}

	// Step 2: exact search; Step 3: normalized search
	for _, attempt := range []struct {
		cmp        func(string, string) bool
		confidence string
	}{{exactEq, ConfidenceExact}, {normEq, ConfidenceFuzzy}} {
		candidates := match(attempt.cmp)
		if len(candidates) == 0 {
			continue
		}
		best := pickCandidate(lines, candidates, a, oldLine)
		return reanchorResult{line: best, confidence: attempt.confidence, found: true}
	}
	return reanchorResult{}
}

// pickCandidate disambiguates multiple matches: context agreement first, proximity second
func pickCandidate(lines []string, candidates []int, a *Anchor, oldLine int) int {
	if len(candidates) == 1 {
		return candidates[0]
	}
	bestScore, best := -1, candidates[0]
	for _, line := range candidates {
		score := 0
		if line >= 2 && normalize(lines[line-2]) == normalize(a.ContextBefore) {
			score += 2
		}
		if line < len(lines) && normalize(lines[line]) == normalize(a.ContextAfter) {
			score += 2
		}
		dist := max(line-oldLine, oldLine-line)
		// proximity as tiebreak: closer is better, bounded contribution
		proximity := max(1000-dist, 0)
		total := score*10000 + proximity
		if total > bestScore {
			bestScore, best = total, line
		}
	}
	return best
}

// ReanchorComment runs the full cascade for one comment against new content.
// Steps: exact position -> exact search -> normalized search -> section fallback -> orphan.
// Returns (moved, orphanReason). confidence is written to the comment.
func ReanchorComment(c *Comment, lines []string, structure *markdown.DocumentStructure) (bool, string) {
	oldLine := c.Line

	if c.Anchor != nil {
		if r := findAnchor(lines, c.Anchor, oldLine); r.found {
			c.AnchorConfidence = r.confidence
			if r.line != oldLine {
				c.OriginalLine = oldLine
				c.Line = r.line
				return true, ""
			}
			return false, ""
		}
	}

	// Step 4: section-path fallback
	if c.SectionPath != "" {
		if section := structure.FindSection(c.SectionPath); section != nil {
			c.AnchorConfidence = ConfidenceSectionLevel
			if section.StartLine != oldLine {
				c.OriginalLine = oldLine
				c.Line = section.StartLine
				return true, ""
			}
			return false, ""
		}
		return false, "Section '" + c.SectionPath + "' no longer exists and anchor text not found"
	}

	// Step 5: orphan
	if c.Anchor != nil {
		return false, "Anchor text not found: \"" + truncate(c.Anchor.SelectedText, 60) + "\""
	}
	if c.Line > len(lines) {
		return false, "Line out of bounds and no anchor to re-locate"
	}
	return false, ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// AnchorHealth counts the unresolved threads whose anchors re-located below
// exact confidence. It is a property of the DOCUMENT, not of any one thread:
// a fuzzy anchor says the document moved under the comment, which is the same
// news however many threads report it.
//
// The TUI used to spend row columns on this — " ~fuzzy" and " §section" rode
// every sidebar row, up to 9 of the 48 columns a row has, competing with the
// comment text for space it could not spare (see pkg/tui/CLAUDE.md, the row
// contract). Reported once in the review rail, it costs nothing per thread.
type AnchorHealth struct {
	Fuzzy        int // re-found by normalized text search
	SectionLevel int // only the section could be re-found, not the line
}

// Total is the number of threads whose anchor needs a human's eye.
func (h AnchorHealth) Total() int { return h.Fuzzy + h.SectionLevel }

// DocumentAnchorHealth counts degraded anchors across a document's unresolved
// root threads. Resolved threads are excluded: their anchor no longer decides
// anything, so re-checking it is work nobody needs done.
func DocumentAnchorHealth(doc *DocumentWithComments) AnchorHealth {
	var h AnchorHealth
	if doc == nil {
		return h
	}
	for _, t := range doc.Threads {
		if t == nil || t.Resolved {
			continue
		}
		switch t.AnchorConfidence {
		case ConfidenceFuzzy:
			h.Fuzzy++
		case ConfidenceSectionLevel:
			h.SectionLevel++
		}
	}
	return h
}
