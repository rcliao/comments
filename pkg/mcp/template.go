package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// resolveDocTemplate picks the template: explicit name wins, else sidecar
// record. Resolution is doc-relative so project templates are found no matter
// where the MCP server was launched from.
func resolveDocTemplate(doc *comment.DocumentWithComments, name, docPath string) (*comment.Template, error) {
	if name == "" {
		name = doc.Template
	}
	if name == "" {
		return nil, fmt.Errorf("no template specified and none recorded in sidecar; call comments_get_template with no name to list available templates")
	}
	return comment.LoadTemplateForDoc(name, docPath)
}

// handleValidate checks document structure against a template so the agent can
// self-correct before requesting human review.
func (s *Server) handleValidate(ctx context.Context, req *mcp.CallToolRequest, args ValidateRequest) (*mcp.CallToolResult, any, error) {
	return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
		t, err := resolveDocTemplate(doc, args.Template, absPath)
		if err != nil {
			return nil, err
		}

		violations := comment.ValidateTemplate(doc.Content, t)
		return map[string]any{
			"template":      t.Name,
			"conforms":      len(violations) == 0,
			"violations":    violations,
			"section_words": comment.SectionWordReport(doc.Content, t),
		}, nil
	})
}

// handleSeed materializes the template's review criteria and ambiguity markers
// as anchored comment threads, and records the template on the document.
func (s *Server) handleSeed(ctx context.Context, req *mcp.CallToolRequest, args SeedRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		t, err := resolveDocTemplate(doc, args.Template, absPath)
		if err != nil {
			return nil, err
		}

		author := args.Author
		if author == "" {
			author = "template"
		}
		added := comment.SeedTemplateThreads(doc, t, author, args.MarkersOnly)

		seeded := make([]map[string]any, 0, len(added))
		for _, c := range added {
			seeded = append(seeded, map[string]any{
				"id":       c.ID,
				"line":     c.Line,
				"text":     c.Text,
				"blocking": c.Blocking,
			})
		}
		return map[string]any{
			// Leads the response: recording the template is a distinct act
			// from seeding threads — an empty seeded list does NOT mean
			// nothing happened; the gate now enforces this template's
			// structure (docs/plan-agent-surface.md Phase 4)
			"template_recorded": t.Name,
			"gate_enforces":     "document structure now validates against this template on every gate",
			"seeded_count":      len(added),
			"seeded":            seeded,
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
