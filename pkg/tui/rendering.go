package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rcliao/comments/pkg/comment"
)

// renderDocument renders the document with line numbers and comment markers
func (m *Model) renderDocument() string {
	if m.doc == nil {
		return "No document loaded"
	}

	lines := strings.Split(m.doc.Content, "\n")
	var rendered strings.Builder

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

		rendered.WriteString(fmt.Sprintf("%s %s %s\n", lineNumStr, marker, line))
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
		if lineNum == m.selectedLine {
			cursor = cursorStyle.Render("▶")
			line = cursorStyle.Render(line)
		}

		rendered.WriteString(fmt.Sprintf("%s %s %s %s\n", cursor, lineNumStr, marker, line))
	}

	return rendered.String()
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

	// Root comment
	rootStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1).
		Width(m.width - 8)

	rootText := fmt.Sprintf("@%s · %s\n\n%s",
		m.selectedThread.RootComment.Author,
		m.selectedThread.RootComment.Timestamp.Format("2006-01-02 15:04"),
		m.selectedThread.RootComment.Text,
	)

	if m.selectedThread.Resolved {
		rootText = "✓ RESOLVED\n\n" + rootText
	}

	rendered.WriteString(rootStyle.Render(rootText))
	rendered.WriteString("\n\n")

	// Replies
	if len(m.selectedThread.Replies) > 0 {
		rendered.WriteString(lipgloss.NewStyle().Bold(true).Render(
			fmt.Sprintf("Replies (%d):\n\n", len(m.selectedThread.Replies))))

		replyStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(2).
			Width(m.width - 10)

		for _, reply := range m.selectedThread.Replies {
			replyText := fmt.Sprintf("@%s · %s\n%s",
				reply.Author,
				reply.Timestamp.Format("2006-01-02 15:04"),
				reply.Text,
			)
			rendered.WriteString(replyStyle.Render(replyText))
			rendered.WriteString("\n\n")
		}
	} else {
		rendered.WriteString(helpStyle.Render("No replies yet\n\nPress 'r' to add a reply"))
	}

	return rendered.String()
}
