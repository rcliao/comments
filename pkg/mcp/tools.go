package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// findSuggestion locates a pending-or-decided suggestion by ID
func findSuggestion(doc *comment.DocumentWithComments, id string) (*comment.Comment, error) {
	c := doc.FindCommentByID(id)
	if c == nil || !c.IsSuggestion {
		return nil, fmt.Errorf("suggestion not found: %s", id)
	}
	return c, nil
}

// Read tools

func (s *Server) handleListComments(ctx context.Context, req *mcp.CallToolRequest, args ListCommentsRequest) (*mcp.CallToolResult, any, error) {
	return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
		// List returns thread ROOTS with replies nested (the CLI JSON shape;
		// flattening roots and replies into one list handed out reply IDs the
		// write path rejected and buried reply text — docs/plan-agent-surface.md
		// Phase 2). Filters match the root OR any of its replies; the ROOT is
		// what gets returned either way.
		var roots []*comment.Comment
		if args.Section != "" {
			seen := map[string]bool{}
			for _, c := range comment.GetCommentsInSection(doc, args.Section) {
				if root := comment.FindThreadContaining(doc.Threads, c.ID); root != nil && !seen[root.ID] {
					seen[root.ID] = true
					roots = append(roots, root)
				}
			}
		} else {
			roots = doc.Threads
		}

		matches := func(c *comment.Comment) bool {
			if args.Author != "" && c.Author != args.Author {
				return false
			}
			if args.Type != "" && c.Type != args.Type {
				return false
			}
			if args.Search != "" && !strings.Contains(strings.ToLower(c.Text), strings.ToLower(args.Search)) {
				return false
			}
			if args.Status != "" && c.Status != args.Status {
				return false
			}
			if args.Priority != "" && c.Priority != args.Priority {
				return false
			}
			return true
		}
		var anyInThread func(c *comment.Comment) bool
		anyInThread = func(c *comment.Comment) bool {
			if matches(c) {
				return true
			}
			for _, r := range c.Replies {
				if anyInThread(r) {
					return true
				}
			}
			return false
		}

		filtered := make([]*comment.Comment, 0)
		for _, c := range roots {
			// Content filters match the root or any reply; positional and
			// resolution filters are properties of the thread root itself
			if !anyInThread(c) {
				continue
			}
			if args.Resolved != nil && c.Resolved != *args.Resolved {
				continue
			}
			if args.LineStart > 0 && c.Line < args.LineStart {
				continue
			}
			if args.LineEnd > 0 && c.Line > args.LineEnd {
				continue
			}
			filtered = append(filtered, c)
		}

		// Build response (snake_case wire shape, unified with CLI JSON)
		result := map[string]any{
			"filepath": absPath,
			"total":    len(filtered),
			"comments": comment.NewCommentViews(filtered),
		}

		// Add context if requested
		if args.WithContext {
			content, _ := os.ReadFile(absPath)
			contextComments := make([]CommentWithContext, len(filtered))
			for i, c := range filtered {
				contextComments[i] = CommentWithContext{
					Comment:      comment.NewCommentView(c),
					SectionPath:  c.SectionPath,
					ContextLines: getContextLines(string(content), c.Line, 3),
					IsOrphaned:   c.Status == "orphaned",
				}
			}
			result["comments_with_context"] = contextComments
		}

		return result, nil
	})
}

func (s *Server) handleGetComment(ctx context.Context, req *mcp.CallToolRequest, args GetCommentRequest) (*mcp.CallToolResult, any, error) {
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
		foundComment := doc.FindCommentByID(args.CommentID)
		if foundComment == nil {
			return nil, fmt.Errorf("comment not found: %s", args.CommentID)
		}

		// Load document content for context
		content, _ := os.ReadFile(absPath)

		return CommentWithContext{
			Comment:      comment.NewCommentView(foundComment),
			SectionPath:  foundComment.SectionPath,
			ContextLines: getContextLines(string(content), foundComment.Line, 5),
			IsOrphaned:   foundComment.Status == "orphaned",
		}, nil
	})
}

