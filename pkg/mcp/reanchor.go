package mcp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// handleReanchor applies agent-declared anchor migrations after a document edit.
func (s *Server) handleReanchor(ctx context.Context, req *mcp.CallToolRequest, args ReanchorRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	doc, _, err := loadDoc(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load document: %w", err)
	}

	results := make([]map[string]any, 0, len(args.Moves))
	for _, move := range args.Moves {
		c := doc.FindCommentByID(move.CommentID)
		if c == nil {
			results = append(results, map[string]any{
				"comment_id": move.CommentID, "moved": false, "error": "comment not found",
			})
			continue
		}

		line := move.Line
		if move.Section != "" && line == 0 {
			startLine, _, err := comment.ResolveSectionToLines(doc.Content, move.Section, false)
			if err != nil {
				results = append(results, map[string]any{
					"comment_id": move.CommentID, "moved": false, "error": err.Error(),
				})
				continue
			}
			line = startLine
		}
		if line < 1 {
			results = append(results, map[string]any{
				"comment_id": move.CommentID, "moved": false, "error": "move needs a line or section",
			})
			continue
		}

		if c.OriginalLine == 0 {
			c.OriginalLine = c.Line
		}
		c.Line = line
		// Re-capture the anchor at the new position; agent-declared moves are ground truth
		c.Anchor = nil
		c.AnchorConfidence = ""
		comment.UpdateCommentSection(c, doc.Content)
		// A successful migration un-orphans the comment
		if c.IsOrphaned() {
			c.Status = "active"
			c.OrphanedReason = ""
			c.OrphanedAt = nil
		}
		results = append(results, map[string]any{
			"comment_id": move.CommentID, "moved": true, "line": line, "section_path": c.SectionPath,
		})
	}

	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save: %w", err)
	}
	return jsonToolResult(map[string]any{"results": results})
}
