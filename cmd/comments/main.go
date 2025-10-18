package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/tui"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "view":
		// View command can be called with or without a filename
		var filename string
		if len(os.Args) >= 3 {
			filename = os.Args[2]
		}
		viewCommand(filename)

	case "list":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments list <file> [flags]")
			os.Exit(1)
		}
		listCommand(os.Args[2], os.Args[3:])

	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments add <file> [flags]")
			os.Exit(1)
		}
		addCommand(os.Args[2], os.Args[3:])

	case "reply":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments reply <file> [flags]")
			os.Exit(1)
		}
		replyCommand(os.Args[2], os.Args[3:])

	case "resolve":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments resolve <file> [flags]")
			os.Exit(1)
		}
		resolveCommand(os.Args[2], os.Args[3:])

	case "export":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments export <file> [flags]")
			os.Exit(1)
		}
		exportCommand(os.Args[2], os.Args[3:])

	case "publish":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments publish <file> [flags]")
			os.Exit(1)
		}
		publishCommand(os.Args[2], os.Args[3:])

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func viewCommand(filename string) {
	var model tui.Model

	if filename == "" {
		// No filename provided - start with file picker
		model = tui.NewModel()
	} else {
		// Filename provided - load it directly
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Parse document
		parser := comment.NewParser()
		doc, err := parser.Parse(string(content))
		if err != nil {
			fmt.Printf("Error parsing document: %v\n", err)
			os.Exit(1)
		}

		// Create model with pre-loaded file
		model = tui.NewModelWithFile(doc, filename)
	}

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func listCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	typeFilter := fs.String("type", "", "Filter by comment type: Q, S, B, T, E")
	showResolved := fs.Bool("resolved", false, "Show resolved comments (default: false, only show unresolved)")

	fs.Parse(args)

	// Read file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse document
	parser := comment.NewParser()
	doc, err := parser.Parse(string(content))
	if err != nil {
		fmt.Printf("Error parsing document: %v\n", err)
		os.Exit(1)
	}

	// Filter by resolved status (only show root comments based on resolved flag)
	filteredComments := comment.GetVisibleComments(doc.Comments, *showResolved)

	// Filter comments by type if specified
	if *typeFilter != "" {
		filteredComments = filterCommentsByType(filteredComments, *typeFilter)
	}

	// Build threads to identify root vs reply
	threads := comment.BuildThreads(doc.Comments)

	// List comments
	statusText := "unresolved"
	if *showResolved {
		statusText = "total"
	}

	if *typeFilter != "" {
		fmt.Printf("Found %d %s comment(s) with type [%s] in %s\n\n", len(filteredComments), statusText, *typeFilter, filename)
	} else {
		fmt.Printf("Found %d %s comment(s) in %s\n\n", len(filteredComments), statusText, filename)
	}

	for i, c := range filteredComments {
		pos := doc.Positions[c.ID]

		// Determine if this is a root comment or reply
		commentType := "Root"
		if c.ParentID != "" {
			commentType = "Reply"
		}

		// Show thread ID, comment type, and basic info
		fmt.Printf("[%d] Line %d • @%s • %s\n", i+1, pos.Line, c.Author, c.Timestamp.Format("2006-01-02 15:04"))
		fmt.Printf("    Type: %s | Thread ID: %s | Comment ID: %s\n", commentType, c.ThreadID, c.ID)

		// Show reply count for root comments
		if c.ParentID == "" {
			if thread, ok := threads[c.ThreadID]; ok {
				replyCount := len(thread.Replies)
				resolvedStatus := ""
				if c.Resolved {
					resolvedStatus = " [RESOLVED]"
				}
				fmt.Printf("    Replies: %d%s\n", replyCount, resolvedStatus)
			}
		}

		fmt.Printf("    %s\n\n", c.Text)
	}
}

