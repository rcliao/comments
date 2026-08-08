package tui

// Compose modes: add-comment and reply textareas, the resolve confirmation
// dialog, and the add-suggestion form.

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/rcliao/comments/pkg/comment"
)

// handleAddCommentKeys handles keys in add comment mode
func (m Model) handleAddCommentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel comment creation
		m.mode = ModeLineSelect
		m.commentInput.Reset()
		// Reset priority and type to defaults
		m.priority = "medium"
		m.commentType = ""
		return m, nil

	case "ctrl+p":
		// Cycle priority: medium -> high -> low -> medium
		switch m.priority {
		case "medium":
			m.priority = "high"
		case "high":
			m.priority = "low"
		case "low":
			m.priority = "medium"
		default:
			m.priority = "medium"
		}
		return m, nil

	case "ctrl+t":
		// Cycle type: none -> Q -> S -> B -> T -> E -> none
		switch m.commentType {
		case "":
			m.commentType = "Q"
		case "Q":
			m.commentType = "S"
		case "S":
			m.commentType = "B"
		case "B":
			m.commentType = "T"
		case "T":
			m.commentType = "E"
		case "E":
			m.commentType = ""
		default:
			m.commentType = ""
		}
		return m, nil

	case "ctrl+s":
		// Save comment (refresh first: see reply save)
		m.refreshDocFromDisk()
		text := strings.TrimSpace(m.commentInput.Value())
		if text == "" {
			// Empty comment, just cancel
			m.mode = ModeLineSelect
			m.commentInput.Reset()
			// Reset priority and type to defaults
			m.priority = "medium"
			m.commentType = ""
			return m, nil
		}

		// Create new comment with type if specified
		var newComment *comment.Comment
		if m.commentType != "" {
			// Auto-prefix text with type like CLI does
			commentText := "[" + m.commentType + "] " + text
			newComment = comment.NewCommentWithType(m.author, m.selectedLine, commentText, m.commentType)
		} else {
			newComment = comment.NewComment(m.author, m.selectedLine, text)
		}

		// Set priority and status
		newComment.Priority = m.priority
		newComment.Status = "active"

		// Add section metadata if targeting section
		if m.targetIsSection {
			comment.UpdateCommentSection(newComment, m.doc.Content)
		}

		m.doc.Threads = append(m.doc.Threads, newComment)

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.commentViewport.SetContent(m.renderComments())

		// Return to line select mode and reset to defaults
		m.mode = ModeLineSelect
		m.commentInput.Reset()
		m.priority = "medium"
		m.commentType = ""
		return m, nil
	}

	// Handle textarea input
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return m, cmd
}

// handleReplyKeys handles keys in reply mode
func (m Model) handleReplyKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel reply: hand the composer's rows back to the thread, keeping
		// the reader's scroll position
		m.mode = ModeThreadView
		m.replyInput.Reset()
		m.closeComposer()
		return m, nil

	case "ctrl+s":
		// Save reply — refresh from disk first so a long-open session doesn't
		// clobber agent-written threads (last-writer-wins per action)
		m.refreshDocFromDisk()
		text := strings.TrimSpace(m.replyInput.Value())
		if text == "" {
			// Empty reply, just cancel
			m.mode = ModeThreadView
			m.replyInput.Reset()
			m.closeComposer()
			return m, nil
		}

		// Add reply to thread using helper
		if err := comment.AddReplyToThread(m.doc.Threads, m.selectedThread.ID, m.author, text); err != nil {
			m.err = err
			return m, nil
		}

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Return to thread view first, then re-derive the panel geometry:
		// the composer's rows go back to the thread and the viewport lands
		// on the newest activity — the reply just posted
		m.mode = ModeThreadView
		m.replyInput.Reset()
		m.closeComposer()
		m.applyThreadPanel()
		// applyThreadPanel is a no-op without a laid-out screen (unit tests
		// drive handlers directly), so the pane still needs its content set
		m.refreshThreadPane()
		m.commentViewport.SetContent(m.renderComments())
		return m, nil
	}

	// Handle textarea input; the composer may have grown a row, so re-fit the
	// thread pane around it
	var cmd tea.Cmd
	m.replyInput, cmd = m.replyInput.Update(msg)
	m.syncComposerLayout()
	return m, cmd
}

