package tui

// Verdict mode: the exit dialog that applies queued suggestion decisions,
// records a signoff, and quits with an approve / request-changes decision.
// The record it writes is the same ReviewRecord `comments signoff` writes —
// including the optional note (n), so a human reviewing in the TUI can leave
// the agent a message without dropping to the CLI.

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/rcliao/comments/pkg/comment"
)

// verdictNoteRows is the height of the note textarea in the verdict dialog.
const verdictNoteRows = 3

// recordVerdict applies the queued suggestion decisions, writes the signoff
// (decision + note) and quits. Shared by both verdict modes so `a`/`c` behave
// identically whether or not the note has focus.
func (m Model) recordVerdict(decision string) (tea.Model, tea.Cmd) {
	// Apply all queued suggestion decisions atomically, then record the
	// signoff — one save covers content, threads, and the review record
	if err := m.applySuggestionQueue(); err != nil {
		m.err = err
		return m, nil
	}
	comment.AddReviewRecord(m.doc, m.author, decision, strings.TrimSpace(m.verdictNote.Value()), false)
	if err := comment.SaveToSidecar(m.filename, m.doc); err != nil {
		m.err = err
		return m, nil
	}
	m.saveViewStateNow()
	m.VerdictDecision = decision
	return m, tea.Quit
}

// handleVerdictKeys handles the exit verdict dialog: a approve / c request
// changes / n write a note / esc back
func (m Model) handleVerdictKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		return m.recordVerdict(comment.DecisionApproved)
	case "c":
		return m.recordVerdict(comment.DecisionChangesRequested)
	case "n":
		// Write the review note (recorded as ReviewRecord.Note, the same
		// field `signoff --note` writes)
		m.mode = ModeVerdictNote
		m.verdictNote.Focus()
		return m, textarea.Blink
	case "esc", "q":
		// Back to review; queued decisions are kept, not discarded
		m.mode = m.verdictReturnMode
		return m, nil
	}
	return m, nil
}

// handleVerdictNoteKeys types the review note. Every key goes to the textarea
// (so "a" and "c" are letters here, not verdicts); Esc and Ctrl+S both return
// to the verdict dialog keeping what was typed.
func (m Model) handleVerdictNoteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+s":
		m.mode = ModeVerdict
		m.verdictNote.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.verdictNote, cmd = m.verdictNote.Update(msg)
	return m, cmd
}

// viewVerdict renders the exit verdict dialog as a popup over the live review
func (m Model) viewVerdict() string {
	return m.dialogOver(m.baseView(), m.renderVerdictBox())
}

// viewVerdictNote renders the same dialog with the note focused — one box, so
// writing a note never hides the counts the decision is based on.
func (m Model) viewVerdictNote() string {
	return m.dialogOver(m.baseView(), m.renderVerdictBox())
}

// renderVerdictBox draws the verdict dialog: gate counts, the queued-decision
// warning, the review note, and the three actions.
func (m Model) renderVerdictBox() string {
	result := comment.EvaluateGate(m.doc, false)

	var b strings.Builder
	fmt.Fprintf(&b, "Submit review for %s\n\n", m.filename)
	fmt.Fprintf(&b, "%d blocking · %d open · %d pending suggestions\n",
		len(result.Blocking), len(result.NonBlocking), len(result.PendingSuggestions))
	if n := len(m.suggestionQueue); n > 0 {
		fmt.Fprintf(&b, "\n%d queued suggestion decision(s) — applied on submit; Esc keeps them\n", n)
	}

	// The note is recorded in the signoff either way; showing it collapsed
	// when empty keeps the dialog small for the common no-note review
	b.WriteString("\n")
	if m.mode == ModeVerdictNote {
		b.WriteString(m.styles.title.Render("Note (Esc/Ctrl+S: done)"))
		b.WriteString("\n")
		b.WriteString(m.verdictNote.View())
		b.WriteString("\n")
	} else if note := strings.TrimSpace(m.verdictNote.Value()); note != "" {
		b.WriteString(m.styles.title.Render("Note: "))
		b.WriteString(truncate(note, 60, "…"))
		b.WriteString("\n")
	}

	b.WriteString("\n[a] Approve (signoff, exit 0)\n[c] Request changes (signoff, exit 10)\n")
	if m.mode != ModeVerdictNote {
		noteAction := "[n] Add note"
		if strings.TrimSpace(m.verdictNote.Value()) != "" {
			noteAction = "[n] Edit note"
		}
		fmt.Fprintf(&b, "%s\n[Esc] Back to review", noteAction)
	} else {
		b.WriteString("[Esc] Done editing the note")
	}

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 3).Render(b.String())
}