// filterCommentsByType filters comments by type prefix ([Q], [S], [B], [T], [E])
func filterCommentsByType(comments []*comment.Comment, typePrefix string) []*comment.Comment {
	filtered := make([]*comment.Comment, 0)
	targetPrefix := "[" + typePrefix + "]"

	for _, c := range comments {
		if len(c.Text) >= len(targetPrefix) && c.Text[:len(targetPrefix)] == targetPrefix {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

func addCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	text := fs.String("text", "", "Comment text (required)")
	line := fs.Int("line", 0, "Line number (required)")
	author := fs.String("author", "", "Author name (defaults to $USER)")
	commentType := fs.String("type", "", "Comment type: Q, S, B, T, E (auto-prefixes text)")

	fs.Parse(args)

	if *text == "" {
		fmt.Println("Error: --text flag is required")
		fmt.Println("Usage: comments add <file> --line N --text \"your comment\"")
		os.Exit(1)
	}

	if *line == 0 {
		fmt.Println("Error: --line flag is required")
		fmt.Println("Usage: comments add <file> --line N --text \"your comment\"")
		os.Exit(1)
	}

	// Default author
	if *author == "" {
		*author = os.Getenv("USER")
		if *author == "" {
			*author = "user"
		}
	}

	// Auto-prefix text with type if specified
	commentText := *text
	if *commentType != "" {
		commentText = "[" + *commentType + "] " + *text
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

	// Create new comment
	newComment := comment.NewComment(*author, *line, commentText)
	doc.Comments = append(doc.Comments, newComment)
	doc.Positions[newComment.ID] = comment.Position{Line: *line}

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

	fmt.Printf("✓ Comment added to line %d by @%s\n", *line, *author)
	fmt.Printf("  Comment ID: %s\n", newComment.ID)
}

func replyCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("reply", flag.ExitOnError)
	text := fs.String("text", "", "Reply text (required)")
	thread := fs.String("thread", "", "Thread ID (required)")
	author := fs.String("author", "", "Author name (defaults to $USER)")

	fs.Parse(args)

	if *text == "" {
		fmt.Println("Error: --text flag is required")
		fmt.Println("Usage: comments reply <file> --thread ID --text \"your reply\"")
		os.Exit(1)
	}

	if *thread == "" {
		fmt.Println("Error: --thread flag is required")
		fmt.Println("Usage: comments reply <file> --thread ID --text \"your reply\"")
		os.Exit(1)
	}

	// Default author
	if *author == "" {
		*author = os.Getenv("USER")
		if *author == "" {
			*author = "user"
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

	// Find the thread
	threads := comment.BuildThreads(doc.Comments)
	targetThread, exists := threads[*thread]
	if !exists {
		fmt.Printf("Error: Thread ID '%s' not found\n", *thread)
		fmt.Println("\nAvailable threads:")
		for id, t := range threads {
			fmt.Printf("  %s (Line %d, %d replies)\n", id, t.Line, len(t.Replies))
		}
		os.Exit(1)
	}

	// Create reply
	reply := comment.NewReply(*author, *thread, *text)
	doc.Comments = append(doc.Comments, reply)
	doc.Positions[reply.ID] = comment.Position{Line: targetThread.Line}

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

	fmt.Printf("✓ Reply added to thread %s by @%s\n", *thread, *author)
	fmt.Printf("  Reply ID: %s\n", reply.ID)
}

func resolveCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("resolve", flag.ExitOnError)
	thread := fs.String("thread", "", "Thread ID (required)")

	fs.Parse(args)

	if *thread == "" {
		fmt.Println("Error: --thread flag is required")
		fmt.Println("Usage: comments resolve <file> --thread ID")
		os.Exit(1)
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

	// Find the thread
	threads := comment.BuildThreads(doc.Comments)
	_, exists := threads[*thread]
	if !exists {
		fmt.Printf("Error: Thread ID '%s' not found\n", *thread)
		fmt.Println("\nAvailable threads:")
		for id, t := range threads {
			fmt.Printf("  %s (Line %d, %d replies)\n", id, t.Line, len(t.Replies))
		}
		os.Exit(1)
	}

	// Resolve the thread
	comment.ResolveThread(doc.Comments, *thread)

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

	fmt.Printf("✓ Thread %s marked as resolved\n", *thread)
}

func exportCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	format := fs.String("format", "json", "Export format (json)")
	output := fs.String("output", "", "Output file (defaults to stdout)")

	fs.Parse(args)

	if *format != "json" {
		fmt.Printf("Error: Unsupported format '%s'. Currently only 'json' is supported.\n", *format)
		os.Exit(1)
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

	// Build threads
	threads := comment.BuildThreads(doc.Comments)

	// Create export structure
	type ExportData struct {
		Comments  []*comment.Comment         `json:"comments"`
		Threads   map[string]*comment.Thread `json:"threads"`
		Positions map[string]comment.Position `json:"positions"`
		Metadata  map[string]interface{}      `json:"metadata"`
	}

	exportData := ExportData{
		Comments:  doc.Comments,
		Threads:   threads,
		Positions: doc.Positions,
		Metadata: map[string]interface{}{
			"filename":      filename,
			"total_comments": len(doc.Comments),
			"total_threads":  len(threads),
		},
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}

	// Output
	if *output != "" {
		// Write to file
		if err := os.WriteFile(*output, jsonData, 0644); err != nil {
			fmt.Printf("Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Exported to %s\n", *output)
	} else {
		// Write to stdout
		fmt.Println(string(jsonData))
	}
}

func publishCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	output := fs.String("output", "", "Output file (defaults to stdout)")

	fs.Parse(args)

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

	// Output clean content (without comment markup)
	if *output != "" {
		// Write to file
		if err := os.WriteFile(*output, []byte(doc.Content), 0644); err != nil {
			fmt.Printf("Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Published clean markdown to %s\n", *output)
	} else {
		// Write to stdout
		fmt.Print(doc.Content)
	}
}

func printUsage() {
	usage := `comments - CLI tool for collaborative document commenting

Usage:
  comments <command> [arguments]

Commands:
  view <file>                 Open interactive TUI viewer
  list <file> [flags]         List all comments in a file
  add <file> [flags]          Add a comment to a specific line
  reply <file> [flags]        Reply to a comment thread
  resolve <file> [flags]      Mark a thread as resolved
  export <file> [flags]       Export comments to JSON format
  publish <file> [flags]      Output clean markdown without comments
  help                        Show this help message

List Command Flags:
  --type <type>               Filter by comment type: Q, S, B, T, E
  --resolved                  Show resolved comments (default: false, only shows unresolved)

Add Command Flags:
  --line <number>             Line number (required)
  --text <text>               Comment text (required)
  --author <name>             Author name (defaults to $USER)
  --type <type>               Comment type: Q, S, B, T, E (auto-prefixes text)

Reply Command Flags:
  --thread <id>               Thread ID (required)
  --text <text>               Reply text (required)
  --author <name>             Author name (defaults to $USER)

Resolve Command Flags:
  --thread <id>               Thread ID (required)

Export Command Flags:
  --format <format>           Export format: json (default: json)
  --output <file>             Output file (default: stdout)

Publish Command Flags:
  --output <file>             Output file (default: stdout)

Examples:
  # Interactive mode
  comments view document.md
  comments list document.md                      # Show only unresolved comments
  comments list document.md --resolved           # Show all comments (including resolved)
  comments list document.md --type Q             # Show only unresolved questions
  comments list document.md --type B --resolved  # Show all blockers (resolved + unresolved)

  # Non-interactive comment management
  comments add document.md --line 10 --text "This needs review"
  comments add document.md --line 15 --text "Great point!" --author "bot"
  comments add document.md --line 20 --type Q --text "Is this correct?"  # Auto-prefixes with [Q]
  comments reply document.md --thread c123 --text "I agree"
  comments resolve document.md --thread c123

  # Export comments for programmatic access
  comments export document.md                    # Print JSON to stdout
  comments export document.md --output comments.json  # Save to file

  # Publish clean markdown (strip all comments)
  comments publish document.md                   # Print to stdout
  comments publish document.md --output final.md # Save to file

Keyboard shortcuts (in view mode):
  j/k or ↓/↑      Navigate comments
  c               Enter line selection mode
  q or Ctrl+C     Quit

For more information, visit: https://github.com/rcliao/comments
`
	fmt.Print(usage)
}
