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
	}
}

// outputTable outputs comments in table format
func outputTable(comments []*comment.Comment, positions map[string]comment.Position, allComments []*comment.Comment) {
	threads := comment.BuildThreads(allComments)

	// Simple ASCII table
	fmt.Println("┌──────┬──────────────┬──────────┬─────────┬────────────────────────────────────────┐")
	fmt.Println("│ Line │ Author       │ Type     │ Replies │ Preview                                │")
	fmt.Println("├──────┼──────────────┼──────────┼─────────┼────────────────────────────────────────┤")

	rootCount := 0
	for _, c := range comments {
		// Only show root comments in table
		if c.ParentID != "" {
			continue
		}
		rootCount++

		pos := positions[c.ID]
		commentType := "Root"
		replyCount := "0"

		if thread, ok := threads[c.ThreadID]; ok {
			replyCount = fmt.Sprintf("%d", len(thread.Replies))
		}

		// Create preview (truncate if too long)
		preview := c.Text
		if len(preview) > 40 {
			preview = preview[:37] + "..."
		}

		resolvedMarker := ""
		if c.Resolved {
			resolvedMarker = " ✓"
		}

		// Format row with padding
		fmt.Printf("│ %-4d │ %-12s │ %-8s │ %-7s │ %-40s │\n",
			pos.Line,
			truncateString(c.Author, 12),
			truncateString(commentType+resolvedMarker, 8),
			replyCount,
			preview,
		)
	}

	fmt.Println("└──────┴──────────────┴──────────┴─────────┴────────────────────────────────────────┘")
	fmt.Printf("\nTotal: %d comment thread(s)\n", rootCount)
}

// truncateString truncates a string to a max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// outputJSON outputs comments in JSON format
func outputJSON(comments []*comment.Comment, positions map[string]comment.Position, allComments []*comment.Comment) error {
	threads := comment.BuildThreads(allComments)

	// Create a simplified output structure
	type CommentOutput struct {
		ID        string `json:"id"`
		ThreadID  string `json:"thread_id"`
		ParentID  string `json:"parent_id,omitempty"`
		Author    string `json:"author"`
		Line      int    `json:"line"`
		Timestamp string `json:"timestamp"`
		Text      string `json:"text"`
		Type      string `json:"type,omitempty"`
		Resolved  bool   `json:"resolved"`
		Replies   int    `json:"replies,omitempty"`
	}

	output := make([]CommentOutput, 0, len(comments))
	for _, c := range comments {
		pos := positions[c.ID]
		replyCount := 0

		if thread, ok := threads[c.ThreadID]; ok {
			replyCount = len(thread.Replies)
		}

		commentOut := CommentOutput{
			ID:        c.ID,
			ThreadID:  c.ThreadID,
			ParentID:  c.ParentID,
			Author:    c.Author,
			Line:      pos.Line,
			Timestamp: c.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Text:      c.Text,
			Type:      c.Type,
			Resolved:  c.Resolved,
		}

		if c.ParentID == "" {
			commentOut.Replies = replyCount
		}

		output = append(output, commentOut)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
