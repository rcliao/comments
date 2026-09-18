package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// defaultWatchTimeout bounds a wait: an MCP client has a per-call ceiling, so a
// watch must come back on its own and let the agent call again.
const defaultWatchTimeout = 10 * time.Minute

// WatchRequest waits for review-state changes on a file or directory.
type WatchRequest struct {
	FilePath       string `json:"filepath" jsonschema:"Path to a markdown file or a directory of markdown files"`
	Until          string `json:"until,omitempty" jsonschema:"Comma-separated event types that end the wait (default: signoff). Others: gate_changed, comment_added, reply_added"`
	Since          string `json:"since,omitempty" jsonschema:"RFC3339 time you handed off to the human. A verdict recorded after it is returned immediately, so a fast reviewer is never missed. Always pass it"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"Give up after this many seconds (default 600) and return status timeout — call again to keep waiting"`
}

// WatchResult is what a wait observed. Events are the same records
// `comments watch` prints as NDJSON, in order; the last one matched until.
type WatchResult struct {
	Status string               `json:"status"` // matched or timeout
	Events []comment.WatchEvent `json:"events"`
}

// handleWatch is the MCP face of `comments watch --until`: both call
// comment.Watch, so an agent waits on a human review the same way on either
// surface and receives the same signoff event (author, decision, note).
func (s *Server) handleWatch(ctx context.Context, req *mcp.CallToolRequest, args WatchRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	until := args.Until
	if until == "" {
		until = "signoff"
	}
	timeout := defaultWatchTimeout
	if args.TimeoutSeconds > 0 {
		timeout = time.Duration(args.TimeoutSeconds) * time.Second
	}
	var since time.Time
	if args.Since != "" {
		if since, err = time.Parse(time.RFC3339, args.Since); err != nil {
			return nil, nil, fmt.Errorf("invalid since timestamp (want RFC3339): %w", err)
		}
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := WatchResult{Status: "matched", Events: []comment.WatchEvent{}}
	err = comment.Watch(waitCtx, absPath, comment.WatchOptions{Interval: watchPollInterval, Until: until, Since: since}, func(e comment.WatchEvent) bool {
		result.Events = append(result.Events, e)
		return false
	})
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		result.Status = "timeout"
	case err != nil:
		return nil, nil, err
	}
	return jsonToolResult(result)
}

// watchPollInterval is a var so tests can shorten it.
var watchPollInterval = time.Second

func jsonToolResult(v any) (*mcp.CallToolResult, any, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}
