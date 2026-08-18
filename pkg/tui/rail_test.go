package tui

import (
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/rcliao/comments/pkg/comment"
)

func plainRail(m *Model, width int) string {
	return termEscapes.ReplaceAllString(m.renderRail(width), "")
}

// The rail and the verdict dialog must never disagree about whether the
// document passes: both derive from comment.EvaluateGate.
func TestRailVerdictMatchesGate(t *testing.T) {
	blocked := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "must fix", Author: "a", Blocking: true},
	})
	if got := plainRail(blocked, 120); !strings.Contains(got, "CHANGES REQUESTED") {
		t.Errorf("blocking thread should show CHANGES REQUESTED, got %q", got)
	}

	clean := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "nit", Author: "a"},
	})
	if got := plainRail(clean, 120); !strings.Contains(got, "APPROVED") {
		t.Errorf("no blocking threads should show APPROVED, got %q", got)
	}
}

func TestRailCounts(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "must fix", Author: "a", Blocking: true},
		{ID: "c2", Line: 6, Text: "nit", Author: "a"},
		{ID: "c3", Line: 7, Text: "done", Author: "a", Resolved: true},
	})
	got := plainRail(m, 140)
	for _, want := range []string{"1 blocking", "1 open", "1 resolved"} {
		if !strings.Contains(got, want) {
			t.Errorf("rail missing %q, got %q", want, got)
		}
	}
}

// Anchor health rides the rail instead of every sidebar row — but only when
// there is something to act on. A clean document must stay quiet.
func TestRailReportsAnchorHealthOnlyWhenDegraded(t *testing.T) {
	clean := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "nit", Author: "a", AnchorConfidence: comment.ConfidenceExact},
	})
	if got := plainRail(clean, 140); strings.Contains(got, "re-check") {
		t.Errorf("exact anchors should not be reported, got %q", got)
	}

	degraded := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "nit", Author: "a", AnchorConfidence: comment.ConfidenceFuzzy},
		{ID: "c2", Line: 6, Text: "nit", Author: "a", AnchorConfidence: comment.ConfidenceSectionLevel},
	})
	if got := plainRail(degraded, 140); !strings.Contains(got, "2 anchors need re-check") {
		t.Errorf("degraded anchors should be reported once, got %q", got)
	}
}

// The rail is chrome on every review screen: exactly one row, exactly the
// width it is given, at any terminal size.
func TestRailIsOneRowAtItsGivenWidth(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "must fix", Author: "a", Blocking: true, AnchorConfidence: comment.ConfidenceFuzzy},
		{ID: "c2", Line: 6, Text: "nit", Author: "a"},
	})
	for _, w := range []int{200, 140, 100, 60, 30, 10, 1} {
		out := m.renderRail(w)
		if n := strings.Count(out, "\n"); n != 0 {
			t.Errorf("width %d: rail must be one row, got %d newlines", w, n)
		}
		if got := lipgloss.Width(out); got > w {
			t.Errorf("width %d: rail overflowed to %d cells", w, got)
		}
	}
}

func TestRailWithoutDocumentDoesNotPanic(t *testing.T) {
	m := testModel(nil)
	m.doc = nil
	if got := plainRail(m, 80); !strings.Contains(got, "APPROVED") {
		t.Errorf("nil document should render an empty approved rail, got %q", got)
	}
}
