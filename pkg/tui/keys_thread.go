package tui

// Thread-view mode: the thread panel over the live document, with replies
// and queueing accept/reject decisions on pending suggestions.
//
// Focus model (fall-through): while the panel is open the screen still reads
// as browse, so keys split three ways —
//   - thread actions stay on the panel: j/k scroll the THREAD, r replies,
//     a/x queue decisions or resolve, Esc closes the panel;
//   - browse-shaped keys fall through with browse semantics instead of dying:
//     c closes the panel and starts the comment flow at the cursor line,
//     q opens the verdict dialog (Esc returns to the panel), ? opens help;
//   - everything else is panel-scoped or ignored (see pkg/tui/CLAUDE.md).

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// handleThreadViewKeys handles keys in thread view mode
func (m Model) handleThreadViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to where the thread was opened from
		if m.returnToLineSelect {
			m.returnToLineSelect = false
			m.selectedThread = nil
			m.mode = ModeLineSelect
			m.refreshCursorView()
			return m, nil
		}
		m.mode = ModeBrowse
		m.selectedThread = nil
		// Drop the panel's anchor highlight; browse focus falls back to the
		// sidebar selection
		m.refreshDocumentPane()
		return m, nil

	case "c":
		// Fall-through: close the panel and start the comment flow at the
		// cursor line, exactly as c does in browse
		m.selectedThread = nil
		m.returnToLineSelect = false
		m.mode = ModeLineSelect
		m.selectedLine = max(m.selectedLine, 1)
		m.documentViewport = newViewport(m.docPaneWidth(), m.height-2)
		m.refreshCursorView()
		return m, nil

	case "q":
		// Fall-through: q reads as browse's verdict entry, not app-quit;
		// Esc from the verdict returns to the open panel
		m.verdictReturnMode = ModeThreadView
		m.mode = ModeVerdict
		return m, nil

	case "?":
		// Fall-through: help overlay over the doc+panel view
		m.helpReturnMode = m.mode
		m.mode = ModeHelp
		return m, nil

	case "r":
		// Enter reply mode: the composer docks inside this panel, under the
		// thread it replies to
		m.mode = ModeReply
		m.replyInput.Reset()
		m.replyInput.Focus()
		m.applyComposerLayout()
		return m, textarea.Blink

	case "a":
		// Queue an accept for this pending suggestion; nothing mutates
		// until the verdict dialog applies the queue ("queue until verdict")
		if m.selectedThread != nil && m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
			m.queueDecision(m.selectedThread.ID, true)
			m.refreshThreadPane()
		}
		return m, nil

	case "x":
		// Queue a reject for a pending suggestion; otherwise resolve thread
		if m.selectedThread != nil && m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
			m.queueDecision(m.selectedThread.ID, false)
			m.refreshThreadPane()
			return m, nil
		}
		// Otherwise, enter resolve mode for regular threads
		m.mode = ModeResolve
		return m, nil
	}

	// Scroll thread viewport
	var cmd tea.Cmd
	m.threadViewport, cmd = m.threadViewport.Update(msg)
	return m, cmd
}