func (s *Server) handleStatus(ctx context.Context, req *mcp.CallToolRequest, args StatusRequest) (*mcp.CallToolResult, any, error) {
	// The load report tells us whether the markdown content changed since the
	// sidecar was last written (staleness); loadDoc persists the revalidated
	// sidecar, so a subsequent status is clean.
	return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, report *comment.LoadReport) (any, error) {
		// Calculate statistics: thread counts over root threads only,
		// comment-level counts over the flattened tree (roots + replies)
		totalThreads := len(doc.Threads)
		resolvedThreads := 0
		unresolvedThreads := 0
		for _, t := range doc.Threads {
			if t.Resolved {
				resolvedThreads++
			} else {
				unresolvedThreads++
			}
		}

		pendingSuggestions := 0
		orphanedComments := 0
		suggestionsByAuthor := make(map[string]int)

		allComments := doc.GetAllComments()
		for _, c := range allComments {
			if c.Status == "orphaned" {
				orphanedComments++
			}

			if c.IsSuggestion && c.Accepted == nil {
				pendingSuggestions++
				suggestionsByAuthor[c.Author]++
			}
		}

		return DocumentStatus{
			FilePath:            absPath,
			TotalThreads:        totalThreads,
			TotalComments:       len(allComments),
			ResolvedThreads:     resolvedThreads,
			UnresolvedThreads:   unresolvedThreads,
			PendingSuggestions:  pendingSuggestions,
			OrphanedComments:    orphanedComments,
			IsStale:             report.Stale,
			DocumentHash:        doc.DocumentHash,
			LastValidated:       doc.LastValidated.Format("2006-01-02T15:04:05Z"),
			SuggestionsByAuthor: suggestionsByAuthor,
		}, nil
	})
}

// Write tools

func (s *Server) handleAddComment(ctx context.Context, req *mcp.CallToolRequest, args AddCommentRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		// Determine line number
		line := args.Line
		if args.Section != "" && line == 0 {
			if err := comment.ValidateSectionPath(doc.Content, args.Section); err != nil {
				return nil, err
			}
			startLine, _, err := comment.ResolveSectionToLines(doc.Content, args.Section, false)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve section: %w", err)
			}
			line = startLine
		}
		if args.Anchor != "" && line == 0 {
			resolved, err := comment.ResolveAnchorText(doc.Content, args.Anchor)
			if err != nil {
				return nil, err
			}
			line = resolved
		}

		// Create new comment (text carries the normalized type marker, same
		// shape as CLI/TUI adds — PrefixType never doubles one the author
		// already wrote)
		newComment := comment.NewComment(args.Author, line, comment.PrefixType(args.Text, args.Type))
		if args.Type != "" {
			newComment.Type = args.Type
		}
		if args.Status != "" {
			newComment.Status = args.Status
		}
		if args.Priority != "" {
			newComment.Priority = args.Priority
		}
		newComment.Blocking = args.Blocking
		comment.UpdateCommentSection(newComment, doc.Content)

		// Add to document
		doc.Threads = append(doc.Threads, newComment)

		return map[string]any{
			"success":    true,
			"comment_id": newComment.ID,
			"message":    fmt.Sprintf("Added comment %s at line %d", newComment.ID, line),
		}, nil
	})
}

func (s *Server) handleReply(ctx context.Context, req *mcp.CallToolRequest, args ReplyRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		if err := comment.AddReplyToThread(doc.Threads, args.ThreadID, args.Author, args.Text); err != nil {
			return nil, fmt.Errorf("failed to add reply: %w", err)
		}

		return map[string]any{
			"success": true,
			"message": fmt.Sprintf("Added reply to thread %s", args.ThreadID),
		}, nil
	})
}

func (s *Server) handleResolve(ctx context.Context, req *mcp.CallToolRequest, args ResolveRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		// Zone enforcement: threads in a template's human-decision zone are the
		// human's to close. An MCP caller is always an agent by construction.
		if !args.Unresolve {
			if err := comment.GuardZoneResolve(doc, absPath, args.ThreadID, comment.ActorAgent); err != nil {
				return nil, err
			}
		}

		// Resolve or unresolve thread
		var actionErr error
		if args.Unresolve {
			actionErr = comment.UnresolveThread(doc.Threads, args.ThreadID)
		} else {
			actionErr = comment.ResolveThread(doc.Threads, args.ThreadID)
		}

		if actionErr != nil {
			return nil, fmt.Errorf("failed to resolve thread: %w", actionErr)
		}

		action := "resolved"
		if args.Unresolve {
			action = "unresolved"
		}
		return map[string]any{
			"success": true,
			"message": fmt.Sprintf("Thread %s %s", args.ThreadID, action),
		}, nil
	})
}

