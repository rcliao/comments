package tui

// Compose modes: add-comment and reply textareas, the resolve confirmation
// dialog, and the add-suggestion form.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		// Save comment
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
		// Cancel reply
		m.mode = ModeThreadView
		m.commentInput.Reset()
		return m, nil

	case "ctrl+s":
		// Save reply
		text := strings.TrimSpace(m.commentInput.Value())
		if text == "" {
			// Empty reply, just cancel
			m.mode = ModeThreadView
			m.commentInput.Reset()
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

		// Refresh views
		m.threadViewport.SetContent(m.renderThread())
		m.commentViewport.SetContent(m.renderComments())

		// Return to thread view
		m.mode = ModeThreadView
		m.commentInput.Reset()
		return m, nil
	}

	// Handle textarea input
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
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
		// Confirm resolution
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
		// Submit suggestion
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

// viewAddComment renders the add comment modal
func (m Model) viewAddComment() string {
	if !m.ready {
		return "Loading..."
	}

	theme := m.styles.theme

	// Base layout with document
	modeStr := "Adding Comment"
	title := m.styles.title.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		m.styles.commentPanel.Render(m.commentViewport.View()),
	)

	// Get section-aware context
	var contextText string
	sectionContext := m.getSectionContext(m.selectedLine)
	if sectionContext != "" {
		// Use section-aware context
		contextText = sectionContext
	} else {
		// Fall back to line-based context if no section found
		contextLines := m.getContextLines(m.selectedLine, 2)
		var builder strings.Builder

		contextStyle := lipgloss.NewStyle().
			Foreground(theme.MetaText).
			Italic(true)
		lineNumStyle := lipgloss.NewStyle().Foreground(theme.LineNumber)
		highlightStyle := lipgloss.NewStyle().
			Background(theme.SelectionBg).
			Foreground(theme.SelectionFg).
			Bold(true)

		builder.WriteString(contextStyle.Render("Document Context:"))
		builder.WriteString("\n")

		for _, cl := range contextLines {
			linePrefix := fmt.Sprintf("%4d │ ", cl.LineNum)
			if cl.LineNum == m.selectedLine {
				builder.WriteString(lineNumStyle.Bold(true).Render(linePrefix))
				builder.WriteString(highlightStyle.Render(cl.Text))
			} else {
				builder.WriteString(lineNumStyle.Render(linePrefix))
				builder.WriteString(cl.Text)
			}
			builder.WriteString("\n")
		}
		contextText = builder.String()
	}

	// Current selection display
	selectionStyle := lipgloss.NewStyle().
		Foreground(theme.Title).
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
		Foreground(theme.MetaText).
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
		Foreground(theme.Title).
		Render(titleText)

	modalHelp := m.styles.help.Render("Ctrl+S: save • Ctrl+P: cycle priority • Ctrl+T: cycle type • Esc: cancel")

	modal := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			contextText,
			"",
			m.commentInput.View(),
			"",
			selectionInfo,
			"",
			modalHelp,
		),
	)

	// Position modal over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.Place(
			m.width,
			m.height-2,
			lipgloss.Left,
			lipgloss.Top,
			content,
		),
		positioned,
	)
}

// viewReply renders the reply modal
func (m Model) viewReply() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	title := m.styles.title.Render(fmt.Sprintf("Thread at Line %d", m.selectedThread.Line))

	// Thread content as background
	threadContent := m.threadViewport.View()

	// Build thread context to show in modal
	var threadContext strings.Builder
	contextStyle := lipgloss.NewStyle().
		Foreground(m.styles.theme.MetaText).
		Italic(true)

	threadContext.WriteString(contextStyle.Render("Thread Context:"))
	threadContext.WriteString("\n\n")

	// Root comment (selectedThread IS the root comment in v2.0)
	fmt.Fprintf(&threadContext, "┌ @%s · %s\n",
		m.selectedThread.Author,
		m.selectedThread.Timestamp.Format("2006-01-02 15:04"))

	// Truncate root comment if too long (rune-safe)
	rootText := truncate(m.selectedThread.Text, 60, "...")
	fmt.Fprintf(&threadContext, "│ %s\n", rootText)

	// Show recent replies (last 2)
	replyCount := len(m.selectedThread.Replies)
	if replyCount > 0 {
		startIdx := 0
		if replyCount > 2 {
			fmt.Fprintf(&threadContext, "│ ... (%d earlier replies)\n", replyCount-2)
			startIdx = replyCount - 2
		}

		for i := startIdx; i < replyCount; i++ {
			reply := m.selectedThread.Replies[i]
			fmt.Fprintf(&threadContext, "├ @%s · %s\n",
				reply.Author,
				reply.Timestamp.Format("2006-01-02 15:04"))

			// Truncate reply if too long (rune-safe)
			replyText := truncate(reply.Text, 60, "...")
			fmt.Fprintf(&threadContext, "│ %s\n", replyText)
		}
	}
	threadContext.WriteString("└──────────────────────\n")

	// Modal overlay for reply input
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Title).
		Render("Reply to Thread")

	modalHelp := m.styles.help.Render("Ctrl+S: save • Esc: cancel")

	modal := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			threadContext.String(),
			"",
			m.commentInput.View(),
			"",
			modalHelp,
		),
	)

	// Position modal over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.Place(
			m.width,
			m.height-2,
			lipgloss.Left,
			lipgloss.Top,
			threadContent,
		),
		positioned,
	)
}

// viewResolve renders the resolve confirmation dialog
func (m Model) viewResolve() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	title := m.styles.title.Render(fmt.Sprintf("Thread at Line %d", m.selectedThread.Line))

	// Thread content as background
	threadContent := m.threadViewport.View()

	// Confirmation dialog
	confirmTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Title).
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

	// Position dialog over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.Place(
			m.width,
			m.height-2,
			lipgloss.Left,
			lipgloss.Top,
			threadContent,
		),
		positioned,
	)
}

// viewAddSuggestion renders the add suggestion modal
func (m Model) viewAddSuggestion() string {
	title := m.styles.title.Render(fmt.Sprintf("Add Suggestion for Line %d", m.selectedLine))

	// Document context as background
	docContent := m.documentViewport.View()

	// Suggestion creation form
	formTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Warning).
		Render("Create Edit Suggestion")

	originalLabel := lipgloss.NewStyle().
		Foreground(m.styles.theme.DimSyntax).
		Render("Original text:")

	originalText := lipgloss.NewStyle().
		Background(m.styles.theme.SelectionBg).
		Padding(0, 1).
		Render(m.suggestionOriginalText)

	proposedLabel := lipgloss.NewStyle().
		Foreground(m.styles.theme.DimSyntax).
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

	// Position dialog over content
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.Place(
			m.width,
			m.height-2,
			lipgloss.Left,
			lipgloss.Top,
			docContent,
		),
		positioned,
	)
}