// handleResolveKeys handles keys in resolve confirmation mode
func (m Model) handleResolveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		// Cancel resolution
		m.mode = ModeThreadView
		return m, nil

	case "y", "enter":
		// Confirm resolution (refresh first: see reply save)
		m.refreshDocFromDisk()
		if err := comment.ResolveThread(m.doc.Threads, m.selectedThread.ID); err != nil {
			m.err = err
			return m, nil
		}

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.commentViewport.SetContent(m.renderComments())

		// Return to browse mode (thread is now resolved)
		m.mode = ModeBrowse
		m.selectedThread = nil
		return m, nil
	}

	return m, nil
}

// handleAddSuggestionKeys handles keys in add suggestion mode
func (m Model) handleAddSuggestionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel suggestion creation
		m.mode = ModeLineSelect
		m.suggestionOriginalText = ""
		m.rangeActive = false
		m.suggestionIsSection = false
		m.proposedTextInput.Reset()
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		return m, nil

	case "ctrl+s", "ctrl+d":
		// Submit suggestion (refresh first: see reply save)
		m.refreshDocFromDisk()
		proposedText := m.proposedTextInput.Value()
		if proposedText == "" {
			// Don't create empty suggestion
			m.mode = ModeLineSelect
			m.suggestionOriginalText = ""
			m.rangeActive = false
			m.suggestionIsSection = false
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			return m, nil
		}

		// Use range if set, otherwise fall back to selectedLine
		startLine := m.selectedLine
		endLine := m.selectedLine
		if m.rangeStartLine > 0 && m.rangeEndLine > 0 {
			startLine = m.rangeStartLine
			endLine = m.rangeEndLine
		}

		// Create suggestion using helper (multi-line)
		suggestion := comment.NewSuggestion(
			m.author,
			startLine,
			endLine,
			"Suggestion",
			m.suggestionOriginalText,
			proposedText,
		)

		// Add section metadata if section-based
		if m.suggestionIsSection {
			comment.UpdateCommentSection(suggestion, m.doc.Content)
		}

		// Add to document
		m.doc.Threads = append(m.doc.Threads, suggestion)

		// Reset state
		m.rangeActive = false
		m.suggestionIsSection = false

		// Save document
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.documentViewport.SetContent(m.renderDocument())
		m.commentViewport.SetContent(m.renderComments())

		// Return to browse mode
		m.mode = ModeBrowse
		m.suggestionOriginalText = ""
		m.proposedTextInput.Reset()
		return m, nil
	}

	// Handle textarea input
	var cmd tea.Cmd
	m.proposedTextInput, cmd = m.proposedTextInput.Update(msg)
	return m, cmd
}

