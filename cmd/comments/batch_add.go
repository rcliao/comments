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
	Line     int    `json:"line,omitempty"`    // Line number (one of line/section/anchor)
	Section  string `json:"section,omitempty"` // Section path (one of line/section/anchor)
	Anchor   string `json:"anchor,omitempty"`  // Quote of the target line (or unique substring)
	Author   string `json:"author"`
	Text     string `json:"text"`
	Type     string `json:"type,omitempty"`     // Q, S, B, T, E
	Priority string `json:"priority,omitempty"` // low, medium, high
	Blocking bool   `json:"blocking,omitempty"` // Must be resolved before gate passes

	// Suggestion fields (optional) - simplified to multi-line only
	IsSuggestion bool   `json:"is_suggestion,omitempty"`
	StartLine    int    `json:"start_line,omitempty"`
	EndLine      int    `json:"end_line,omitempty"`
	OriginalText string `json:"original_text,omitempty"`
	ProposedText string `json:"proposed_text,omitempty"`
}

func batchAddCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("batch-add", flag.ContinueOnError)
	jsonInput := fs.String("json", "", "JSON file path (use '-' for stdin)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *jsonInput == "" {
		return failf("Error: --json flag is required\n" +
			"Usage: comments batch-add <file> --json <file|->\n" +
			"Example: comments batch-add doc.md --json reviews.json\n" +
			"Example: echo '[{\"line\":10,\"text\":\"comment\"}]' | comments batch-add doc.md --json -")
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

	// Parse batch comments
	var batchComments []BatchComment
	if err := json.Unmarshal(input, &batchComments); err != nil {
		return failf("Error parsing JSON: %v\n%s", err, `
Expected format (regular comment with line):
[
  {"line": 10, "author": "alice", "text": "Add examples", "type": "S"},
  {"line": 25, "author": "bob", "text": "Great point!"}
]

Expected format (comment with section):
[
  {"section": "Introduction > Overview", "author": "alice", "text": "Consider adding examples", "type": "S"}
]

Expected format (multi-line suggestion):
[
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
	}

	if len(batchComments) == 0 {
		fmt.Println("No comments found in JSON input")
		return nil
	}

	// Validate comments
	for i, bc := range batchComments {
		// Exactly one of line / section / anchor locates the comment
		given := 0
		for _, ok := range []bool{bc.Line != 0, bc.Section != "", bc.Anchor != ""} {
			if ok {
				given++
			}
		}
		if given == 0 {
			return failf("Error: Comment %d must specify one of 'line', 'section' or 'anchor'", i+1)
		}
		if given > 1 {
			return failf("Error: Comment %d: 'line', 'section' and 'anchor' are mutually exclusive", i+1)
		}
		if bc.Text == "" {
			return failf("Error: Comment %d has empty text", i+1)
		}
		if bc.Author == "" {
			return failf("Error: Comment %d has empty author (author is required)", i+1)
		}
		// Validate type if specified
		if bc.Type != "" {
			validTypes := map[string]bool{"Q": true, "S": true, "B": true, "T": true, "E": true}
			if !validTypes[bc.Type] {
				return failf("Error: Comment %d has invalid type '%s'. Valid types: Q, S, B, T, E", i+1, bc.Type)
			}
		}
		// Validate suggestion fields if is_suggestion is true
		if bc.IsSuggestion {
			if bc.StartLine == 0 {
				return failf("Error: Comment %d is a suggestion but missing 'start_line' field", i+1)
			}
			if bc.EndLine == 0 {
				return failf("Error: Comment %d is a suggestion but missing 'end_line' field", i+1)
			}
			if bc.StartLine > bc.EndLine {
				return failf("Error: Comment %d has start_line (%d) > end_line (%d)", i+1, bc.StartLine, bc.EndLine)
			}
			if bc.ProposedText == "" {
				return failf("Error: Comment %d is a suggestion but missing 'proposed_text' field", i+1)
			}
		}
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Resolve section paths to line numbers
	for i := range batchComments {
		if batchComments[i].Section != "" {
			// Validate section exists
			if err := comment.ValidateSectionPath(doc.Content, batchComments[i].Section); err != nil {
				return failf("Error in comment %d: %v", i+1, err)
			}

			// Resolve section to line number (use section start line)
			startLine, _, err := comment.ResolveSectionToLines(doc.Content, batchComments[i].Section, false)
			if err != nil {
				return failf("Error resolving section for comment %d: %v", i+1, err)
			}
			batchComments[i].Line = startLine
		}
		if batchComments[i].Anchor != "" {
			resolved, err := comment.ResolveAnchorText(doc.Content, batchComments[i].Anchor)
			if err != nil {
				return failf("Error in comment %d: %v", i+1, err)
			}
			batchComments[i].Line = resolved
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
			// Auto-prefix text with type (normalized: never doubles a
			// marker the author already wrote)
			text := comment.PrefixType(bc.Text, bc.Type)

			// Create regular comment with type metadata
			if bc.Type != "" {
				newComment = comment.NewCommentWithType(bc.Author, bc.Line, text, bc.Type)
			} else {
				newComment = comment.NewComment(bc.Author, bc.Line, text)
			}
		}

		if bc.Priority != "" {
			newComment.Priority = bc.Priority
		}
		newComment.Blocking = bc.Blocking

		// Compute section metadata for the new comment
		comment.UpdateCommentSection(newComment, doc.Content)

		doc.Threads = append(doc.Threads, newComment)
		addedComments = append(addedComments, newComment)
		addedCount++
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	// Verify comments were added correctly by re-loading
	verifyDoc, _, err := comment.LoadFromSidecar(filename)
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
	return nil
}
