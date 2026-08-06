package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// handleDocumentResource handles requests for document resources
// URI format: comments://doc/{filepath}
func (s *Server) handleDocumentResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract filepath from URI using regex
	// URI format: comments://doc/{filepath}
	re := regexp.MustCompile(`^comments://doc/(.+)$`)
	matches := re.FindStringSubmatch(req.Params.URI)
	if len(matches) < 2 {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	filePath := matches[1]

	// Make path absolute
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, _, err := loadDoc(absPath)
	if err != nil {
		// If sidecar doesn't exist, load just the document
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		doc = &comment.DocumentWithComments{
			Content: string(content),
			Threads: []*comment.Comment{},
		}
	}

	// Serialize the canonical snake_case document view
	jsonData, err := json.MarshalIndent(comment.NewDocumentView(doc), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize document: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}

// handleThreadResource handles requests for specific thread resources
// URI format: comments://thread/{filepath}/{thread_id}
func (s *Server) handleThreadResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract filepath and thread_id from URI
	// URI format: comments://thread/{filepath}/{thread_id}
	re := regexp.MustCompile(`^comments://thread/(.+)/([^/]+)$`)
	matches := re.FindStringSubmatch(req.Params.URI)
	if len(matches) < 3 {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	filePath := matches[1]
	threadID := matches[2]

	// Make path absolute
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, _, err := loadDoc(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Find the thread
	var thread *comment.Comment
	for _, t := range doc.Threads {
		if t.ID == threadID {
			thread = t
			break
		}
	}

	if thread == nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	// Serialize the canonical snake_case thread view (includes all nested replies)
	jsonData, err := json.MarshalIndent(comment.NewCommentView(thread), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize thread: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}
