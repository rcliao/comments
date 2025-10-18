package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/rcliao/comments/pkg/comment"
)

// renderDocument renders the document with line numbers and comment markers
func (m *Model) renderDocument() string {
	if m.doc == nil {
		return "No document loaded"
	}

	lines := strings.Split(m.doc.Content, "\n")
	var rendered strings.Builder

	// Calculate available width for text: viewport width - line number (4) - marker (3) - spacing (3)
	availableWidth := m.documentViewport.Width - 10
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Group comments by line (only root comments)
	commentsByLine := comment.GroupCommentsByLine(m.doc.Comments)

	for i, line := range lines {
		lineNum := i + 1
		lineNumStr := lineNumberStyle.Render(fmt.Sprintf("%d", lineNum))

		// Add comment marker if this line has comments
		marker := "  "
		if comments := commentsByLine[lineNum]; len(comments) > 0 {
			marker = commentMarkerStyle.Render(fmt.Sprintf("💬%d", len(comments)))
		}

		// Wrap long lines
		wrappedLines := strings.Split(wordwrap.String(line, availableWidth), "\n")
		for j, wrappedLine := range wrappedLines {
			if j == 0 {
				// First line: show line number and marker
				rendered.WriteString(fmt.Sprintf("%s %s %s\n", lineNumStr, marker, wrappedLine))
			} else {
				// Continuation lines: indent with spaces
				rendered.WriteString(fmt.Sprintf("%s %s %s\n", strings.Repeat(" ", 4), "  ", wrappedLine))
			}
		}
	}

	return rendered.String()
}

// renderDocumentWithCursor renders the document with a cursor for line selection
func (m *Model) renderDocumentWithCursor() string {
	if m.doc == nil {
		return "No document loaded"
	}

	lines := strings.Split(m.doc.Content, "\n")
	var rendered strings.Builder

	// Calculate available width for text: viewport width - cursor (2) - line number (4) - marker (3) - spacing (3)
	availableWidth := m.documentViewport.Width - 12
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Group comments by line
	commentsByLine := comment.GroupCommentsByLine(m.doc.Comments)

	for i, line := range lines {
		lineNum := i + 1
		lineNumStr := lineNumberStyle.Render(fmt.Sprintf("%d", lineNum))

		// Add comment marker if this line has comments
		marker := "  "
		if comments := commentsByLine[lineNum]; len(comments) > 0 {
			marker = commentMarkerStyle.Render(fmt.Sprintf("💬%d", len(comments)))
		}

		// Highlight cursor line
		cursor := "  "
		isSelected := lineNum == m.selectedLine

		// Wrap long lines
		wrappedLines := strings.Split(wordwrap.String(line, availableWidth), "\n")
		for j, wrappedLine := range wrappedLines {
			if j == 0 {
				// First line: show cursor, line number and marker
				if isSelected {
					cursor = cursorStyle.Render("▶")
					wrappedLine = cursorStyle.Render(wrappedLine)
				}
				rendered.WriteString(fmt.Sprintf("%s %s %s %s\n", cursor, lineNumStr, marker, wrappedLine))
			} else {
				// Continuation lines: indent with spaces
				displayCursor := "  "
				if isSelected {
					displayCursor = cursorStyle.Render("  ")
					wrappedLine = cursorStyle.Render(wrappedLine)
				}
				rendered.WriteString(fmt.Sprintf("%s %s %s %s\n", displayCursor, strings.Repeat(" ", 4), "  ", wrappedLine))
			}
		}
	}

	return rendered.String()
}

// getCommentTypeColor returns the color for a comment based on its type prefix
func getCommentTypeColor(text string) string {
	if len(text) < 3 {
		return ""
	}

	prefix := text[:3]
	switch prefix {
	case "[B]": // Blocker
		return "196" // Red
	case "[Q]": // Question
		return "220" // Yellow
	case "[S]": // Suggestion
		return "33"  // Blue
	case "[T]": // Technical
		return "13"  // Magenta
	case "[E]": // Editorial
		return "14"  // Cyan
	default:
		return ""
	}
}

// renderComments renders the comment panel
func (m *Model) renderComments() string {
	if m.doc == nil {
		return "No comments"
	}

	visibleComments := comment.GetVisibleComments(m.doc.Comments, m.showResolved)
	if len(visibleComments) == 0 {
		if m.showResolved {
			return "No comments"
		}
		return "No unresolved comments\n\nPress R to show resolved"
	}

	var rendered strings.Builder
	statusText := "unresolved"
	if m.showResolved {
		statusText = "all"
	}
	rendered.WriteString(fmt.Sprintf("Comments (%d %s)\n\n", len(visibleComments), statusText))

	for i, c := range visibleComments {
		// Get thread info
		thread, hasThread := m.threads[c.ThreadID]
		replyCount := 0
		if hasThread {
			replyCount = len(thread.Replies)
		}

		// Highlight selected comment
		style := lipgloss.NewStyle()
		if i == m.selectedComment {
			style = selectedCommentStyle
		} else {
			// Apply color-coding based on comment type
			if typeColor := getCommentTypeColor(c.Text); typeColor != "" {
				style = style.Foreground(lipgloss.Color(typeColor))
			}
		}

		// Build comment text
		var commentText string
		if c.Resolved {
			commentText = fmt.Sprintf("✓ Line %d • @%s\n%s\n%s\n└─ %d replies",
				c.Line,
				c.Author,
				c.Timestamp.Format("2006-01-02 15:04"),
				c.Text,
				replyCount,
			)
		} else {
			commentText = fmt.Sprintf("Line %d • @%s\n%s\n%s\n└─ %d replies",
				c.Line,
				c.Author,
				c.Timestamp.Format("2006-01-02 15:04"),
				c.Text,
				replyCount,
			)
		}

		rendered.WriteString(style.Render(commentText))
		rendered.WriteString("\n\n")
	}

	return rendered.String()
}

