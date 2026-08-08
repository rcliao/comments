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

// registerTools registers all MCP tools (operations)
func (s *Server) registerTools() {
	// Read operations
	s.registerListTool()
	s.registerGetTool()
	s.registerStatusTool()

	// Write operations
	s.registerAddTool()
	s.registerReplyTool()
	s.registerResolveTool()

	// Suggestion operations
	s.registerSuggestTool()
	s.registerAcceptTool()
	s.registerBatchAcceptTool()
	s.registerRejectTool()

	// Batch operations
	s.registerBatchAddTool()
	s.registerBatchReplyTool()

	// Review gate operations
	s.registerGateTool()
	s.registerRequestReviewTool()
	s.registerCheckReviewTool()

	// Agent inbox
	s.registerInboxTool()

	// Template operations
	s.registerTemplateTools()

	// Anchor migration
	s.toolNames = append(s.toolNames, "comments_reanchor")
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_reanchor",
		Description: "After editing a document that has comments, migrate the anchors your edits displaced (batch comment_id -> new line/section). The editing agent knows the mapping; call this as a required post-edit step.",
	}, s.handleReanchor)
}

func (s *Server) registerTemplateTools() {
	s.toolNames = append(s.toolNames,
		"comments_get_template", "comments_validate", "comments_seed")

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_get_template",
		Description: "Read a document template (section structure, length budgets, zones, review criteria) to use as a writing brief before drafting; call without a name to list available templates",
	}, s.handleGetTemplate)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_validate",
		Description: "Validate a document's structure against a template (required sections, order, length caps, unresolved ambiguity markers); fix violations before requesting human review",
	}, s.handleValidate)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "comments_seed",
		Description: "Materialize a template's per-section review criteria and NEEDS CLARIFICATION markers as anchored comment threads, and record the template on the document",
	}, s.handleSeed)
}

func (s *Server) registerGateTool() {
	tool := &mcp.Tool{
		Name:        "comments_gate",
		Description: "Evaluate the review gate for a file or directory: approved when no unresolved blocking comments remain (strict mode fails on any unresolved item)",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleGate)
}

func (s *Server) registerRequestReviewTool() {
	tool := &mcp.Tool{
		Name:        "comments_request_review",
		Description: "Request human review of a document. Default: block until the reviewer signs off or the timeout elapses, then return the review (decision + the reviewer's note) and the remaining comments. A signoff is recorded either by the human's TUI review ('comments view' -> q -> approve/request changes) or by 'comments signoff' — ask for ONE of them, not both. With blocking=false: return immediately with a durable since handle for comments_check_review polling. Without MCP an agent can wait the same way with 'comments watch <file> --until signoff'",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleRequestReview)
}

func (s *Server) registerCheckReviewTool() {
	tool := &mcp.Tool{
		Name:        "comments_check_review",
		Description: "Check a pending review handle: given the since timestamp from a non-blocking comments_request_review, returns pending or review_completed (the review — decision, author and the reviewer's note — plus gate state) by comparing the sidecar's newest signoff against since. Sees signoffs from the TUI verdict and from 'comments signoff' alike",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleCheckReview)
}

func (s *Server) registerInboxTool() {
	tool := &mcp.Tool{
		Name:        "comments_inbox",
		Description: "Agent inbox for a file or directory: unresolved threads with replies newer than since (or any replies when since is empty) plus all unresolved blocking threads, each with its last reply — everything needing attention in one call",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleInbox)
}

func (s *Server) registerListTool() {
	tool := &mcp.Tool{
		Name:        "comments_list",
		Description: "List and filter comments in a document",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleListComments)
}

func (s *Server) registerGetTool() {
	tool := &mcp.Tool{
		Name:        "comments_get",
		Description: "Get a specific comment with full context",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleGetComment)
}

func (s *Server) registerStatusTool() {
	tool := &mcp.Tool{
		Name:        "comments_status",
		Description: "Get document status including pending suggestions and orphaned comments",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleStatus)
}

func (s *Server) registerAddTool() {
	tool := &mcp.Tool{
		Name:        "comments_add",
		Description: "Add a root comment to a document (specify line or section)",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleAddComment)
}

func (s *Server) registerReplyTool() {
	tool := &mcp.Tool{
		Name:        "comments_reply",
		Description: "Reply to an existing comment thread",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleReply)
}

func (s *Server) registerResolveTool() {
	tool := &mcp.Tool{
		Name:        "comments_resolve",
		Description: "Mark a comment thread as resolved or unresolved",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleResolve)
}

func (s *Server) registerSuggestTool() {
	tool := &mcp.Tool{
		Name:        "comments_suggest",
		Description: "Create a multi-line edit suggestion (track-changes style)",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleSuggest)
}

func (s *Server) registerAcceptTool() {
	tool := &mcp.Tool{
		Name:        "comments_accept",
		Description: "Accept an edit suggestion (applies the change to the document)",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleAccept)
}

func (s *Server) registerBatchAcceptTool() {
	tool := &mcp.Tool{
		Name:        "comments_batch_accept",
		Description: "Accept several edit suggestions at once, by explicit IDs or by matching pending suggestions on author/type",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleBatchAccept)
}

func (s *Server) registerRejectTool() {
	tool := &mcp.Tool{
		Name:        "comments_reject",
		Description: "Reject an edit suggestion",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleReject)
}

func (s *Server) registerBatchAddTool() {
	tool := &mcp.Tool{
		Name:        "comments_batch_add",
		Description: "Add multiple root comments in a single operation",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleBatchAdd)
}

func (s *Server) registerBatchReplyTool() {
	tool := &mcp.Tool{
		Name:        "comments_batch_reply",
		Description: "Add multiple replies to threads in a single operation",
	}
	s.toolNames = append(s.toolNames, tool.Name)
	mcp.AddTool(s.mcp, tool, s.handleBatchReply)
}
