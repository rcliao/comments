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
	Line   int    `json:"line"`
	Author string `json:"author"`
	Text   string `json:"text"`
	Type   string `json:"type,omitempty"` // Q, S, B, T, E

	// Suggestion fields (optional)
	SuggestionType string              `json:"suggestion_type,omitempty"` // line, char-range, multi-line, diff-hunk
	Selection      *BatchSelection     `json:"selection,omitempty"`
	ProposedText   string              `json:"proposed_text,omitempty"`
}

// BatchSelection represents the selection for a suggestion in batch mode
type BatchSelection struct {
	StartLine  int    `json:"start_line,omitempty"`
	EndLine    int    `json:"end_line,omitempty"`
	ByteOffset int    `json:"byte_offset,omitempty"`
	Length     int    `json:"length,omitempty"`
	Original   string `json:"original,omitempty"`
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
		fmt.Println("\nExpected format (regular comment):")
		fmt.Println(`[
  {"line": 10, "author": "alice", "text": "Add examples", "type": "S"},
  {"line": 25, "author": "bob", "text": "Great point!"}
]`)
		fmt.Println("\nExpected format (suggestion):")
		fmt.Println(`[
  {
    "line": 15,
    "author": "claude",
    "text": "Improve wording",
    "type": "S",
    "suggestion_type": "line",
    "selection": {
      "start_line": 15,
      "end_line": 15,
      "original": "old text"
    },
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
		// Validate suggestion fields if suggestion_type is specified
		if bc.SuggestionType != "" {
			validSuggestionTypes := map[string]bool{
				"line": true, "char-range": true, "multi-line": true, "diff-hunk": true,
			}
			if !validSuggestionTypes[bc.SuggestionType] {
				fmt.Printf("Error: Comment %d has invalid suggestion_type '%s'. Valid types: line, char-range, multi-line, diff-hunk\n", i+1, bc.SuggestionType)
				os.Exit(1)
			}
			if bc.Selection == nil {
				fmt.Printf("Error: Comment %d is a suggestion but missing 'selection' field\n", i+1)
				os.Exit(1)
			}
			if bc.ProposedText == "" {
				fmt.Printf("Error: Comment %d is a suggestion but missing 'proposed_text' field\n", i+1)
				os.Exit(1)
			}
			// Type-specific validation
			if bc.SuggestionType == "char-range" {
				if bc.Selection.ByteOffset < 0 {
					fmt.Printf("Error: Comment %d (char-range) has invalid byte_offset\n", i+1)
					os.Exit(1)
				}
			}
		}
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Sort comments by line number in DESCENDING order for consistency
	sort.Slice(batchComments, func(i, j int) bool {
		return batchComments[i].Line > batchComments[j].Line
	})

	// Add all comments to the document structure
	addedCount := 0
	addedComments := []*comment.Comment{}

	for _, bc := range batchComments {
		// Auto-prefix text with type if specified (only for non-suggestions)
		text := bc.Text
		if bc.Type != "" && bc.SuggestionType == "" {
			text = "[" + bc.Type + "] " + text
		}

		// Create new comment with type metadata
		var newComment *comment.Comment
		if bc.Type != "" {
			newComment = comment.NewCommentWithType(bc.Author, bc.Line, text, bc.Type)
		} else {
			newComment = comment.NewComment(bc.Author, bc.Line, text)
		}

		// If this is a suggestion, populate suggestion fields
		if bc.SuggestionType != "" {
			newComment.SuggestionType = comment.SuggestionType(bc.SuggestionType)
			newComment.ProposedText = bc.ProposedText
			newComment.AcceptanceState = comment.AcceptancePending

			// Convert BatchSelection to Selection
			newComment.Selection = &comment.Selection{
				StartLine:  bc.Selection.StartLine,
				EndLine:    bc.Selection.EndLine,
				ByteOffset: bc.Selection.ByteOffset,
				Length:     bc.Selection.Length,
				Original:   bc.Selection.Original,
			}

			// Set defaults for line-based suggestions
			if bc.SuggestionType == "line" || bc.SuggestionType == "multi-line" {
				if newComment.Selection.StartLine == 0 {
					newComment.Selection.StartLine = bc.Line
				}
				if bc.SuggestionType == "line" && newComment.Selection.EndLine == 0 {
					newComment.Selection.EndLine = bc.Line
				}
			}
		}

		doc.Comments = append(doc.Comments, newComment)
		doc.Positions[newComment.ID] = comment.Position{Line: bc.Line}
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

		for _, c := range verifyDoc.Comments {
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
