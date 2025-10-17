package main

import (
	"fmt"
	"os"

	"github.com/rcliao/comments/pkg/comment"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <markdown-file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse the document
	parser := comment.NewParser()
	doc, err := parser.Parse(string(content))
	if err != nil {
		fmt.Printf("Error parsing document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d comments\n\n", len(doc.Comments))

	// Display comments
	for i, c := range doc.Comments {
		fmt.Printf("Comment %d:\n", i+1)
		fmt.Printf("  ID: %s\n", c.ID)
		fmt.Printf("  Author: %s\n", c.Author)
		fmt.Printf("  Line: %d\n", c.Line)
		fmt.Printf("  Time: %s\n", c.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Text: %s\n", c.Text)
		pos := doc.Positions[c.ID]
		fmt.Printf("  Position: Line %d, Column %d\n", pos.Line, pos.Column)
		fmt.Println()
	}

	// Display clean content
	fmt.Println("Clean Content:")
	fmt.Println("==============")
	fmt.Println(doc.Content)
	fmt.Println()

	// Test round-trip
	fmt.Println("Round-trip test:")
	fmt.Println("================")
	serializer := comment.NewSerializer()
	serialized, err := serializer.Serialize(doc.Content, doc.Comments, doc.Positions)
	if err != nil {
		fmt.Printf("Error serializing: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(serialized)
}
