package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// The agent surface: one command per purpose, each the twin of an MCP tool of
// the same name and each a thin adapter over one pkg/comment entry point, so
// the two surfaces cannot drift. Human decisions — accepting a suggestion, the
// verdict — are made in `comments view` and have no command here.

// absOrSame returns the absolute form of path, or path itself if that fails.
func absOrSame(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func printJSON(v any) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return failf("Error encoding JSON: %v", err)
	}
	fmt.Println(string(encoded))
	return nil
}

// addCommand adds one comment from flags, or many from --json. A single
// comment is a batch of one: both go through comment.AddComments.
func addCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	text := fs.String("text", "", "Comment text (supports @filename)")
	line := fs.Int("line", 0, "Line number (use one of --line, --section or --anchor)")
	section := fs.String("section", "", "Section path (use one of --line, --section or --anchor)")
	anchor := fs.String("anchor", "", "Quote the target line, or a unique substring of it (preferred for agents)")
	author := fs.String("author", "", "Author name")
	commentType := fs.String("type", "", "Comment type: Q, S, B, T, E (auto-prefixes text)")
	priority := fs.String("priority", "", "Priority: low, medium (default), high")
	blocking := fs.Bool("blocking", false, "Must be resolved before the gate passes")
	pick := fs.String("pick", "", "The option you proceed with unless the human objects (never blocking; settled at the end review)")
	jsonInput := fs.String("json", "", "Add many: JSON file with an array of comments (use '-' for stdin)")
	jsonOut := fs.Bool("json-out", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	var specs []comment.NewCommentSpec
	if *jsonInput != "" {
		input, err := readJSONInput(*jsonInput)
		if err != nil {
			return failf("%v", err)
		}
		if err := json.Unmarshal(input, &specs); err != nil {
			return failf("Error parsing JSON: %v\n%s", err, `
Expected format — each comment is placed by exactly one of anchor, section or line:
[
  {"anchor": "quoted target line", "author": "claude", "text": "Is this measured?", "type": "Q", "blocking": true},
  {"section": "Doc Title > Proposed Design", "author": "claude", "text": "Assumption, not a measurement."}
]`)
		}
	} else {
		resolved, err := resolveTextInput(*text)
		if err != nil {
			return failf("Error: %v", err)
		}
		specs = []comment.NewCommentSpec{{
			Line: *line, Section: *section, Anchor: *anchor, Author: *author,
			Text: resolved, Type: *commentType, Priority: *priority, Blocking: *blocking, Pick: *pick,
		}}
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}
	result, err := comment.AddComments(doc, specs)
	if err != nil {
		return failf("Error: %v\nUsage: comments add <file> --anchor \"quoted line\" --author NAME --text \"...\" [--blocking]\n   or: comments add <file> --json comments.json", err)
	}
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}
	if *jsonOut {
		return printJSON(result)
	}
	for _, c := range result.Added {
		where := fmt.Sprintf("line %d", c.Line)
		if c.SectionPath != "" {
			where = fmt.Sprintf("%s (Line %d)", c.SectionPath, c.Line)
		}
		fmt.Printf("✓ Comment added to %s by @%s\n  Comment ID: %s\n", where, c.Author, c.ID)
	}
	return nil
}

