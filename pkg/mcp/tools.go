package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

// Read tools

func (s *Server) handleListComments(ctx context.Context, req *mcp.CallToolRequest, args ListCommentsRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Get all comments (flattened from threads)
	allComments := doc.GetAllComments()

	// Apply filters
	filtered := make([]*comment.Comment, 0)
	for _, c := range allComments {
		// Author filter
		if args.Author != "" && c.Author != args.Author {
			continue
		}

		// Type filter
		if args.Type != "" && c.Type != args.Type {
			continue
		}

		// Section filter
		if args.Section != "" && !strings.HasPrefix(c.SectionPath, args.Section) {
			continue
		}

		// Search filter
		if args.Search != "" && !strings.Contains(strings.ToLower(c.Text), strings.ToLower(args.Search)) {
			continue
		}

		// Status filter
		if args.Status != "" && c.Status != args.Status {
			continue
		}

		// Priority filter
		if args.Priority != "" && c.Priority != args.Priority {
			continue
		}

		// Resolved filter
		if args.Resolved != nil && c.Resolved != *args.Resolved {
			continue
		}

		// Line range filter
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
		"comments": toCommentJSONList(filtered),
	}

	// Add context if requested
	if args.WithContext {
		content, _ := os.ReadFile(absPath)
		contextComments := make([]CommentWithContext, len(filtered))
		for i, c := range filtered {
			contextComments[i] = CommentWithContext{
				Comment:      toCommentJSON(c),
				SectionPath:  c.SectionPath,
				ContextLines: getContextLines(string(content), c.Line, 3),
				IsOrphaned:   c.Status == "orphaned",
			}
		}
		result["comments_with_context"] = contextComments
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleGetComment(ctx context.Context, req *mcp.CallToolRequest, args GetCommentRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Find the comment
	var foundComment *comment.Comment
	allComments := doc.GetAllComments()
	for _, c := range allComments {
		if c.ID == args.CommentID {
			foundComment = c
			break
		}
	}

	if foundComment == nil {
		return nil, nil, fmt.Errorf("comment not found: %s", args.CommentID)
	}

	// Load document content for context
	content, _ := os.ReadFile(absPath)

	// Build response with context
	result := CommentWithContext{
		Comment:      toCommentJSON(foundComment),
		SectionPath:  foundComment.SectionPath,
		ContextLines: getContextLines(string(content), foundComment.Line, 5),
		IsOrphaned:   foundComment.Status == "orphaned",
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleStatus(ctx context.Context, req *mcp.CallToolRequest, args StatusRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Compute staleness from the raw sidecar BEFORE LoadFromSidecar, which
	// re-validates and rewrites the sidecar with a refreshed hash. Stale means
	// the markdown content changed since the sidecar was last written.
	isStale := false
	if raw, readErr := os.ReadFile(comment.GetSidecarPath(absPath)); readErr == nil {
		var stored struct {
			DocumentHash string `json:"documentHash"`
		}
		if json.Unmarshal(raw, &stored) == nil && stored.DocumentHash != "" {
			if content, contentErr := os.ReadFile(absPath); contentErr == nil {
				isStale = stored.DocumentHash != comment.ComputeDocumentHash(string(content))
			}
		}
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

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

	// Build status response
	status := DocumentStatus{
		FilePath:            absPath,
		TotalThreads:        totalThreads,
		TotalComments:       len(allComments),
		ResolvedThreads:     resolvedThreads,
		UnresolvedThreads:   unresolvedThreads,
		PendingSuggestions:  pendingSuggestions,
		OrphanedComments:    orphanedComments,
		IsStale:             isStale,
		DocumentHash:        doc.DocumentHash,
		LastValidated:       doc.LastValidated.Format("2006-01-02T15:04:05Z"),
		SuggestionsByAuthor: suggestionsByAuthor,
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize status: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

// Write tools

func (s *Server) handleAddComment(ctx context.Context, req *mcp.CallToolRequest, args AddCommentRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Determine line number
	line := args.Line
	if args.Section != "" && line == 0 {
		if err := comment.ValidateSectionPath(doc.Content, args.Section); err != nil {
			return nil, nil, err
		}
		startLine, _, err := comment.ResolveSectionToLines(doc.Content, args.Section, false)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve section: %w", err)
		}
		line = startLine
	}

	// Create new comment
	newComment := comment.NewComment(args.Author, line, args.Text)
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

	// Save
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success with comment ID
	result := map[string]any{
		"success":    true,
		"comment_id": newComment.ID,
		"message":    fmt.Sprintf("Added comment %s at line %d", newComment.ID, line),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleReply(ctx context.Context, req *mcp.CallToolRequest, args ReplyRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Add reply to thread
	if err := comment.AddReplyToThread(doc.Threads, args.ThreadID, args.Author, args.Text); err != nil {
		return nil, nil, fmt.Errorf("failed to add reply: %w", err)
	}

	// Save
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success
	result := map[string]any{
		"success": true,
		"message": fmt.Sprintf("Added reply to thread %s", args.ThreadID),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleResolve(ctx context.Context, req *mcp.CallToolRequest, args ResolveRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Zone enforcement: threads in a template's human-decision zone must be
	// resolved by the human (CLI/TUI), not by an agent over MCP
	if !args.Unresolve && doc.Template != "" {
		if thread := doc.FindThreadByID(args.ThreadID); thread != nil {
			if t, err := comment.LoadTemplate(doc.Template); err == nil {
				if comment.SectionZone(doc.Content, t, thread.Line) == comment.ZoneHuman {
					return nil, nil, fmt.Errorf(
						"thread %s is in a human-decision zone (template %q); reply with your input instead — the human resolves it via 'comments resolve' or the TUI",
						args.ThreadID, doc.Template)
				}
			}
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
		return nil, nil, fmt.Errorf("failed to resolve thread: %w", actionErr)
	}

	// Save
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success
	action := "resolved"
	if args.Unresolve {
		action = "unresolved"
	}
	result := map[string]any{
		"success": true,
		"message": fmt.Sprintf("Thread %s %s", args.ThreadID, action),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

// Suggestion tools

func (s *Server) handleSuggest(ctx context.Context, req *mcp.CallToolRequest, args SuggestRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Create suggestion
	suggestion := comment.NewSuggestion(
		args.Author,
		args.StartLine,
		args.EndLine,
		args.Text,
		args.OriginalText,
		args.ProposedText,
	)

	// Add to document
	doc.Threads = append(doc.Threads, suggestion)

	// Save
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success with suggestion ID
	result := map[string]any{
		"success":       true,
		"suggestion_id": suggestion.ID,
		"message":       fmt.Sprintf("Created suggestion %s for lines %d-%d", suggestion.ID, args.StartLine, args.EndLine),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleAccept(ctx context.Context, req *mcp.CallToolRequest, args AcceptSuggestionRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// If preview mode, just show what would change
	if args.Preview {
		// Find the suggestion
		var suggestion *comment.Comment
		allComments := doc.GetAllComments()
		for _, c := range allComments {
			if c.ID == args.SuggestionID && c.IsSuggestion {
				suggestion = c
				break
			}
		}

		if suggestion == nil {
			return nil, nil, fmt.Errorf("suggestion not found: %s", args.SuggestionID)
		}

		// Return preview
		result := map[string]any{
			"preview":       true,
			"suggestion_id": args.SuggestionID,
			"start_line":    suggestion.StartLine,
			"end_line":      suggestion.EndLine,
			"original_text": suggestion.OriginalText,
			"proposed_text": suggestion.ProposedText,
			"message":       "Preview only - no changes applied",
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(jsonData)},
			},
		}, nil, nil
	}

	// Find the suggestion first
	var suggestion *comment.Comment
	allComments := doc.GetAllComments()
	for _, c := range allComments {
		if c.ID == args.SuggestionID && c.IsSuggestion {
			suggestion = c
			break
		}
	}

	if suggestion == nil {
		return nil, nil, fmt.Errorf("suggestion not found: %s", args.SuggestionID)
	}

	// Apply the suggestion to the document content
	newContent, err := comment.ApplySuggestion(doc.Content, suggestion)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to apply suggestion: %w", err)
	}

	// Accept the suggestion (marks it as accepted)
	if err := comment.AcceptSuggestion(doc.Threads, args.SuggestionID); err != nil {
		return nil, nil, fmt.Errorf("failed to accept suggestion: %w", err)
	}

	// Recalculate line numbers for all other comments/suggestions displaced
	// by the applied edit (mirrors the CLI accept path)
	comment.RecalculateCommentLines(doc.Threads, suggestion.StartLine, suggestion.EndLine,
		comment.SuggestionLinesAdded(suggestion))

	// Write the updated content
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to write updated document: %w", err)
	}

	// Update document content and save
	doc.Content = newContent
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success
	result := map[string]any{
		"success": true,
		"message": fmt.Sprintf("Accepted and applied suggestion %s", args.SuggestionID),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleReject(ctx context.Context, req *mcp.CallToolRequest, args RejectSuggestionRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

	// Reject the suggestion
	if err := comment.RejectSuggestion(doc.Threads, args.SuggestionID); err != nil {
		return nil, nil, fmt.Errorf("failed to reject suggestion: %w", err)
	}

	// Save
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success
	result := map[string]any{
		"success": true,
		"message": fmt.Sprintf("Rejected suggestion %s", args.SuggestionID),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

// Batch tools

func (s *Server) handleBatchAdd(ctx context.Context, req *mcp.CallToolRequest, args BatchAddRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

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
					return nil, nil, err
				}
				startLine, _, err := comment.ResolveSectionToLines(doc.Content, commentData.Section, false)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to resolve section: %w", err)
				}
				line = startLine
			}

			newComment = comment.NewComment(commentData.Author, line, commentData.Text)
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

	// Save
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
	}

	// Return success
	result := map[string]any{
		"success":     true,
		"added_count": len(addedIDs),
		"comment_ids": addedIDs,
		"message":     fmt.Sprintf("Added %d comments", len(addedIDs)),
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}

func (s *Server) handleBatchReply(ctx context.Context, req *mcp.CallToolRequest, args BatchReplyRequest) (*mcp.CallToolResult, any, error) {
	// Make path absolute
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Load document with comments
	doc, err := comment.LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load comments: %w", err)
	}

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
		return nil, nil, fmt.Errorf(
			"batch rejected, no replies added: thread ID(s) not found: %s (available threads: %s)",
			strings.Join(missing, ", "), strings.Join(available, ", "))
	}

	// Add all replies; any failure past validation is reported explicitly
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

	// Save (only what succeeded is in the document)
	if err := comment.SaveToSidecar(absPath, doc); err != nil {
		return nil, nil, fmt.Errorf("failed to save comments: %w", err)
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

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
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
