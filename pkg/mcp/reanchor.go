package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// handleReanchor applies agent-declared anchor migrations after a document
// edit. The migration itself lives in pkg/comment so the CLI's `reanchor`
// command runs exactly the same code.
func (s *Server) handleReanchor(ctx context.Context, req *mcp.CallToolRequest, args ReanchorRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		moves := make([]comment.Move, 0, len(args.Moves))
		for _, m := range args.Moves {
			moves = append(moves, comment.Move{
				CommentID: m.CommentID,
				Line:      m.Line,
				Section:   m.Section,
			})
		}
		return map[string]any{"results": comment.ApplyMoves(doc, moves)}, nil
	})
}