// replyCommand replies to a thread and, with --resolve, closes it. Resolving
// used to be its own command; agents always ran the two as a pair, and a bare
// resolve let a thread close unexplained. The zone guard travels with it.
func replyCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("reply", flag.ContinueOnError)
	threadID := fs.String("thread", "", "Thread ID (a reply ID resolves to its thread)")
	text := fs.String("text", "", "Reply text (supports @filename); optional with --resolve")
	author := fs.String("author", "", "Author name")
	resolve := fs.Bool("resolve", false, "Also resolve the thread (refused for an agent in a zone: human section)")
	jsonInput := fs.String("json", "", "Reply to many: JSON file with an array of replies (use '-' for stdin)")
	jsonOut := fs.Bool("json-out", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	var specs []comment.ReplySpec
	if *jsonInput != "" {
		input, err := readJSONInput(*jsonInput)
		if err != nil {
			return failf("%v", err)
		}
		if err := json.Unmarshal(input, &specs); err != nil {
			return failf("Error parsing JSON: %v\n%s", err, `
Expected format:
[
  {"thread_id": "c7f3k", "author": "claude", "text": "Fixed: now cites the dashboard.", "resolve": true},
  {"thread_id": "c9b21", "author": "claude", "text": "I disagree, because ..."}
]`)
		}
	} else {
		resolved, err := resolveTextInput(*text)
		if err != nil {
			return failf("Error: %v", err)
		}
		specs = []comment.ReplySpec{{ThreadID: *threadID, Author: *author, Text: resolved, Resolve: *resolve}}
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}
	absPath := absOrSame(filename)
	// The guard needs to know who is calling: the same rule on every surface.
	result, err := comment.ReplyToThreads(doc, absPath, specs, comment.ResolveActor(comment.StdoutIsTTY()))
	if err != nil {
		msg := fmt.Sprintf("Error: %v", err)
		if strings.Contains(err.Error(), "thread not found") {
			msg += "\n" + availableThreadsMsg(doc)
		}
		return failf("%s", msg)
	}
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}
	if *jsonOut {
		return printJSON(result)
	}
	for _, id := range result.Replied {
		fmt.Printf("✓ Reply added to thread %s\n", id)
	}
	for _, id := range result.Resolved {
		fmt.Printf("✓ Thread %s resolved\n", id)
	}
	return nil
}

// getCommand is the lookup path: one comment with its context, or — with no
// --thread — every thread in the document, resolved ones included. The
// worklist is `comments inbox`; this answers "what did that thread say".
func getCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	threadID := fs.String("thread", "", "Comment or reply ID; omit to list every thread")
	fromDoc := fs.String("from", "", "Citing document, for resolving a citation's relative path / same-doc form")
	unresolved := fs.Bool("unresolved", false, "When listing: leave out resolved threads")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	// Citation-literal form: `comments get thread:research.md#c1abc [--from plan.md]`
	// — accept the syntax exactly as it appears in docs, so agents following a
	// citation paste it instead of translating it
	if strings.HasPrefix(filename, "thread:") {
		path, id, err := comment.ResolveThreadCitation(filename, *fromDoc)
		if err != nil {
			return failf("Error: %v", err)
		}
		filename, *threadID = path, id
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}
	comment.ComputeSectionsForComments(doc)

	if *threadID == "" {
		list := comment.ListThreads(doc, absOrSame(filename), *unresolved)
		if *jsonOut {
			return printJSON(list)
		}
		if list.Total == 0 {
			fmt.Println("No comments found")
			return nil
		}
		var shown []*comment.Comment
		for _, t := range doc.Threads {
			if !*unresolved || !t.Resolved {
				shown = append(shown, t)
			}
		}
		fmt.Print(formatListWithContext(shown, doc.Content))
		return nil
	}

	if *jsonOut {
		detail, err := comment.GetThread(doc, *threadID, 5)
		if err != nil {
			return failf("Error: %v\n%s", err, availableThreadsMsg(doc))
		}
		return printJSON(detail)
	}
	found := doc.FindCommentByID(*threadID)
	if found == nil {
		return failf("Error: comment not found: %s\n%s", *threadID, availableThreadsMsg(doc))
	}
	fmt.Print(formatCommentWithContext(found, getCommentContext(found, doc.Content), true))
	return nil
}

// watchCommand emits review-state changes as NDJSON, one event per line. The
// sidecar is the shared event bus — every writer persists there — so one
// watcher observes them all. The MCP comments_watch tool runs the same loop.
func watchCommand(target string, args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	interval := fs.Duration("interval", time.Second, "Poll interval")
	until := fs.String("until", "", "Exit 0 after emitting an event matching this comma-separated list of event types (e.g. signoff,gate_changed)")
	sinceFlag := fs.String("since", "", "RFC3339 hand-off time: a verdict recorded after it is emitted at once, so a fast reviewer is never missed")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}
	var since time.Time
	if *sinceFlag != "" {
		var err error
		if since, err = time.Parse(time.RFC3339, *sinceFlag); err != nil {
			return failf("Error: invalid --since timestamp (want RFC3339): %v", err)
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	err := comment.Watch(context.Background(), target, comment.WatchOptions{Interval: *interval, Until: *until, Since: since}, func(e comment.WatchEvent) bool {
		// A failed write means stdout is gone (EPIPE: the consumer exited). Stop
		// cleanly instead of lingering as an orphan writing into a broken pipe.
		return encoder.Encode(e) != nil
	})
	if err != nil {
		return failf("Error: %v", err)
	}
	return nil
}
