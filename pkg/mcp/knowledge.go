package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

func (s *Server) registerKnowledgeTools() {
	s.toolNames = append(s.toolNames, "comments_new", "comments_context", "comments_bundle_index")
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_new",
		Description: "Create a template-guided OKF concept in the bundle collection assigned to that template, including frontmatter, a review sidecar, and refreshed indexes",
	}, s.handleNewDocument)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_context",
		Description: "Load an explainable OKF neighborhood for an agent role: explicit relations, links, backlinks, sources, review state, and optional bodies or threads. coverage-scout is draft-blind",
	}, s.handleContext)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_bundle_index",
		Description: "Regenerate the OKF root and collection indexes for the bundle discovered from a path",
	}, s.handleBundleIndex)
}

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

func (s *Server) handleBundleIndex(ctx context.Context, req *mcp.CallToolRequest, args BundleIndexRequest) (*mcp.CallToolResult, any, error) {
	bundle, err := comment.FindBundle(args.Path)
	if err != nil {
		return nil, nil, err
	}
	if err := comment.WriteBundleIndexes(bundle); err != nil {
		return nil, nil, err
	}
	return jsonToolResult(map[string]any{"bundle": bundle.Config.Bundle, "root": bundle.RootPath, "indexed": true})
}
