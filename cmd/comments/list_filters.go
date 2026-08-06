package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
)

// filterByAuthor filters comments by author name
func filterByAuthor(comments []*comment.Comment, author string) []*comment.Comment {
	result := make([]*comment.Comment, 0)
	for _, c := range comments {
		if c.Author == author {
			result = append(result, c)
		}
	}
	return result
}

// filterBySearch filters comments by text search (case-insensitive)
func filterBySearch(comments []*comment.Comment, query string) []*comment.Comment {
	query = strings.ToLower(query)
	result := make([]*comment.Comment, 0)
	for _, c := range comments {
		if strings.Contains(strings.ToLower(c.Text), query) {
			result = append(result, c)
		}
	}
	return result
}

// filterByLineRange filters comments within a line range
func filterByLineRange(comments []*comment.Comment, lineRange string) ([]*comment.Comment, error) {
	parts := strings.Split(lineRange, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid line range format. Expected: start-end (e.g., 10-30)")
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid start line: %s", parts[0])
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, fmt.Errorf("invalid end line: %s", parts[1])
	}

	if start > end {
		return nil, fmt.Errorf("start line (%d) must be less than or equal to end line (%d)", start, end)
	}

	result := make([]*comment.Comment, 0)
	for _, c := range comments {
		if c.Line >= start && c.Line <= end {
			result = append(result, c)
		}
	}
	return result, nil
}

// sortComments sorts comments by the specified field
func sortComments(comments []*comment.Comment, sortBy string) {
	switch sortBy {
	case "line":
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].Line < comments[j].Line
		})
	case "timestamp":
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].Timestamp.Before(comments[j].Timestamp)
		})
	case "author":
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].Author < comments[j].Author
		})
	case "priority":
		rank := map[string]int{"high": 0, "medium": 1, "low": 2}
		sort.Slice(comments, func(i, j int) bool {
			return rank[comments[i].GetPriority()] < rank[comments[j].GetPriority()]
		})
	}
}

// outputTable outputs comment threads in table format (v2.0)
func outputTable(threads []*comment.Comment, allThreads []*comment.Comment) {
	// Simple ASCII table
	fmt.Println("┌──────┬──────────────┬──────────┬─────────┬────────────────────────────────────────┐")
	fmt.Println("│ Line │ Author       │ Type     │ Replies │ Preview                                │")
	fmt.Println("├──────┼──────────────┼──────────┼─────────┼────────────────────────────────────────┤")

	for _, thread := range threads {
		commentType := "Root"
		replyCount := thread.CountReplies()

		// Create preview (truncate if too long)
		preview := truncateString(thread.Text, 40)

		resolvedMarker := ""
		if thread.Resolved {
			resolvedMarker = " ✓"
		}

		// Format row with padding
		fmt.Printf("│ %-4d │ %-12s │ %-8s │ %-7d │ %-40s │\n",
			thread.Line,
			truncateString(thread.Author, 12),
			truncateString(commentType+resolvedMarker, 8),
			replyCount,
			preview,
		)
	}

	fmt.Println("└──────┴──────────────┴──────────┴─────────┴────────────────────────────────────────┘")
	fmt.Printf("\nTotal: %d comment thread(s)\n", len(threads))
}

// truncateString truncates a string to a max length in runes (never mid-rune)
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// outputJSON outputs comment threads in JSON format (v2.0)
func outputJSON(threads []*comment.Comment, allThreads []*comment.Comment, docContent string, withContext bool) error {
	// Create a simplified output structure
	type ContextLine struct {
		LineNum  int    `json:"line_num"`
		Text     string `json:"text"`
		IsTarget bool   `json:"is_target"`
	}

	// CommentOutput is the canonical comment view plus context fields
	// (only included when --with-context is specified).
	type CommentOutput struct {
		comment.CommentView
		LineContent   string        `json:"line_content,omitempty"`
		ContextBefore string        `json:"context_before,omitempty"`
		ContextAfter  string        `json:"context_after,omitempty"`
		ContextLines  []ContextLine `json:"context_lines,omitempty"`
	}

	lines := strings.Split(docContent, "\n")

	output := make([]CommentOutput, 0, len(threads))
	for _, thread := range threads {
		commentOut := CommentOutput{CommentView: comment.NewCommentView(thread)}

		// Add context if requested
		if withContext && thread.Line > 0 && thread.Line <= len(lines) {
			// Line content
			commentOut.LineContent = lines[thread.Line-1]

			// Context lines (5 before and 5 after)
			contextSize := 5
			start := thread.Line - contextSize
			if start < 1 {
				start = 1
			}
			end := thread.Line + contextSize
			if end > len(lines) {
				end = len(lines)
			}

			// Build context before
			var beforeLines []string
			for i := start; i < thread.Line; i++ {
				if i > 0 && i <= len(lines) {
					beforeLines = append(beforeLines, lines[i-1])
				}
			}
			commentOut.ContextBefore = strings.Join(beforeLines, "\n")

			// Build context after
			var afterLines []string
			for i := thread.Line + 1; i <= end; i++ {
				if i > 0 && i <= len(lines) {
					afterLines = append(afterLines, lines[i-1])
				}
			}
			commentOut.ContextAfter = strings.Join(afterLines, "\n")

			// Build detailed context lines
			commentOut.ContextLines = make([]ContextLine, 0)
			for i := start; i <= end; i++ {
				if i > 0 && i <= len(lines) {
					commentOut.ContextLines = append(commentOut.ContextLines, ContextLine{
						LineNum:  i,
						Text:     lines[i-1],
						IsTarget: i == thread.Line,
					})
				}
			}
		}

		output = append(output, commentOut)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(output)
}
