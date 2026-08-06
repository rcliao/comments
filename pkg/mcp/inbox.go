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
	Since    string `json:"since,omitempty" jsonschema:"Optional RFC3339 timestamp: include unresolved threads with replies newer than this (empty = any replies). Unresolved blocking threads are always included"`
}

// inboxItem is one thread needing attention
type inboxItem struct {
	File      string              `json:"file"`
	Reasons   []string            `json:"reasons"` // new_reply and/or blocking
	Thread    comment.CommentView `json:"thread"`
	LastReply *inboxReply         `json:"last_reply,omitempty"`
}

// inboxReply summarizes the newest reply on a thread
type inboxReply struct {
	Author    string `json:"author"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

// handleInbox returns unresolved threads that have replies newer than `since`
// (or any replies when since is empty), plus all unresolved blocking threads —
// the agent's one-call attention view.
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

	targets, err := comment.FindGateTargets(absPath)
	if err != nil {
		return nil, nil, err
	}
	if len(targets) == 0 {
		return nil, nil, fmt.Errorf("no markdown files with comment sidecars found under %s", absPath)
	}

	items := []inboxItem{}
	for _, file := range targets {
		doc, _, err := loadDoc(file)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		for _, thread := range doc.Threads {
			if thread.Resolved {
				continue
			}
			reasons := []string{}
			last := newestReply(thread)
			if last != nil && (args.Since == "" || last.Timestamp.After(since)) {
				reasons = append(reasons, "new_reply")
			}
			if thread.Blocking {
				reasons = append(reasons, "blocking")
			}
			if len(reasons) == 0 {
				continue
			}
			item := inboxItem{
				File:    file,
				Reasons: reasons,
				Thread:  comment.NewCommentView(thread),
			}
			if last != nil {
				item.LastReply = &inboxReply{
					Author:    last.Author,
					Text:      last.Text,
					Timestamp: last.Timestamp.Format(time.RFC3339),
				}
			}
			items = append(items, item)
		}
	}

	result := map[string]any{
		"since": args.Since,
		"count": len(items),
		"items": items,
	}
	return jsonToolResult(result)
}

// newestReply returns the reply with the newest timestamp anywhere in the
// thread's nested reply tree, or nil if the thread has no replies.
func newestReply(thread *comment.Comment) *comment.Comment {
	var newest *comment.Comment
	var walk func(replies []*comment.Comment)
	walk = func(replies []*comment.Comment) {
		for _, r := range replies {
			if newest == nil || r.Timestamp.After(newest.Timestamp) {
				newest = r
			}
			walk(r.Replies)
		}
	}
	walk(thread.Replies)
	return newest
}
