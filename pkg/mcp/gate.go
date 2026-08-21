package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rcliao/comments/pkg/comment"
)

const (
	defaultReviewTimeout = 600 * time.Second
	reviewPollInterval   = 2 * time.Second
)

// gateFileResult is the per-file payload returned by comments_gate
type gateFileResult struct {
	File               string                `json:"file"`
	Decision           string                `json:"decision"`
	Blocking           []gateComment         `json:"blocking"`
	NonBlocking        []gateComment         `json:"non_blocking"`
	PendingSuggestions []gateComment         `json:"pending_suggestions"`
	LastReview         *comment.ReviewRecord `json:"last_review,omitempty"`
	Template           string                `json:"template,omitempty"`
	Violations         []comment.Violation   `json:"violations,omitempty"`
	StructureUnchecked bool                  `json:"structure_unchecked,omitempty"`
}

// gateComment mirrors the CLI gate JSON shape (snake_case)
type gateComment struct {
	ID          string   `json:"id"`
	Author      string   `json:"author"`
	Line        int      `json:"line"`
	Text        string   `json:"text"`
	Type        string   `json:"type,omitempty"`
	Priority    string   `json:"priority"`
	Blocking    bool     `json:"blocking"`
	SectionPath string   `json:"section_path,omitempty"`
	Context     []string `json:"context,omitempty"`
}

func (s *Server) handleGate(ctx context.Context, req *mcp.CallToolRequest, args GateRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}

	decision, files, err := evaluateGateForPath(absPath, args.Strict, args.Template)
	if err != nil {
		return nil, nil, err
	}

	result := map[string]any{
		"decision": decision,
		"strict":   args.Strict,
		"files":    files,
	}
	return jsonToolResult(result)
}

// handleRequestReview requests a human review. Default (blocking) mode waits
// until a signoff (comments signoff) newer than the request is recorded, then
// returns the review decision and gate state. With blocking=false it returns
// immediately with a `since` handle for comments_check_review polling — the
// handle is just a timestamp compared against the sidecar's review history, so
// it survives agent restarts.
func (s *Server) handleRequestReview(ctx context.Context, req *mcp.CallToolRequest, args RequestReviewRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, fmt.Errorf("document not found: %w", err)
	}

	if args.Blocking != nil && !*args.Blocking {
		result := map[string]any{
			"status":  "requested",
			"since":   time.Now().Format(time.RFC3339Nano),
			"message": fmt.Sprintf("review requested; poll comments_check_review with this since value after the human runs: comments signoff %s", args.FilePath),
		}
		return jsonToolResult(result)
	}

	timeout := defaultReviewTimeout
	if args.TimeoutSeconds > 0 {
		timeout = time.Duration(args.TimeoutSeconds) * time.Second
	}
	requestedAt := time.Now()
	deadline := requestedAt.Add(timeout)

	for {
		if review := comment.LatestReviewSince(absPath, requestedAt); review != nil {
			decision, files, err := evaluateGateForPath(absPath, args.Strict, "")
			if err != nil {
				return nil, nil, err
			}
			result := map[string]any{
				"status":        "review_completed",
				"review":        review,
				"gate_decision": decision,
				"files":         files,
			}
			return jsonToolResult(result)
		}

		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(reviewPollInterval):
		}
		if time.Now().After(deadline) {
			result := map[string]any{
				"status":  "timeout",
				"message": fmt.Sprintf("no human signoff within %s; ask the reviewer to run: comments signoff %s", timeout, args.FilePath),
			}
			return jsonToolResult(result)
		}
	}
}

// handleCheckReview polls a durable review handle: it compares the sidecar's
// newest review record against the `since` timestamp. Pending until a human
// signoff newer than `since` exists; then returns the same completed payload
// shape as blocking comments_request_review.
func (s *Server) handleCheckReview(ctx context.Context, req *mcp.CallToolRequest, args CheckReviewRequest) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, fmt.Errorf("document not found: %w", err)
	}
	since, err := time.Parse(time.RFC3339, args.Since)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid since timestamp (want RFC3339, e.g. the value returned by comments_request_review): %w", err)
	}

	review := comment.LatestReviewSince(absPath, since)
	if review == nil {
		result := map[string]any{
			"status": "pending",
			"since":  args.Since,
		}
		return jsonToolResult(result)
	}

	decision, files, err := evaluateGateForPath(absPath, args.Strict, "")
	if err != nil {
		return nil, nil, err
	}
	result := map[string]any{
		"status":        "review_completed",
		"review":        review,
		"gate_decision": decision,
		"files":         files,
	}
	return jsonToolResult(result)
}

func evaluateGateForPath(path string, strict bool, templateName string) (string, []gateFileResult, error) {
	targets, err := comment.FindGateTargets(path)
	if err != nil {
		return "", nil, err
	}
	if len(targets) == 0 {
		return "", nil, fmt.Errorf("no markdown files with comment sidecars found under %s", path)
	}

	decision := comment.DecisionApproved
	files := make([]gateFileResult, 0, len(targets))
	for _, file := range targets {
		doc, _, err := loadDoc(file)
		if err != nil {
			return "", nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		result := comment.EvaluateGate(doc, strict)
		fileResult := gateFileResult{
			File:               file,
			Decision:           result.Decision,
			Blocking:           toGateComments(result.Blocking, doc.Content),
			NonBlocking:        toGateComments(result.NonBlocking, doc.Content),
			PendingSuggestions: toGateComments(result.PendingSuggestions, doc.Content),
			LastReview:         result.LastReview,
		}
		tName := templateName
		if tName == "" {
			tName = doc.Template
		}
		if tName != "" {
			t, err := comment.LoadTemplateForDoc(tName, file)
			if err != nil {
				return "", nil, err
			}
			fileResult.Template = t.Name
			fileResult.Violations = comment.ValidateDocument(doc.Content, file, t)
			if len(fileResult.Violations) > 0 {
				fileResult.Decision = comment.DecisionChangesRequested
			}
		} else if len(doc.Threads) > 0 {
			fileResult.StructureUnchecked = true
		}
		files = append(files, fileResult)
		if fileResult.Decision == comment.DecisionChangesRequested {
			decision = comment.DecisionChangesRequested
		}
	}
	return decision, files, nil
}

func toGateComments(comments []*comment.Comment, docContent string) []gateComment {
	out := []gateComment{}
	for _, c := range comments {
		out = append(out, gateComment{
			ID:          c.ID,
			Author:      c.Author,
			Line:        c.Line,
			Text:        c.Text,
			Type:        c.Type,
			Priority:    c.GetPriority(),
			Blocking:    c.Blocking,
			SectionPath: c.SectionPath,
			Context:     getContextLines(docContent, c.Line, 2),
		})
	}
	return out
}

func jsonToolResult(v any) (*mcp.CallToolResult, any, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(jsonData)},
		},
	}, nil, nil
}
