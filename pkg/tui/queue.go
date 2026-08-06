package tui

// Queued suggestion decisions: a/x during review mark suggestions
// accept/reject pending in the model; nothing mutates until the verdict
// dialog submits, when all queued decisions apply atomically before the
// signoff is recorded ("queue until verdict" — decided in review).

import (
	"fmt"
	"sort"

	"github.com/rcliao/comments/pkg/comment"
)

// queueDecision records a pending accept/reject for a suggestion thread
func (m *Model) queueDecision(suggestionID string, accept bool) {
	if m.suggestionQueue == nil {
		m.suggestionQueue = map[string]bool{}
	}
	m.suggestionQueue[suggestionID] = accept
}

// queuedLabel describes the queued decision for a suggestion, if any
func (m *Model) queuedLabel(suggestionID string) string {
	decision, ok := m.suggestionQueue[suggestionID]
	if !ok {
		return ""
	}
	if decision {
		return "QUEUED: ACCEPT"
	}
	return "QUEUED: REJECT"
}

// applySuggestionQueue applies all queued accept/reject decisions to the
// document. Accepts are applied bottom-up (descending StartLine) so earlier
// suggestions' line ranges are untouched by later edits; comment anchors are
// recalculated after each applied edit. The caller saves once afterwards.
func (m *Model) applySuggestionQueue() error {
	if len(m.suggestionQueue) == 0 {
		return nil
	}

	pending := comment.GetPendingSuggestions(m.doc.Threads)
	byID := map[string]*comment.Comment{}
	for _, s := range pending {
		byID[s.ID] = s
	}

	accepts := []*comment.Comment{}
	for id, accept := range m.suggestionQueue {
		s, ok := byID[id]
		if !ok {
			// Decided elsewhere (e.g. CLI) since it was queued — skip
			continue
		}
		if accept {
			accepts = append(accepts, s)
		} else {
			if err := comment.RejectSuggestion(m.doc.Threads, s.ID); err != nil {
				return fmt.Errorf("rejecting %s: %w", s.ID, err)
			}
		}
	}

	// Bottom-up so each apply leaves the lines above it untouched
	sort.Slice(accepts, func(i, j int) bool { return accepts[i].StartLine > accepts[j].StartLine })

	for _, s := range accepts {
		if _, err := comment.ApplyAndAcceptSuggestion(m.doc, s.ID); err != nil {
			return fmt.Errorf("accepting %s: %w", s.ID, err)
		}
	}
	m.suggestionQueue = map[string]bool{}
	return nil
}
