package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// resolveDocTemplate picks the template from the explicit argument,
// frontmatter, legacy sidecar, or an unambiguous bundle collection.
func resolveDocTemplate(doc *comment.DocumentWithComments, name, docPath string) (*comment.Template, error) {
	template, _, err := comment.ResolveTemplateForDocument(docPath, doc.Content, name, doc.Template)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("no template specified; use the template argument, comments.template frontmatter, or a bundle collection with one template")
	}
	return template, nil
}

// handleValidate checks document structure against a template so the agent can
// self-correct before requesting human review.
func (s *Server) handleValidate(ctx context.Context, req *mcp.CallToolRequest, args ValidateRequest) (*mcp.CallToolResult, any, error) {
	return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
		t, err := resolveDocTemplate(doc, args.Template, absPath)
		if err != nil {
			return nil, err
		}

		violations := comment.ValidateManagedDocument(doc.Content, absPath, t)
		return map[string]any{
			"template":      t.Name,
			"conforms":      len(violations) == 0,
			"violations":    violations,
			"section_words": comment.SectionWordReport(doc.Content, t),
		}, nil
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
