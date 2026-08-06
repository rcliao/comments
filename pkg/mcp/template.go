package mcp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// resolveDocTemplate picks the template: explicit name wins, else sidecar record
func resolveDocTemplate(doc *comment.DocumentWithComments, name string) (*comment.Template, error) {
	if name == "" {
		name = doc.Template
	}
	if name == "" {
		return nil, fmt.Errorf("no template specified and none recorded in sidecar; call comments_get_template with no name to list available templates")
	}
	return comment.LoadTemplate(name)
}

// handleValidate checks document structure against a template so the agent can
// self-correct before requesting human review.
func (s *Server) handleValidate(ctx context.Context, req *mcp.CallToolRequest, args ValidateRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	doc, _, err := loadDoc(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load document: %w", err)
	}
	t, err := resolveDocTemplate(doc, args.Template)
	if err != nil {
		return nil, nil, err
	}

	violations := comment.ValidateTemplate(doc.Content, t)
	return jsonToolResult(map[string]any{
		"template":   t.Name,
		"conforms":   len(violations) == 0,
		"violations": violations,
	})
}

// handleSeed materializes the template's review criteria and ambiguity markers
// as anchored comment threads, and records the template on the document.
func (s *Server) handleSeed(ctx context.Context, req *mcp.CallToolRequest, args SeedRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	doc, _, err := loadDoc(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load document: %w", err)
	}
	t, err := resolveDocTemplate(doc, args.Template)
	if err != nil {
		return nil, nil, err
	}

	author := args.Author
	if author == "" {
		author = "template"
	}
	added := comment.SeedTemplateThreads(doc, t, author, args.MarkersOnly)
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save: %w", err)
	}

	seeded := make([]map[string]any, 0, len(added))
	for _, c := range added {
		seeded = append(seeded, map[string]any{
			"id":       c.ID,
			"line":     c.Line,
			"text":     c.Text,
			"blocking": c.Blocking,
		})
	}
	return jsonToolResult(map[string]any{
		"template":     t.Name,
		"seeded_count": len(added),
		"seeded":       seeded,
	})
}

// handleGetTemplate returns a template definition (the agent's writing brief),
// or lists available templates when no name is given.
func (s *Server) handleGetTemplate(ctx context.Context, req *mcp.CallToolRequest, args GetTemplateRequest) (*mcp.CallToolResult, any, error) {
	if args.Name == "" {
		templates, err := comment.ListTemplates()
		if err != nil {
			return nil, nil, err
		}
		return jsonToolResult(map[string]any{"templates": templates})
	}

	t, err := comment.LoadTemplate(args.Name)
	if err != nil {
		return nil, nil, err
	}
	return jsonToolResult(t)
}
