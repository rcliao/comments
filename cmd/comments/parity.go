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
	reviewer := fs.String("reviewer", "", "Whose last verdict to diff changed lines against (default: the latest reviewer)")
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

	inbox, err := comment.BuildInbox(absPath, comment.InboxOptions{Since: sinceTime, Reviewer: *reviewer, ContextSize: 2})
	if err != nil {
		return failf("Error: %v", err)
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(inbox, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
		return nil
	}
	printInbox(target, absPath, inbox)
	return nil
}

// printInbox renders the inbox for a terminal: the verdict first, then what
// stands between the document and approval.
func printInbox(target, absPath string, inbox *comment.Inbox) {
	rel := func(file string) string {
		if r, err := filepath.Rel(absPath, file); err == nil && r != "." && !strings.HasPrefix(r, "..") {
			return r
		}
		return filepath.Base(file)
	}
	fmt.Printf("Decision: %s — %d open thread(s), %d pending suggestion(s) in %s\n", inbox.Decision, inbox.Count, len(inbox.PendingSuggestions), target)
	for _, f := range inbox.Files {
		for _, v := range f.Violations {
			fmt.Printf("  ✗ %s [%s] %s\n", rel(f.File), v.Rule, v.Message)
		}
		if f.Orphaned > 0 {
			fmt.Printf("  ⚠ %s: %d orphaned comment(s) — run comments reanchor\n", rel(f.File), f.Orphaned)
		}
		if c := f.Changes; c != nil && (c.Lines > 0 || c.Deleted > 0) {
			fmt.Printf("  Δ %s: %d line(s), %d deletion(s) since @%s's last verdict\n", rel(f.File), c.Lines, c.Deleted, c.Reviewer)
		}
	}
	if inbox.Count == 0 && len(inbox.PendingSuggestions) == 0 {
		fmt.Println("\nInbox empty — nothing waiting.")
		return
	}
	fmt.Println()
	for i, item := range inbox.Items {
		fmt.Printf("[%d] %s (line %d) • %s\n", i+1, rel(item.File), item.Thread.Line, strings.Join(item.Reasons, ", "))
		// item.Thread is a CommentView, whose Text is already decorated
		fmt.Printf("    %s: %s\n", item.Thread.Author, item.Thread.Text)
		if item.LastReply != nil {
			fmt.Printf("    ↳ latest reply @%s: %s\n", item.LastReply.Author, item.LastReply.Text)
		}
		fmt.Printf("    Thread ID: %s\n\n", item.Thread.ID)
	}
	for _, sg := range inbox.PendingSuggestions {
		fmt.Printf("[suggestion] line %d • awaiting the human's decision in comments view\n    %s: %s\n    Suggestion ID: %s\n\n", sg.Line, sg.Author, sg.Text, sg.ID)
	}
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
