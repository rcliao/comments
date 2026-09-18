package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

func (s *Server) handleNewDocument(ctx context.Context, req *mcp.CallToolRequest, args NewDocumentRequest) (*mcp.CallToolResult, any, error) {
	result, err := comment.CreateBundleDocument(comment.NewDocumentOptions{
		Name: args.Name, Template: args.Template, Title: args.Title,
		Description: args.Description, From: args.From, StartDir: args.BundlePath,
	})
	if err != nil {
		return nil, nil, err
	}
	return jsonToolResult(result)
}

func (s *Server) handleContext(ctx context.Context, req *mcp.CallToolRequest, args ContextRequest) (*mcp.CallToolResult, any, error) {
	result, err := comment.BuildDocumentContext(args.FilePath, comment.ContextOptions{
		For: args.For, IncludeBody: args.IncludeBody, IncludeThreads: args.IncludeThreads,
	})
	if err != nil {
		return nil, nil, err
	}
	return jsonToolResult(result)
}
