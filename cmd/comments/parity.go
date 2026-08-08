package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// The commands in this file mirror MCP tools that previously had no CLI
// equivalent. Each one calls the same pkg/comment entry point the MCP handler
// calls, so the two surfaces cannot drift again.

// reanchorCommand migrates comment anchors the caller's edits displaced.
// Single move via flags, or a batch via --json.
func reanchorCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("reanchor", flag.ContinueOnError)
	thread := fs.String("comment", "", "Comment ID to move (or use --json for a batch)")
	line := fs.Int("line", 0, "New line number for the comment")
	section := fs.String("section", "", "New section path for the comment")
	jsonInput := fs.String("json", "", "JSON file path with a moves array (use '-' for stdin)")
	jsonOut := fs.Bool("json-out", false, "Output machine-readable JSON results")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	var moves []comment.Move
	if *jsonInput != "" {
		input, err := readJSONInput(*jsonInput)
		if err != nil {
			return failf("%v", err)
		}
		if err := json.Unmarshal(input, &moves); err != nil {
			return failf("Error parsing JSON: %v\n%s", err, `
Expected format:
[
  {"comment_id": "c7f3k", "line": 42},
  {"comment_id": "c9b21", "section": "Proposed Design"}
]`)
		}
	} else {
		if *thread == "" {
			return failf("Error: --comment is required (or --json for a batch)\n" +
				"Usage: comments reanchor <file> --comment ID --line N\n" +
				"       comments reanchor <file> --json moves.json")
		}
		if *line == 0 && *section == "" {
			return failf("Error: a move needs --line or --section")
		}
		moves = []comment.Move{{CommentID: *thread, Line: *line, Section: *section}}
	}
	if len(moves) == 0 {
		return failf("Error: no moves given")
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	results := comment.ApplyMoves(doc, moves)

	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(map[string]any{"results": results}, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
		return nil
	}

	moved := 0
	for _, r := range results {
		if r.Moved {
			moved++
			fmt.Printf("  ✓ %s → line %d", r.CommentID, r.Line)
			if r.SectionPath != "" {
				fmt.Printf(" (%s)", r.SectionPath)
			}
			fmt.Println()
		} else {
			fmt.Printf("  ✗ %s: %s\n", r.CommentID, r.Error)
		}
	}
	fmt.Printf("Re-anchored %d of %d comment(s) in %s\n", moved, len(results), filename)
	if moved < len(results) {
		return exitSilent(1)
	}
	return nil
}

// inboxCommand is the one-call attention view: unresolved threads with new
// replies, plus every unresolved blocking thread.
func inboxCommand(target string, args []string) error {
	fs := flag.NewFlagSet("inbox", flag.ContinueOnError)
	since := fs.String("since", "", "RFC3339 timestamp: only threads with replies newer than this")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		return failf("Error: invalid path: %v", err)
	}

	var sinceTime time.Time
	if *since != "" {
		sinceTime, err = time.Parse(time.RFC3339, *since)
		if err != nil {
			return failf("Error: invalid --since timestamp (want RFC3339): %v", err)
		}
	}

	items, err := comment.BuildInbox(absPath, sinceTime)
	if err != nil {
		return failf("Error: %v", err)
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(map[string]any{
			"since": *since,
			"count": len(items),
			"items": items,
		}, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
		return nil
	}

	if len(items) == 0 {
		fmt.Printf("Inbox empty — nothing waiting in %s\n", target)
		return nil
	}

	fmt.Printf("%d thread(s) need attention in %s\n\n", len(items), target)
	for i, item := range items {
		rel := item.File
		if r, err := filepath.Rel(absPath, item.File); err == nil && !strings.HasPrefix(r, "..") {
			rel = r
		}
		fmt.Printf("[%d] %s (line %d) • %s\n", i+1, rel, item.Thread.Line, strings.Join(item.Reasons, ", "))
		// item.Thread is a CommentView, whose Text is already decorated
		fmt.Printf("    %s: %s\n", item.Thread.Author, item.Thread.Text)
		if item.LastReply != nil {
			fmt.Printf("    ↳ latest reply @%s: %s\n", item.LastReply.Author, item.LastReply.Text)
		}
		fmt.Printf("    Thread ID: %s\n\n", item.Thread.ID)
	}
	return nil
}

// statusCommand reports document-level review statistics.
func statusCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return failf("Error: invalid path: %v", err)
	}
	doc, report, err := comment.LoadDocument(absPath)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	var resolved, unresolved, blocking int
	for _, t := range doc.Threads {
		if t.Resolved {
			resolved++
			continue
		}
		unresolved++
		if t.Blocking {
			blocking++
		}
	}

	all := doc.GetAllComments()
	var pendingSuggestions, orphaned int
	for _, c := range all {
		if c.Status == "orphaned" {
			orphaned++
		}
		if c.IsSuggestion && c.Accepted == nil {
			pendingSuggestions++
		}
	}

	status := map[string]any{
		"filepath":            absPath,
		"total_threads":       len(doc.Threads),
		"total_comments":      len(all),
		"resolved_threads":    resolved,
		"unresolved_threads":  unresolved,
		"blocking_threads":    blocking,
		"pending_suggestions": pendingSuggestions,
		"orphaned_comments":   orphaned,
		"is_stale":            report.Stale,
		"template":            doc.Template,
		"document_hash":       doc.DocumentHash,
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
		return nil
	}

	fmt.Printf("Status: %s\n\n", filename)
	fmt.Printf("  Threads         %d total — %d unresolved (%d blocking), %d resolved\n",
		len(doc.Threads), unresolved, blocking, resolved)
	fmt.Printf("  Comments        %d including replies\n", len(all))
	fmt.Printf("  Suggestions     %d pending\n", pendingSuggestions)
	fmt.Printf("  Orphaned        %d\n", orphaned)
	if doc.Template != "" {
		fmt.Printf("  Template        %s\n", doc.Template)
	}
	if report.Stale {
		fmt.Printf("  ⚠ Document changed since the sidecar was written — anchors were revalidated\n")
	}
	return nil
}

