package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rcliao/comments/pkg/comment"
)

// BatchReply represents a reply to be added in batch mode
type BatchReply struct {
	Thread string `json:"thread"`
	Author string `json:"author"`
	Text   string `json:"text"`
}

func batchReplyCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("batch-reply", flag.ExitOnError)
	jsonInput := fs.String("json", "", "JSON file path (use '-' for stdin)")

	fs.Parse(args)

	if *jsonInput == "" {
		fmt.Println("Error: --json flag is required")
		fmt.Println("Usage: comments batch-reply <file> --json <file|->")
		fmt.Println("Example: comments batch-reply doc.md --json replies.json")
		fmt.Println("Example: echo '[{\"thread\":\"c123\",\"author\":\"claude\",\"text\":\"reply\"}]' | comments batch-reply doc.md --json -")
		os.Exit(1)
	}

	// Read JSON input (from file or stdin)
	var input []byte
	var err error

	if *jsonInput == "-" {
		// Read from stdin
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Read from file
		input, err = os.ReadFile(*jsonInput)
		if err != nil {
			fmt.Printf("Error reading JSON file: %v\n", err)
			os.Exit(1)
		}
	}

	// Parse batch replies
	var batchReplies []BatchReply
	if err := json.Unmarshal(input, &batchReplies); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		fmt.Println("\nExpected format:")
		fmt.Println(`[
  {"thread": "c123", "author": "claude", "text": "This looks good"},
  {"thread": "c456", "author": "alice", "text": "I agree"}
]`)
		os.Exit(1)
	}

	if len(batchReplies) == 0 {
		fmt.Println("No replies found in JSON input")
		os.Exit(0)
	}

	// Validate replies
	for i, br := range batchReplies {
		if br.Thread == "" {
			fmt.Printf("Error: Reply %d has empty thread ID\n", i+1)
			os.Exit(1)
		}
		if br.Author == "" {
			fmt.Printf("Error: Reply %d has empty author (author is required)\n", i+1)
			os.Exit(1)
		}
		if br.Text == "" {
			fmt.Printf("Error: Reply %d has empty text\n", i+1)
			os.Exit(1)
		}
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Build threads to validate thread IDs exist
	threads := comment.BuildThreads(doc.Comments)

	// Validate all thread IDs exist before adding any replies
	invalidThreads := []string{}
	for _, br := range batchReplies {
		if _, exists := threads[br.Thread]; !exists {
			invalidThreads = append(invalidThreads, br.Thread)
		}
	}

	if len(invalidThreads) > 0 {
		fmt.Printf("Error: The following thread IDs were not found:\n")
		for _, tid := range invalidThreads {
			fmt.Printf("  - %s\n", tid)
		}
		fmt.Println("\nAvailable threads:")
		for id, t := range threads {
			fmt.Printf("  %s (Line %d, %d replies)\n", id, t.Line, len(t.Replies))
		}
		os.Exit(1)
	}

	// Add all replies to the document structure
	addedCount := 0
	addedReplies := []*comment.Comment{}

	for _, br := range batchReplies {
		targetThread := threads[br.Thread]

		// Create reply
		reply := comment.NewReply(br.Author, br.Thread, br.Text)
		doc.Comments = append(doc.Comments, reply)
		doc.Positions[reply.ID] = comment.Position{Line: targetThread.Line}
		addedReplies = append(addedReplies, reply)
		addedCount++
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	// Verify replies were added correctly by re-loading
	verifyDoc, err := comment.LoadFromSidecar(filename)
	if err == nil {
		// Count how many of our replies are present
		verifiedCount := 0
		replyIDs := make(map[string]bool)
		for _, r := range addedReplies {
			replyIDs[r.ID] = true
		}

		for _, c := range verifyDoc.Comments {
			if replyIDs[c.ID] {
				verifiedCount++
			}
		}

		if verifiedCount != addedCount {
			fmt.Printf("⚠ Warning: Added %d reply/replies but only %d were verified in the file\n", addedCount, verifiedCount)
		}
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
}
