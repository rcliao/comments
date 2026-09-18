package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const ServerName = "comments-mcp-server"

// ServerVersion is what the MCP server advertises in its handshake. main sets
// it to the binary's stamped version at startup so the two cannot disagree —
// a stale server version is exactly what `comments doctor` exists to catch.
var ServerVersion = "dev"

// Server wraps the MCP server and provides comment-specific functionality
type Server struct {
	mcp *mcp.Server
	// toolNames records every registered tool so the startup banner and any
	// other consumer read from registration itself rather than a parallel
	// hand-maintained list that silently goes stale.
	toolNames []string
}

// ToolNames returns the names of all registered tools, in registration order.
func (s *Server) ToolNames() []string { return s.toolNames }

// NewServer creates a new MCP server for the comments tool
func NewServer() *Server {
	impl := &mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}

	mcpServer := mcp.NewServer(impl, nil)

	s := &Server{
		mcp: mcpServer,
	}

	// Register resources
	s.registerResources()

	// Register tools
	s.registerTools()

	return s
}

// Serve starts the MCP server using stdio transport
func (s *Server) Serve(ctx context.Context) error {
	log.Println("Starting MCP server...")
	transport := &mcp.StdioTransport{}
	if err := s.mcp.Run(ctx, transport); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// registerResources registers all MCP resources (documents and threads)
func (s *Server) registerResources() {
	// Register document resource with URI template
	docResource := &mcp.ResourceTemplate{
		URITemplate: "comments://doc/{filepath}",
		Name:        "Document with Comments",
		Description: "Access a document with all its comments and threads",
		MIMEType:    "application/json",
	}
	s.mcp.AddResourceTemplate(docResource, s.handleDocumentResource)

	// Register thread resource with URI template
	threadResource := &mcp.ResourceTemplate{
		URITemplate: "comments://thread/{filepath}/{thread_id}",
		Name:        "Comment Thread",
		Description: "Access a specific comment thread with full context",
		MIMEType:    "application/json",
	}
	s.mcp.AddResourceTemplate(threadResource, s.handleThreadResource)
}

// register adds one tool and records its name, so the startup banner, doctor
// and the CLI-parity test all read the catalog from registration instead of a
// hand-maintained list (those have gone stale here every time).
func register[In any](s *Server, name, description string, handler mcp.ToolHandlerFor[In, any]) {
	s.toolNames = append(s.toolNames, name)
	mcp.AddTool(s.mcp, &mcp.Tool{Name: name, Description: description}, handler)
}

// registerTools registers the agent surface: one tool per purpose, each the
// twin of a CLI command of the same name. Human decisions (accepting a
// suggestion, the verdict) are deliberately absent — they happen in
// `comments view` / `comments serve`, so no agent-reachable path can make them.
func (s *Server) registerTools() {
	// Create a document under a template
	register(s, "comments_new",
		"Create a template-guided OKF concept with frontmatter, review sidecar, and refreshed indexes. If the project has no bundle config, initialize the standard docs/artifacts bundle automatically",
		s.handleNewDocument)
	register(s, "comments_context",
		"Load what you need before writing or reviewing a document: its template as a writing brief (sections, word budgets, zones, review criteria, reading path) plus an explainable OKF neighborhood — explicit relations, links, backlinks, sources, review state, optional bodies or threads. Modes: drafting (default), review, coverage-scout (draft-blind), evidence-verifier, human-review, implementation (plan ledger)",
		s.handleContext)
	register(s, "comments_validate",
		"Validate a document's structure against its template (required sections, order, length caps, unresolved ambiguity markers); fix violations before asking for human review",
		s.handleValidate)
	register(s, "comments_analyze",
		"Advisory artifact analysis: expose numbered-question coverage, citation violations, and—with against—research findings cited, explicitly excluded, or uncovered by a plan. ready=false never changes gate state",
		s.handleAnalyze)

	// Annotate the draft
	register(s, "comments_add",
		"Add one or many root comments, each placed by anchor (quote the target line — preferred), section path, or line. Atomic. Mark must-fix items blocking",
		s.handleAdd)

	// Iterate with the human
	register(s, "comments_watch",
		"Wait for the human: blocks until a review event (default: signoff) and returns it with the reviewer's decision and note. Tell the human to review with `comments view <file>`, then call this. Returns status timeout after timeout_seconds — call again to keep waiting",
		s.handleWatch)
	register(s, "comments_inbox",
		"The one read while iterating: the gate decision, every unresolved thread (blocking first, with replies and document context), pending suggestions, template violations, orphaned anchors, and lines changed since the reviewer's last verdict. Done means decision approved AND no items",
		s.handleInbox)
	register(s, "comments_get",
		"Look up one comment with context, or every thread in a document (resolved included) when comment_id is omitted. Also resolves a thread: citation from another document",
		s.handleGet)
	register(s, "comments_reply",
		"Reply to one or many threads; set resolve to also close a thread once your fix is applied and explained. Resolving is refused in a template's zone: human section — reply and leave it to the human",
		s.handleReply)
	register(s, "comments_suggest",
		"Propose an edit for the human to accept or reject in `comments view` (agents cannot accept). Use when a fix is a judgment call rather than editing the document directly",
		s.handleSuggest)
	register(s, "comments_reanchor",
		"After editing a document that has comments, migrate the anchors your edits displaced (batch comment_id -> new line/section). The editing agent knows the mapping; call this as a required post-edit step",
		s.handleReanchor)
}
