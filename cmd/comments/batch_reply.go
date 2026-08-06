package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
)

// BatchReply represents a reply to be added in batch mode
type BatchReply struct {
	Thread string `json:"thread"`
	Author string `json:"author"`
	Text   string `json:"text"`
}

func batchReplyCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("batch-reply", flag.ContinueOnError)
	jsonInput := fs.String("json", "", "JSON file path (use '-' for stdin)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *jsonInput == "" {
		return failf("Error: --json flag is required\n" +
			"Usage: comments batch-reply <file> --json <file|->\n" +
			"Example: comments batch-reply doc.md --json replies.json\n" +
			"Example: echo '[{\"thread\":\"c123\",\"author\":\"claude\",\"text\":\"reply\"}]' | comments batch-reply doc.md --json -")
	}

	// Read JSON input (from file or stdin)
	var input []byte
	var err error

	if *jsonInput == "-" {
		// Read from stdin
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			return failf("Error reading from stdin: %v", err)
		}
	} else {
		// Read from file
		input, err = os.ReadFile(*jsonInput)
		if err != nil {
			return failf("Error reading JSON file: %v", err)
		}
	}

	// Parse batch replies
	var batchReplies []BatchReply
	if err := json.Unmarshal(input, &batchReplies); err != nil {
		return failf("Error parsing JSON: %v\n%s", err, `
Expected format:
[
  {"thread": "c123", "author": "claude", "text": "This looks good"},
  {"thread": "c456", "author": "alice", "text": "I agree"}
]`)
	}

	if len(batchReplies) == 0 {
		fmt.Println("No replies found in JSON input")
		return nil
	}

	// Validate replies
	for i, br := range batchReplies {
		if br.Thread == "" {
			return failf("Error: Reply %d has empty thread ID", i+1)
		}
		if br.Author == "" {
			return failf("Error: Reply %d has empty author (author is required)", i+1)
		}
		if br.Text == "" {
			return failf("Error: Reply %d has empty text", i+1)
		}
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Build thread ID lookup for validation
	threadIDs := make(map[string]bool)
	for _, t := range doc.Threads {
		threadIDs[t.ID] = true
	}

	// Validate all thread IDs exist before adding any replies
	invalidThreads := []string{}
	for _, br := range batchReplies {
		if !threadIDs[br.Thread] {
			invalidThreads = append(invalidThreads, br.Thread)
		}
	}

	if len(invalidThreads) > 0 {
		var b strings.Builder
		b.WriteString("Error: The following thread IDs were not found:")
		for _, tid := range invalidThreads {
			fmt.Fprintf(&b, "\n  - %s", tid)
		}
		b.WriteString("\n")
		b.WriteString(availableThreadsMsg(doc))
		return failf("%s", b.String())
	}

	// Add all replies to the document structure
	addedCount := 0

	for _, br := range batchReplies {
		// Use helper to add reply to thread
		if err := comment.AddReplyToThread(doc.Threads, br.Thread, br.Author, br.Text); err != nil {
			return failf("Error adding reply to thread %s: %v", br.Thread, err)
		}
		addedCount++
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	fmt.Printf("✓ Added %d reply/replies to %s\n", addedCount, filename)

	// Show summary of which threads were replied to
	threadCounts := make(map[string]int)
	for _, br := range batchReplies {
		threadCounts[br.Thread]++
	}

	fmt.Println("\nReplies by thread:")
	for threadID, count := range threadCounts {
		fmt.Printf("  %s: %d reply/replies\n", threadID, count)
	}
	return nil
}
