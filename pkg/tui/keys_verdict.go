package tui

// Verdict mode: the exit dialog that applies queued suggestion decisions,
// records a signoff, and quits with an approve / request-changes decision.

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/rcliao/comments/pkg/comment"
)

// handleVerdictKeys handles the exit verdict dialog: a approve / c request changes / esc back
func (m Model) handleVerdictKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	record := func(decision string) (tea.Model, tea.Cmd) {
		// Apply all queued suggestion decisions atomically, then record the
		// signoff — one save covers content, threads, and the review record
		if err := m.applySuggestionQueue(); err != nil {
			m.err = err
			return m, nil
		}
		comment.AddReviewRecord(m.doc, m.author, decision, "", false)
		if err := comment.SaveToSidecar(m.filename, m.doc); err != nil {
			m.err = err
			return m, nil
		}
		m.saveViewStateNow()
		m.VerdictDecision = decision
		return m, tea.Quit
	}
	switch msg.String() {
	case "a":
		return record(comment.DecisionApproved)
	case "c":
		return record(comment.DecisionChangesRequested)
	case "esc", "q":
		// Back to review; queued decisions are kept, not discarded
		m.mode = m.verdictReturnMode
		return m, nil
	}
	return m, nil
}

// viewVerdict renders the exit verdict dialog over a dimmed summary
func (m Model) viewVerdict() string {
	result := comment.EvaluateGate(m.doc, false)
	queueLine := ""
	if n := len(m.suggestionQueue); n > 0 {
		queueLine = fmt.Sprintf("\n%d queued suggestion decision(s) — applied on submit; Esc keeps them\n", n)
	}
	dialog := fmt.Sprintf(
		"Submit review for %s\n\n%d blocking · %d open · %d pending suggestions\n%s\n[a] Approve (signoff, exit 0)\n[c] Request changes (signoff, exit 10)\n[Esc] Back to review",
		m.filename, len(result.Blocking), len(result.NonBlocking), len(result.PendingSuggestions), queueLine)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 3).Render(dialog)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
