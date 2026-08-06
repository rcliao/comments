package tui

// Thread-view mode: an expanded thread with replies, plus queueing
// accept/reject decisions on pending suggestions.

import (
	"fmt"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
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
		return m, nil

	case "q":
		// If file was provided directly, quit the app
		// Otherwise, go back to file picker
		if m.startedWithFile {
			m.saveViewStateNow()
			return m, tea.Quit
		}
		m.mode = ModeFilePicker
		m.selectedThread = nil
		m.doc = nil
		m.filename = ""
		m.ready = false
		return m, nil

	case "r":
		// Enter reply mode
		m.mode = ModeReply
		m.commentInput.Reset()
		m.commentInput.Focus()
		return m, textarea.Blink

	case "a":
		// Queue an accept for this pending suggestion; nothing mutates
		// until the verdict dialog applies the queue ("queue until verdict")
		if m.selectedThread != nil && m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
			m.queueDecision(m.selectedThread.ID, true)
			m.threadViewport.SetContent(m.renderThread())
		}
		return m, nil

	case "x":
		// Queue a reject for a pending suggestion; otherwise resolve thread
		if m.selectedThread != nil && m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
			m.queueDecision(m.selectedThread.ID, false)
			m.threadViewport.SetContent(m.renderThread())
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

// viewThread renders the thread view
func (m Model) viewThread() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	title := m.styles.title.Render(fmt.Sprintf("Thread at Line %d", m.selectedThread.Line))

	quitText := "file picker"
	if m.startedWithFile {
		quitText = "quit"
	}
	actionText := "x: resolve"
	if m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
		actionText = "a/x: queue accept/reject"
	}
	help := m.styles.help.Render(fmt.Sprintf("r: reply • %s • Esc: back • q: %s", actionText, quitText))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.threadViewport.View(),
		"",
		help,
	)
}
