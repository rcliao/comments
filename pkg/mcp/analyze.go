package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

func (s *Server) handleAnalyze(ctx context.Context, req *mcp.CallToolRequest, args AnalyzeRequest) (*mcp.CallToolResult, any, error) {
	return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
		name := args.Template
		if name == "" {
			name = doc.Template
		}
		var template *comment.Template
		var err error
		if name != "" {
			template, err = comment.LoadTemplateForDoc(name, absPath)
			if err != nil {
				return nil, err
			}
		}

		var againstContent, againstPath string
		if args.Against != "" {
			againstPath, err = filepath.Abs(args.Against)
			if err != nil {
				return nil, fmt.Errorf("invalid against path: %w", err)
			}
			data, err := os.ReadFile(againstPath)
			if err != nil {
				return nil, fmt.Errorf("read against document: %w", err)
			}
			againstContent = string(data)
		}

		return comment.AnalyzeDocument(doc.Content, absPath, template, againstContent, againstPath), nil
	})
}
