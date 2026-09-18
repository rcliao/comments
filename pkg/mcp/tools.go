package mcp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// loadDoc loads a document plus its comments and persists any re-anchoring or
// orphan-status migrations discovered during the load — the write half of what
// comment.LoadFromSidecar used to do internally. MCP surfaces load state via
// the returned report (e.g. staleness in comments_status) rather than printing.
func loadDoc(absPath string) (*comment.DocumentWithComments, *comment.LoadReport, error) {
	return comment.LoadDocument(absPath)
}

// withDoc is the shared prelude for single-document tool handlers: resolve the
// path, load the document, run fn, and serialize fn's payload as the tool
// result. fn must not persist the document — use withDocSave for mutations.
func withDoc(path string, fn func(absPath string, doc *comment.DocumentWithComments, report *comment.LoadReport) (any, error)) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	doc, report, err := loadDoc(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}
	result, err := fn(absPath, doc, report)
	if err != nil {
		return nil, nil, err
	}
	return jsonToolResult(result)
}

// withDocSave is withDoc for mutating handlers: when fn succeeds, the document
// is saved back to the sidecar (SaveToSidecar also writes the markdown
// atomically if fn changed doc.Content) before the result is returned.
func withDocSave(path string, fn func(absPath string, doc *comment.DocumentWithComments) (any, error)) (*mcp.CallToolResult, any, error) {
	return withDoc(path, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
		result, err := fn(absPath, doc)
		if err != nil {
			return nil, err
		}
		if err := comment.SaveToSidecar(absPath, doc); err != nil {
			return nil, fmt.Errorf("failed to save comments: %w", err)
		}
		return result, nil
	})
}

// handleGet is the lookup path: one comment with its context, or — with no
// comment_id — every thread in the document, resolved ones included. The
// worklist is comments_inbox; this answers "what did that thread say".
func (s *Server) handleGet(ctx context.Context, req *mcp.CallToolRequest, args GetRequest) (*mcp.CallToolResult, any, error) {
	// Citation-literal form: cite carries the syntax exactly as it appears in
	// a document; relative paths resolve against the CITING doc (from)
	if args.Cite != "" {
		path, id, err := comment.ResolveThreadCitation(args.Cite, args.From)
		if err != nil {
			return nil, nil, err
		}
		args.FilePath, args.CommentID = path, id
	}
	return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
		if args.CommentID == "" {
			return comment.ListThreads(doc, absPath, args.UnresolvedOnly), nil
		}
		return comment.GetThread(doc, args.CommentID, 5)
	})
}

// handleAdd adds one or many root comments; a single comment is a batch of one.
func (s *Server) handleAdd(ctx context.Context, req *mcp.CallToolRequest, args AddRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		return comment.AddComments(doc, args.Comments)
	})
}

// handleReply replies to — and optionally resolves — one or many threads. An
// MCP caller is always an agent by construction, so the zone guard applies.
func (s *Server) handleReply(ctx context.Context, req *mcp.CallToolRequest, args ReplyRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		return comment.ReplyToThreads(doc, absPath, args.Replies, comment.ActorAgent)
	})
}

func (s *Server) handleSuggest(ctx context.Context, req *mcp.CallToolRequest, args SuggestRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		return comment.AddSuggestion(doc, comment.SuggestionSpec{
			Author: args.Author, Text: args.Text, StartLine: args.StartLine, EndLine: args.EndLine,
			Section: args.Section, Anchor: args.Anchor, OriginalText: args.OriginalText, ProposedText: args.ProposedText,
		})
	})
}
