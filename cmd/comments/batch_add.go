package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/rcliao/comments/pkg/comment"
)

// BatchComment represents a comment to be added in batch mode
type BatchComment struct {
	Line    int    `json:"line,omitempty"`    // Line number (use either line or section)
	Section string `json:"section,omitempty"` // Section path (use either line or section)
	Author  string `json:"author"`
	Text    string `json:"text"`
	Type    string `json:"type,omitempty"` // Q, S, B, T, E

	// Suggestion fields (optional) - simplified to multi-line only
	IsSuggestion bool   `json:"is_suggestion,omitempty"`
	StartLine    int    `json:"start_line,omitempty"`
	EndLine      int    `json:"end_line,omitempty"`
	OriginalText string `json:"original_text,omitempty"`
	ProposedText string `json:"proposed_text,omitempty"`
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
		fmt.Println("\nExpected format (regular comment with line):")
		fmt.Println(`[
  {"line": 10, "author": "alice", "text": "Add examples", "type": "S"},
  {"line": 25, "author": "bob", "text": "Great point!"}
]`)
		fmt.Println("\nExpected format (comment with section):")
		fmt.Println(`[
  {"section": "Introduction > Overview", "author": "alice", "text": "Consider adding examples", "type": "S"}
]`)
		fmt.Println("\nExpected format (multi-line suggestion):")
		fmt.Println(`[
  {
    "line": 15,
    "author": "claude",
    "text": "Improve wording",
    "is_suggestion": true,
    "start_line": 15,
    "end_line": 17,
    "original_text": "old text",
    "proposed_text": "new text"
  }
]`)
		os.Exit(1)
	}

	if len(batchComments) == 0 {
		fmt.Println("No comments found in JSON input")
		os.Exit(0)
	}

	// Validate comments
	for i, bc := range batchComments {
		// Validate that either line or section is provided (but not both)
		if bc.Line == 0 && bc.Section == "" {
			fmt.Printf("Error: Comment %d must specify either 'line' or 'section'\n", i+1)
			os.Exit(1)
		}
		if bc.Line != 0 && bc.Section != "" {
			fmt.Printf("Error: Comment %d cannot specify both 'line' and 'section'\n", i+1)
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
		// Validate suggestion fields if is_suggestion is true
		if bc.IsSuggestion {
			if bc.StartLine == 0 {
				fmt.Printf("Error: Comment %d is a suggestion but missing 'start_line' field\n", i+1)
				os.Exit(1)
			}
			if bc.EndLine == 0 {
				fmt.Printf("Error: Comment %d is a suggestion but missing 'end_line' field\n", i+1)
				os.Exit(1)
			}
			if bc.StartLine > bc.EndLine {
				fmt.Printf("Error: Comment %d has start_line (%d) > end_line (%d)\n", i+1, bc.StartLine, bc.EndLine)
				os.Exit(1)
			}
			if bc.ProposedText == "" {
				fmt.Printf("Error: Comment %d is a suggestion but missing 'proposed_text' field\n", i+1)
				os.Exit(1)
			}
		}
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Resolve section paths to line numbers
	for i := range batchComments {
		if batchComments[i].Section != "" {
			// Validate section exists
			if err := comment.ValidateSectionPath(doc.Content, batchComments[i].Section); err != nil {
				fmt.Printf("Error in comment %d: %v\n", i+1, err)
				os.Exit(1)
			}

			// Resolve section to line number (use section start line)
			startLine, _, err := comment.ResolveSectionToLines(doc.Content, batchComments[i].Section, false)
			if err != nil {
				fmt.Printf("Error resolving section for comment %d: %v\n", i+1, err)
				os.Exit(1)
			}
			batchComments[i].Line = startLine
		}
	}

	// Sort comments by line number in DESCENDING order for consistency
	sort.Slice(batchComments, func(i, j int) bool {
		return batchComments[i].Line > batchComments[j].Line
	})

	// Add all comments to the document structure
	addedCount := 0
	addedComments := []*comment.Comment{}

	for _, bc := range batchComments {
		var newComment *comment.Comment

		// Check if this is a suggestion
		if bc.IsSuggestion {
			// Create suggestion comment
			newComment = comment.NewSuggestion(
				bc.Author,
				bc.StartLine,
				bc.EndLine,
				bc.Text,
				bc.OriginalText,
				bc.ProposedText,
			)
		} else {
			// Auto-prefix text with type if specified
			text := bc.Text
			if bc.Type != "" {
				text = "[" + bc.Type + "] " + text
			}

			// Create regular comment with type metadata
			if bc.Type != "" {
				newComment = comment.NewCommentWithType(bc.Author, bc.Line, text, bc.Type)
			} else {
				newComment = comment.NewComment(bc.Author, bc.Line, text)
			}
		}

		// Compute section metadata for the new comment
		comment.UpdateCommentSection(newComment, doc.Content)

		doc.Threads = append(doc.Threads, newComment)
		addedComments = append(addedComments, newComment)
		addedCount++
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	// Verify comments were added correctly by re-loading
	verifyDoc, err := comment.LoadFromSidecar(filename)
	if err == nil {
		// Count how many of our comments are present
		verifiedCount := 0
		commentIDs := make(map[string]bool)
		for _, c := range addedComments {
			commentIDs[c.ID] = true
		}

		for _, c := range verifyDoc.Threads {
			if commentIDs[c.ID] {
				verifiedCount++
			}
		}

		if verifiedCount != addedCount {
			fmt.Printf("⚠ Warning: Added %d comment(s) but only %d were verified in the file\n", addedCount, verifiedCount)
		}
	}

	fmt.Printf("✓ Added %d comment(s) to %s\n", addedCount, filename)
}
