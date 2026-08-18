package tui

// The review rail: one row, above the panes, carrying the facts that belong to
// the DOCUMENT rather than to any one thread.
//
// Two things drove it. The gate decision — the whole point of a review pass —
// was reachable only by pressing q and reading the verdict dialog, so the
// screen you spend the review in never said whether the document passed. And
// anchor confidence was being reported per thread, spending up to 9 of a
// sidebar row's 48 columns to repeat one piece of document news (see
// comment.DocumentAnchorHealth).
//
// The rail is deliberately ONE row. It is chrome on every screen in the
// review, and chrome that grows is chrome that eats the artifact.

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/rcliao/comments/pkg/comment"
)

// railState is everything the rail draws, computed from the document alone.
type railState struct {
	decision    string
	blocking    int
	open        int
	resolved    int
	suggestions int
	anchors     comment.AnchorHealth
}

// newRailState derives the rail from the gate, so the rail and the verdict
// dialog can never disagree about whether the document passes.
func (m *Model) newRailState() railState {
	if m.doc == nil {
		return railState{decision: comment.DecisionApproved}
	}
	g := comment.EvaluateGate(m.doc, false)
	st := railState{
		decision:    g.Decision,
		blocking:    len(g.Blocking),
		open:        len(g.NonBlocking),
		suggestions: len(g.PendingSuggestions),
		anchors:     comment.DocumentAnchorHealth(m.doc),
	}
	for _, t := range m.doc.Threads {
		if t != nil && t.Resolved {
			st.resolved++
		}
	}
	return st
}

// renderRail draws the rail row at the given width.
func (m Model) renderRail(width int) string {
	st := (&m).newRailState()
	s := m.styles

	verdict := s.railApproved.Render("✓ APPROVED")
	if st.decision == comment.DecisionChangesRequested {
		verdict = s.blockingMarker.Render("⛔ CHANGES REQUESTED")
	}

	// Counts read left to right in the order they cost you: what blocks the
	// gate, what is merely open, what is already done.
	var counts []string
	if st.blocking > 0 {
		counts = append(counts, s.blockingMarker.Render(fmt.Sprintf("%d blocking", st.blocking)))
	}
	counts = append(counts, s.commentMarker.Render(fmt.Sprintf("%d open", st.open)))
	if st.resolved > 0 {
		counts = append(counts, s.resolvedMarker.Render(fmt.Sprintf("%d resolved", st.resolved)))
	}
	if st.suggestions > 0 {
		counts = append(counts, s.newBadge.Render(fmt.Sprintf("%d pending suggestion%s", st.suggestions, plural(st.suggestions))))
	}
	// Anchor health rides the rail instead of every sidebar row. It appears
	// only when there is something to act on — a clean document says nothing.
	if n := st.anchors.Total(); n > 0 {
		phrase := fmt.Sprintf("%d anchors need re-check", n)
		if n == 1 {
			phrase = "1 anchor needs re-check"
		}
		counts = append(counts, s.newBadge.Render(phrase))
	}

	left := " " + verdict + s.help.Render("   "+strings.Join(counts, s.help.Render(" · ")))
	right := s.help.Render("q  verdict ")

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		// Narrow terminal: the verdict is the half that must survive.
		return truncateANSI(left, max(width, 1))
	}
	return left + strings.Repeat(" ", gap) + right
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// truncateANSI clips a styled string to n visible cells.
func truncateANSI(s string, n int) string {
	if lipgloss.Width(s) <= n {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(n).Render(s)
}