// renderThread renders an expanded thread view
func (m *Model) renderThread() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	var rendered strings.Builder

	// Thread header
	rendered.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("Thread at Line %d\n", m.selectedThread.Line)))
	rendered.WriteString("\n")

	// Document context - show lines around the comment
	contextLines := m.getContextLines(m.selectedThread.Line, 2) // 2 lines before/after
	if len(contextLines) > 0 {
		contextStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(m.width - 8)

		// Calculate width for context text wrapping
		contextWidth := m.width - 22 // Account for borders, padding, line numbers, markers
		if contextWidth < 40 {
			contextWidth = 40
		}

		var contextText strings.Builder
		contextText.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Document Context:"))
		contextText.WriteString("\n\n")

		for _, cl := range contextLines {
			marker := " "
			lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			lineStyle := lipgloss.NewStyle()

			// Wrap the line text
			wrappedLines := strings.Split(wordwrap.String(cl.Text, contextWidth), "\n")

			for i, wrappedLine := range wrappedLines {
				if cl.LineNum == m.selectedThread.Line {
					if i == 0 {
						marker = lipgloss.NewStyle().
							Foreground(lipgloss.Color("170")).
							Bold(true).
							Render("►")
						lineNumStyle = lineNumStyle.Bold(true).Foreground(lipgloss.Color("170"))
						lineStyle = lineStyle.
							Background(lipgloss.Color("235")).
							Foreground(lipgloss.Color("255")).
							Bold(true)
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							marker,
							lineNumStyle.Render(fmt.Sprintf("%4d", cl.LineNum)),
							lineStyle.Render(wrappedLine)))
					} else {
						// Continuation lines for highlighted line
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							" ",
							strings.Repeat(" ", 4),
							lineStyle.Render(wrappedLine)))
					}
				} else {
					if i == 0 {
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							marker,
							lineNumStyle.Render(fmt.Sprintf("%4d", cl.LineNum)),
							wrappedLine))
					} else {
						// Continuation lines for non-highlighted line
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							" ",
							strings.Repeat(" ", 4),
							wrappedLine))
					}
				}
			}
		}

		rendered.WriteString(contextStyle.Render(contextText.String()))
		rendered.WriteString("\n\n")
	}

	// Root comment
	rootStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1).
		Width(m.width - 8)

	// Wrap root comment text to fit within the box
	rootTextWidth := m.width - 16 // Account for border, padding, and margins
	if rootTextWidth < 40 {
		rootTextWidth = 40
	}
	wrappedRootText := wordwrap.String(m.selectedThread.RootComment.Text, rootTextWidth)

	rootText := fmt.Sprintf("@%s · %s\n\n%s",
		m.selectedThread.RootComment.Author,
		m.selectedThread.RootComment.Timestamp.Format("2006-01-02 15:04"),
		wrappedRootText,
	)

	if m.selectedThread.Resolved {
		rootText = "✓ RESOLVED\n\n" + rootText
	}

	rendered.WriteString(rootStyle.Render(rootText))
	rendered.WriteString("\n\n")

	// Replies
	if len(m.selectedThread.Replies) > 0 {
		rendered.WriteString(lipgloss.NewStyle().Bold(true).Render(
			fmt.Sprintf("Replies (%d):", len(m.selectedThread.Replies))))
		rendered.WriteString("\n\n")

		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

		// Calculate available width for reply text: width - padding - border characters
		replyWidth := m.width - 12
		if replyWidth < 40 {
			replyWidth = 40
		}

		for _, reply := range m.selectedThread.Replies {
			// Reply header with styled border and author
			rendered.WriteString(borderStyle.Render("│ "))
			rendered.WriteString(authorStyle.Render(fmt.Sprintf("@%s · %s",
				reply.Author,
				reply.Timestamp.Format("2006-01-02 15:04"))))
			rendered.WriteString("\n")

			// Wrap and render reply text
			lines := strings.Split(reply.Text, "\n")
			for _, line := range lines {
				// Wrap each line if it's too long
				wrappedLines := strings.Split(wordwrap.String(line, replyWidth), "\n")
				for _, wrappedLine := range wrappedLines {
					rendered.WriteString(borderStyle.Render("│ "))
					rendered.WriteString(wrappedLine)
					rendered.WriteString("\n")
				}
			}
			rendered.WriteString("\n")
		}
	} else {
		rendered.WriteString(helpStyle.Render("No replies yet\n\nPress 'r' to add a reply"))
	}

	return rendered.String()
}

// ContextLine represents a line with its line number for context display
type ContextLine struct {
	LineNum int
	Text    string
}

// getContextLines extracts lines around a specific line number for context
func (m *Model) getContextLines(lineNum int, contextSize int) []ContextLine {
	if m.doc == nil {
		return nil
	}

	lines := strings.Split(m.doc.Content, "\n")
	var result []ContextLine

	// Calculate range
	start := lineNum - contextSize - 1 // -1 for 0-based indexing
	if start < 0 {
		start = 0
	}

	end := lineNum + contextSize // inclusive
	if end > len(lines) {
		end = len(lines)
	}

	// Extract lines
	for i := start; i < end; i++ {
		result = append(result, ContextLine{
			LineNum: i + 1, // 1-based for display
			Text:    lines[i],
		})
	}

	return result
}