// Suggestion tools

func (s *Server) handleSuggest(ctx context.Context, req *mcp.CallToolRequest, args SuggestRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		start, end := args.StartLine, args.EndLine
		if args.Anchor != "" && start == 0 {
			resolved, err := comment.ResolveAnchorText(doc.Content, args.Anchor)
			if err != nil {
				return nil, err
			}
			start = resolved
			// original_text's line count defines the range; single line otherwise
			end = resolved
			if args.OriginalText != "" {
				end = resolved + strings.Count(strings.TrimRight(args.OriginalText, "\n"), "\n")
			}
		}
		if end == 0 {
			end = start
		}
		suggestion := comment.NewSuggestion(
			args.Author,
			start,
			end,
			args.Text,
			args.OriginalText,
			args.ProposedText,
		)

		doc.Threads = append(doc.Threads, suggestion)

		return map[string]any{
			"success":       true,
			"suggestion_id": suggestion.ID,
			"message":       fmt.Sprintf("Created suggestion %s for lines %d-%d", suggestion.ID, start, end),
		}, nil
	})
}

func (s *Server) handleAccept(ctx context.Context, req *mcp.CallToolRequest, args AcceptSuggestionRequest) (*mcp.CallToolResult, any, error) {
	// Preview mode: show what would change, apply nothing
	if args.Preview {
		return withDoc(args.FilePath, func(absPath string, doc *comment.DocumentWithComments, _ *comment.LoadReport) (any, error) {
			suggestion, err := findSuggestion(doc, args.SuggestionID)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"preview":       true,
				"suggestion_id": args.SuggestionID,
				"start_line":    suggestion.StartLine,
				"end_line":      suggestion.EndLine,
				"original_text": suggestion.OriginalText,
				"proposed_text": suggestion.ProposedText,
				"message":       "Preview only - no changes applied",
			}, nil
		})
	}

	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		// Validate existence + suggestion-ness first for the pinned error shapes
		if _, err := findSuggestion(doc, args.SuggestionID); err != nil {
			return nil, err
		}

		// Apply, mark accepted, and shift displaced lines; withDocSave persists
		// the sidecar, but the changed CONTENT needs its own write — accept is
		// the one comment operation that edits the markdown
		if _, err := comment.ApplyAndAcceptSuggestion(doc, args.SuggestionID); err != nil {
			return nil, fmt.Errorf("failed to accept suggestion: %w", err)
		}
		if err := comment.SaveDocumentContent(absPath, doc); err != nil {
			return nil, fmt.Errorf("failed to write document: %w", err)
		}

		return map[string]any{
			"success": true,
			"message": fmt.Sprintf("Accepted and applied suggestion %s", args.SuggestionID),
		}, nil
	})
}

func (s *Server) handleReject(ctx context.Context, req *mcp.CallToolRequest, args RejectSuggestionRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		if err := comment.RejectSuggestion(doc.Threads, args.SuggestionID); err != nil {
			return nil, fmt.Errorf("failed to reject suggestion: %w", err)
		}

		return map[string]any{
			"success": true,
			"message": fmt.Sprintf("Rejected suggestion %s", args.SuggestionID),
		}, nil
	})
}

// Batch tools

