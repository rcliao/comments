package main

import (
	"fmt"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/markdown"
)

// CommentContext represents the context information for a comment
type CommentContext struct {
	SectionPath     string
	SectionHeading  string
	SectionRange    string
	ContextLines    []ContextLine
	OriginalText    string // For suggestions
	ProposedText    string // For suggestions
}

// ContextLine represents a single line with its number and text
type ContextLine struct {
	LineNum int
	Text    string
	IsTarget bool
}

// getCommentContext extracts context information for a comment
func getCommentContext(c *comment.Comment, docContent string) CommentContext {
	ctx := CommentContext{}

	lines := strings.Split(docContent, "\n")

	// Get section information if available
	if c.SectionPath != "" {
		ctx.SectionPath = c.SectionPath

		// Parse document to get section details
		docStructure := markdown.ParseDocument(docContent)
		section, exists := docStructure.SectionsByLine[c.Line]
		if exists {
			ctx.SectionHeading = section.Title
			ctx.SectionRange = fmt.Sprintf("lines %d-%d", section.StartLine, section.EndLine)
		}
	}

	// Get context lines (5 lines before and after, or less if at boundaries)
	contextSize := 5
	start := c.Line - contextSize
	if start < 1 {
		start = 1
	}
	end := c.Line + contextSize
	if end > len(lines) {
		end = len(lines)
	}

	ctx.ContextLines = make([]ContextLine, 0)
	for i := start; i <= end; i++ {
		if i > 0 && i <= len(lines) {
			ctx.ContextLines = append(ctx.ContextLines, ContextLine{
				LineNum: i,
				Text: lines[i-1],
				IsTarget: i == c.Line,
			})
		}
	}

	// For suggestions, include original and proposed text
	if c.IsSuggestion {
		ctx.OriginalText = c.OriginalText
		ctx.ProposedText = c.ProposedText
	}

	return ctx
}

// formatCommentWithContext formats a comment with its context for display
func formatCommentWithContext(c *comment.Comment, ctx CommentContext, includeReplies bool) string {
	var output strings.Builder

	// Header with ID and metadata
	output.WriteString(fmt.Sprintf("━━━ Comment ID: %s ━━━\n", c.ID))
	output.WriteString(fmt.Sprintf("Author: @%s\n", c.Author))
	output.WriteString(fmt.Sprintf("Timestamp: %s\n", c.Timestamp.Format("2006-01-02 15:04:05")))

	// Location info
	if ctx.SectionPath != "" {
		output.WriteString(fmt.Sprintf("Location: 📍 %s (Line %d)\n", ctx.SectionPath, c.Line))
		if ctx.SectionHeading != "" {
			output.WriteString(fmt.Sprintf("Section: %s (%s)\n", ctx.SectionHeading, ctx.SectionRange))
		}
	} else {
		output.WriteString(fmt.Sprintf("Location: 💬 Line %d\n", c.Line))
	}

	// Type and status
	if c.Type != "" {
		output.WriteString(fmt.Sprintf("Type: [%s]\n", c.Type))
	}
	if c.IsSuggestion {
		status := "pending"
		if c.Accepted != nil {
			if *c.Accepted {
				status = "accepted"
			} else {
				status = "rejected"
			}
		}
		output.WriteString(fmt.Sprintf("Suggestion: %s\n", status))
	}
	if c.Resolved {
		output.WriteString("Status: ✓ Resolved\n")
	}

	output.WriteString("\n")

	// Comment text
	output.WriteString("Comment:\n")
	output.WriteString(fmt.Sprintf("  %s\n", c.Text))
	output.WriteString("\n")

	// Context section
	if len(ctx.ContextLines) > 0 {
		output.WriteString("Document Context:\n")
		output.WriteString("─────────────────\n")
		for _, line := range ctx.ContextLines {
			marker := " "
			if line.IsTarget {
				marker = "►"
			}
			output.WriteString(fmt.Sprintf("%s %4d │ %s\n", marker, line.LineNum, line.Text))
		}
		output.WriteString("\n")
	}

	// Suggestion details
	if c.IsSuggestion {
		output.WriteString("Suggestion Details:\n")
		output.WriteString("───────────────────\n")
		output.WriteString(fmt.Sprintf("Lines: %d-%d\n\n", c.StartLine, c.EndLine))

		if ctx.OriginalText != "" {
			output.WriteString("Original:\n")
			for _, line := range strings.Split(ctx.OriginalText, "\n") {
				output.WriteString(fmt.Sprintf("  - %s\n", line))
			}
			output.WriteString("\n")
		}

		if ctx.ProposedText != "" {
			output.WriteString("Proposed:\n")
			for _, line := range strings.Split(ctx.ProposedText, "\n") {
				output.WriteString(fmt.Sprintf("  + %s\n", line))
			}
			output.WriteString("\n")
		}
	}

	// Replies
	if includeReplies && len(c.Replies) > 0 {
		output.WriteString(fmt.Sprintf("Replies (%d):\n", len(c.Replies)))
		output.WriteString("─────────\n")
		for i, reply := range c.Replies {
			output.WriteString(fmt.Sprintf("[%d] @%s · %s\n", i+1, reply.Author, reply.Timestamp.Format("2006-01-02 15:04")))
			output.WriteString(fmt.Sprintf("    %s\n", reply.Text))
			if i < len(c.Replies)-1 {
				output.WriteString("\n")
			}
		}
	}

	return output.String()
}

// formatListWithContext formats a list of comments with context
func formatListWithContext(comments []*comment.Comment, docContent string) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("Found %d comment thread(s) with context\n\n", len(comments)))

	for i, c := range comments {
		ctx := getCommentContext(c, docContent)
		output.WriteString(formatCommentWithContext(c, ctx, false))

		if i < len(comments)-1 {
			output.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
		}
	}

	return output.String()
}