// checkReviewCommand polls for a signoff landed after --since. It is the
// non-blocking counterpart to `watch --until signoff`, and survives restarts
// because the handle is just a timestamp.
func checkReviewCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("check-review", flag.ContinueOnError)
	since := fs.String("since", "", "RFC3339 timestamp to check for reviews after (required)")
	strict := fs.Bool("strict", false, "Fail on any unresolved comment or pending suggestion")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}
	if *since == "" {
		return failf("Error: --since is required (RFC3339)\n" +
			"Usage: comments check-review <file> --since 2026-08-08T09:00:00Z")
	}
	sinceTime, err := time.Parse(time.RFC3339, *since)
	if err != nil {
		return failf("Error: invalid --since timestamp (want RFC3339): %v", err)
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return failf("Error: invalid path: %v", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return failf("Error: document not found: %v", err)
	}

	review := comment.LatestReviewSince(absPath, sinceTime)
	if review == nil {
		if *jsonOut {
			encoded, _ := json.MarshalIndent(map[string]any{
				"status": "pending", "since": *since,
			}, "", "  ")
			fmt.Println(string(encoded))
		} else {
			fmt.Printf("Pending — no signoff on %s since %s\n", filename, *since)
		}
		// Pending is not a failure: exit 0 so a polling loop can distinguish
		// "no review yet" from the gate's 10 (changes requested).
		return nil
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}
	result := comment.EvaluateGate(doc, *strict)

	if *jsonOut {
		encoded, err := json.MarshalIndent(map[string]any{
			"status":        "review_completed",
			"review":        review,
			"gate_decision": result.Decision,
		}, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
	} else {
		fmt.Printf("Review completed by @%s: %s\n", review.Author, review.Decision)
		if review.Note != "" {
			fmt.Printf("  Note: %s\n", review.Note)
		}
		fmt.Printf("  Gate: %s\n", result.Decision)
	}
	if result.Decision != comment.DecisionApproved {
		return exitSilent(comment.GateExitCode)
	}
	return nil
}

// batchAcceptCommand accepts several suggestions in one call. The usage text
// has advertised these flags since before the command was wired up.
func batchAcceptCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("batch-accept", flag.ContinueOnError)
	jsonInput := fs.String("json", "", "JSON file path with suggestion IDs (use '-' for stdin)")
	author := fs.String("author", "", "Accept all pending suggestions from this author")
	typeFilter := fs.String("type", "", "Accept all pending suggestions of this type")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}
	if *jsonInput == "" && *author == "" && *typeFilter == "" {
		return failf("Error: one of --json, --author or --type is required\n" +
			"Usage: comments batch-accept <file> --author claude\n" +
			"       comments batch-accept <file> --json ids.json")
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	var ids []string
	if *jsonInput != "" {
		input, err := readJSONInput(*jsonInput)
		if err != nil {
			return failf("%v", err)
		}
		if err := json.Unmarshal(input, &ids); err != nil {
			return failf("Error parsing JSON: %v\nExpected format: [\"c7f3k\", \"c9b21\"]", err)
		}
	} else {
		for _, c := range doc.GetAllComments() {
			if !c.IsSuggestion || c.Accepted != nil {
				continue
			}
			if *author != "" && c.Author != *author {
				continue
			}
			if *typeFilter != "" {
				if t, ok := comment.LeadingType(c.Text); !ok || t != *typeFilter {
					continue
				}
			}
			ids = append(ids, c.ID)
		}
	}

	if len(ids) == 0 {
		fmt.Println("No matching pending suggestions.")
		return nil
	}

	// ApplyAndAcceptSuggestion shifts the lines of comments the edit displaced,
	// so applying sequentially keeps the remaining ranges valid.
	accepted, skipped := 0, 0
	for _, id := range ids {
		if _, err := comment.ApplyAndAcceptSuggestion(doc, id); err != nil {
			fmt.Printf("  ✗ %s: %v\n", id, err)
			skipped++
			continue
		}
		fmt.Printf("  ✓ %s accepted\n", id)
		accepted++
	}

	// Accept is a content-changing path, so the markdown is written too
	if err := comment.SaveDocumentContent(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	fmt.Printf("Accepted %d suggestion(s)", accepted)
	if skipped > 0 {
		fmt.Printf(", skipped %d", skipped)
	}
	fmt.Printf(" in %s\n", filename)
	if skipped > 0 {
		return exitSilent(1)
	}
	return nil
}

// readJSONInput reads a JSON payload from a file path or stdin ("-").
func readJSONInput(source string) ([]byte, error) {
	if source == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("error reading from stdin: %v", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file: %v", err)
	}
	return data, nil
}
