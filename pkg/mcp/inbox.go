package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// InboxRequest asks "what needs my attention" across a file or directory.
type InboxRequest struct {
	FilePath string `json:"filepath" jsonschema:"Path to a markdown file or a directory of markdown files"`
	Since    string `json:"since,omitempty" jsonschema:"Optional RFC3339 timestamp: replies and threads newer than this are flagged new_reply / new_thread. Never hides a thread"`
	Reviewer string `json:"reviewer,omitempty" jsonschema:"Whose last verdict to diff changed lines against (default: the latest reviewer)"`
}

// handleInbox is the MCP wrapper over comment.BuildInbox; the CLI's `inbox`
// command calls the same builder.
func (s *Server) handleInbox(ctx context.Context, req *mcp.CallToolRequest, args InboxRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	var since time.Time
	if args.Since != "" {
		since, err = time.Parse(time.RFC3339, args.Since)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid since timestamp (want RFC3339): %w", err)
		}
	}

	inbox, err := comment.BuildInbox(absPath, comment.InboxOptions{Since: since, Reviewer: args.Reviewer, ContextSize: 2})
	if err != nil {
		return nil, nil, err
	}
	return jsonToolResult(inbox)
}