func (s *Server) handleBatchAdd(ctx context.Context, req *mcp.CallToolRequest, args BatchAddRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		// Add all comments
		addedIDs := make([]string, 0, len(args.Comments))
		for _, commentData := range args.Comments {
			var newComment *comment.Comment

			if commentData.IsSuggestion {
				// Create suggestion
				newComment = comment.NewSuggestion(
					commentData.Author,
					commentData.StartLine,
					commentData.EndLine,
					commentData.Text,
					commentData.OriginalText,
					commentData.ProposedText,
				)
			} else {
				// Create regular comment
				line := commentData.Line
				if commentData.Section != "" && line == 0 {
					if err := comment.ValidateSectionPath(doc.Content, commentData.Section); err != nil {
						return nil, err
					}
					startLine, _, err := comment.ResolveSectionToLines(doc.Content, commentData.Section, false)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve section: %w", err)
					}
					line = startLine
				}
				if commentData.Anchor != "" && line == 0 {
					resolved, err := comment.ResolveAnchorText(doc.Content, commentData.Anchor)
					if err != nil {
						return nil, fmt.Errorf("anchor for comment by %s: %w", commentData.Author, err)
					}
					line = resolved
				}

				newComment = comment.NewComment(commentData.Author, line, comment.PrefixType(commentData.Text, commentData.Type))
				if commentData.Type != "" {
					newComment.Type = commentData.Type
				}
				if commentData.Status != "" {
					newComment.Status = commentData.Status
				}
				if commentData.Priority != "" {
					newComment.Priority = commentData.Priority
				}
				newComment.Blocking = commentData.Blocking
			}

			comment.UpdateCommentSection(newComment, doc.Content)
			doc.Threads = append(doc.Threads, newComment)
			addedIDs = append(addedIDs, newComment.ID)
		}

		return map[string]any{
			"success":     true,
			"added_count": len(addedIDs),
			"comment_ids": addedIDs,
			"message":     fmt.Sprintf("Added %d comments", len(addedIDs)),
		}, nil
	})
}

func (s *Server) handleBatchReply(ctx context.Context, req *mcp.CallToolRequest, args BatchReplyRequest) (*mcp.CallToolResult, any, error) {
	return withDocSave(args.FilePath, func(absPath string, doc *comment.DocumentWithComments) (any, error) {
		// Validate ALL thread IDs exist before adding any replies (atomic
		// all-or-nothing, matching the CLI batch-reply contract)
		threadIDs := make(map[string]bool)
		for _, t := range doc.Threads {
			threadIDs[t.ID] = true
		}
		missing := []string{}
		seenMissing := make(map[string]bool)
		for _, replyData := range args.Replies {
			if !threadIDs[replyData.ThreadID] && !seenMissing[replyData.ThreadID] {
				missing = append(missing, replyData.ThreadID)
				seenMissing[replyData.ThreadID] = true
			}
		}
		if len(missing) > 0 {
			available := make([]string, 0, len(doc.Threads))
			for _, t := range doc.Threads {
				available = append(available, t.ID)
			}
			return nil, fmt.Errorf(
				"batch rejected, no replies added: thread ID(s) not found: %s (available threads: %s)",
				strings.Join(missing, ", "), strings.Join(available, ", "))
		}

		// Add all replies; any failure past validation is reported explicitly,
		// and only what succeeded is in the saved document
		successCount := 0
		failed := []map[string]any{}
		for _, replyData := range args.Replies {
			if err := comment.AddReplyToThread(doc.Threads, replyData.ThreadID, replyData.Author, replyData.Text); err != nil {
				failed = append(failed, map[string]any{
					"thread_id": replyData.ThreadID,
					"error":     err.Error(),
				})
				continue
			}
			successCount++
		}

		// Report result; success only when every reply was added
		result := map[string]any{
			"success":     len(failed) == 0,
			"added_count": successCount,
			"total_count": len(args.Replies),
			"message":     fmt.Sprintf("Added %d replies out of %d", successCount, len(args.Replies)),
		}
		if len(failed) > 0 {
			result["failed"] = failed
		}
		return result, nil
	})
}

// Helper functions

func getContextLines(content string, targetLine int, contextSize int) []string {
	lines := strings.Split(content, "\n")
	start := targetLine - contextSize - 1
	if start < 0 {
		start = 0
	}
	end := targetLine + contextSize
	if end > len(lines) {
		end = len(lines)
	}

	result := make([]string, 0)
	for i := start; i < end; i++ {
		if i >= 0 && i < len(lines) {
			result = append(result, lines[i])
		}
	}
	return result
}