// viewAddComment renders the add-comment dialog as a popup over the live
// document view — the cursor line behind the dialog IS the context, so no
// document lines are re-printed inside the box.
func (m Model) viewAddComment() string {
	if !m.ready {
		return "Loading..."
	}

	theme := m.styles.theme

	// Current selection display
	selectionStyle := lipgloss.NewStyle().
		Foreground(theme.Title.Color()).
		Bold(true)

	priorityLabel := selectionStyle.Render(fmt.Sprintf("Priority: %s", strings.ToUpper(m.priority)))

	typeLabel := "None"
	if m.commentType != "" {
		typeLabel = fmt.Sprintf("[%s]", m.commentType)
		switch m.commentType {
		case "Q":
			typeLabel += " Question"
		case "S":
			typeLabel += " Suggestion"
		case "B":
			typeLabel += " Bug"
		case "T":
			typeLabel += " TODO"
		case "E":
			typeLabel += " Enhancement"
		}
	}
	typeDisplay := selectionStyle.Render(fmt.Sprintf("Type: %s", typeLabel))

	selectionInfo := lipgloss.NewStyle().
		Foreground(theme.MetaText.Color()).
		Render(fmt.Sprintf("%s  •  %s", priorityLabel, typeDisplay))

	// Modal overlay for comment input
	var titleText string
	if m.targetIsSection {
		section := m.getSectionAtLine(m.selectedLine)
		if section != nil {
			titleText = fmt.Sprintf("📍 Add Section Comment: %s", section.Title)
		} else {
			titleText = fmt.Sprintf("Add Comment at Line %d", m.selectedLine)
		}
	} else {
		titleText = fmt.Sprintf("💬 Add Comment at Line %d", m.selectedLine)
	}

	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Title.Color()).
		Render(titleText)

	modalHelp := m.styles.help.Render("Ctrl+S: save • Ctrl+P: cycle priority • Ctrl+T: cycle type • Esc: cancel")

	modal := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			m.commentInput.View(),
			"",
			selectionInfo,
			"",
			modalHelp,
		),
	)

	return m.dialogOver(m.baseView(), modal)
}

// viewReply renders the reply composer docked inside the thread panel rather
// than floating over the middle of the screen: you write the reply in the same
// column as the thread you are replying to, with the document still beside it.
// The panel draws the composer itself (renderThreadPanelBox), so there is no
// second box and no repeated "Reply to thread…" title — the panel header is
// still the thread's one header.
func (m Model) viewReply() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}
	if !m.ready {
		// No laid-out screen to dock into (unit tests, startup)
		return m.styles.modalOverlay.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(m.styles.theme.Title.Color()).
					Render(fmt.Sprintf("Reply to Thread at Line %d", m.selectedThread.Line)),
				"",
				m.replyInput.View(),
				"",
				m.styles.help.Render("Ctrl+S: save • Esc: cancel"),
			),
		)
	}
	return m.viewThreadPanel()
}

// viewResolve renders the resolve confirmation as a popup over the live
// document + thread panel.
func (m Model) viewResolve() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	// Confirmation dialog
	confirmTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Title.Color()).
		Render("Resolve this thread?")

	confirmText := lipgloss.NewStyle().
		Render("This will mark the entire conversation as resolved.\nResolved comments can be toggled with 'R' in browse mode.")

	confirmHelp := m.styles.help.Render("y/Enter: confirm • n/Esc: cancel")

	dialog := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			confirmTitle,
			"",
			confirmText,
			"",
			confirmHelp,
		),
	)

	return m.dialogOver(m.baseView(), dialog)
}

// viewAddSuggestion renders the add-suggestion form as a popup over the live
// document view (the selected range stays highlighted behind it). The
// original-text field stays in the form: it is the suggestion's payload
// being edited against, not re-printed document context.
func (m Model) viewAddSuggestion() string {
	// Suggestion creation form
	formTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Warning.Color()).
		Render("Create Edit Suggestion")

	originalLabel := lipgloss.NewStyle().
		Foreground(m.styles.theme.DimSyntax.Color()).
		Render("Original text:")

	originalText := lipgloss.NewStyle().
		Background(m.styles.theme.SelectionBg.Color()).
		Padding(0, 1).
		Render(m.suggestionOriginalText)

	proposedLabel := lipgloss.NewStyle().
		Foreground(m.styles.theme.DimSyntax.Color()).
		Render("Proposed text (edit below):")

	help := m.styles.help.Render("Ctrl+S or Ctrl+D: submit • Esc: cancel")

	dialog := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			formTitle,
			"",
			originalLabel,
			originalText,
			"",
			proposedLabel,
			m.proposedTextInput.View(),
			"",
			help,
		),
	)

	return m.dialogOver(m.baseView(), dialog)
}
