package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rcliao/comments/pkg/comment"
)

// BatchComment represents a comment to be added in batch mode
type BatchComment struct {
	Line   int    `json:"line"`
	Author string `json:"author"`
	Text   string `json:"text"`
	Type   string `json:"type,omitempty"` // Q, S, B, T, E
}

func batchAddCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("batch-add", flag.ExitOnError)
	jsonInput := fs.String("json", "", "JSON file path (use '-' for stdin)")

	fs.Parse(args)

	if *jsonInput == "" {
		fmt.Println("Error: --json flag is required")
		fmt.Println("Usage: comments batch-add <file> --json <file|->")
		fmt.Println("Example: comments batch-add doc.md --json reviews.json")
		fmt.Println("Example: echo '[{\"line\":10,\"text\":\"comment\"}]' | comments batch-add doc.md --json -")
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

	// Parse batch comments
	var batchComments []BatchComment
	if err := json.Unmarshal(input, &batchComments); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		fmt.Println("\nExpected format:")
		fmt.Println(`[
  {"line": 10, "author": "alice", "text": "Add examples", "type": "S"},
  {"line": 25, "text": "Great point!"}
]`)
		os.Exit(1)
	}

	if len(batchComments) == 0 {
		fmt.Println("No comments found in JSON input")
		os.Exit(0)
	}

	// Validate comments
	for i, bc := range batchComments {
		if bc.Line <= 0 {
			fmt.Printf("Error: Comment %d has invalid line number: %d\n", i+1, bc.Line)
			os.Exit(1)
		}
		if bc.Text == "" {
			fmt.Printf("Error: Comment %d has empty text\n", i+1)
			os.Exit(1)
		}
		if bc.Author == "" {
			fmt.Printf("Error: Comment %d has empty author (author is required)\n", i+1)
			os.Exit(1)
		}
		// Validate type if specified
		if bc.Type != "" {
			validTypes := map[string]bool{"Q": true, "S": true, "B": true, "T": true, "E": true}
			if !validTypes[bc.Type] {
				fmt.Printf("Error: Comment %d has invalid type '%s'. Valid types: Q, S, B, T, E\n", i+1, bc.Type)
				os.Exit(1)
			}
		}
	}

	// Read and parse file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	parser := comment.NewParser()
	doc, err := parser.Parse(string(content))
	if err != nil {
		fmt.Printf("Error parsing document: %v\n", err)
		os.Exit(1)
	}

	// Add all comments
	addedCount := 0
	for _, bc := range batchComments {
		// Auto-prefix text with type if specified
		text := bc.Text
		if bc.Type != "" {
			text = "[" + bc.Type + "] " + text
		}

		// Create new comment
		newComment := comment.NewComment(bc.Author, bc.Line, text)
		doc.Comments = append(doc.Comments, newComment)
		doc.Positions[newComment.ID] = comment.Position{Line: bc.Line}
		addedCount++
	}

	// Serialize and save
	serializer := comment.NewSerializer()
	updatedContent, err := serializer.Serialize(doc.Content, doc.Comments, doc.Positions)
	if err != nil {
		fmt.Printf("Error serializing document: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filename, []byte(updatedContent), 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Added %d comment(s) to %s\n", addedCount, filename)
}
