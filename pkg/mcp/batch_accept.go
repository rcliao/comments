package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// BatchAcceptRequest accepts several suggestions in one call, either by an
// explicit ID list or by matching pending suggestions on author/type.
type BatchAcceptRequest struct {
	FilePath      string   `json:"filepath" jsonschema:"Path to the markdown file"`
	SuggestionIDs []string `json:"suggestion_ids,omitempty" jsonschema:"Explicit suggestion IDs to accept"`
	Author        string   `json:"author,omitempty" jsonschema:"Accept all pending suggestions from this author"`
	Type          string   `json:"type,omitempty" jsonschema:"Accept all pending suggestions of this type (Q, S, B, T, E)"`
}

// handleBatchAccept mirrors the CLI's batch-accept, using the same selection
// and application logic in pkg/comment.
func (s *Server) handleBatchAccept(ctx context.Context, req *mcp.CallToolRequest, args BatchAcceptRequest) (*mcp.CallToolResult, any, error) {
	if len(args.SuggestionIDs) == 0 && args.Author == "" && args.Type == "" {
		return nil, nil, fmt.Errorf("one of suggestion_ids, author or type is required")
	}

	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		ids := args.SuggestionIDs
		if len(ids) == 0 {
			ids = comment.SelectPendingSuggestions(doc, args.Author, args.Type)
		}
		if len(ids) == 0 {
			return map[string]any{
				"accepted_count": 0,
				"results":        []comment.AcceptResult{},
				"message":        "no matching pending suggestions",
			}, nil
		}

		results := comment.AcceptSuggestions(doc, ids)
		accepted := 0
		for _, r := range results {
			if r.Accepted {
				accepted++
			}
		}

		// Accept is the one operation that edits the markdown; withDocSave
		// persists the sidecar but the content needs its own write.
		if accepted > 0 {
			if err := comment.SaveDocumentContent(absPath, doc); err != nil {
				return nil, fmt.Errorf("failed to write document: %w", err)
			}
		}

		return map[string]any{
			"accepted_count": accepted,
			"failed_count":   len(results) - accepted,
			"results":        results,
		}, nil
	})
}
