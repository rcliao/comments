package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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

	case "batch-add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments batch-add <file> [flags]")
			os.Exit(1)
		}
		batchAddCommand(os.Args[2], os.Args[3:])

	case "reply":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments reply <file> [flags]")
			os.Exit(1)
		}
		replyCommand(os.Args[2], os.Args[3:])

	case "batch-reply":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments batch-reply <file> [flags]")
			os.Exit(1)
		}
		batchReplyCommand(os.Args[2], os.Args[3:])

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

	case "suggest":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments suggest <file> [flags]")
			os.Exit(1)
		}
		suggestCommand(os.Args[2], os.Args[3:])

	case "accept":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments accept <file> [flags]")
			os.Exit(1)
		}
		acceptCommand(os.Args[2], os.Args[3:])

	case "reject":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments reject <file> [flags]")
			os.Exit(1)
		}
		rejectCommand(os.Args[2], os.Args[3:])

	case "batch-accept":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments batch-accept <file> [flags]")
			os.Exit(1)
		}
		batchAcceptCommand(os.Args[2], os.Args[3:])

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
		doc, err := comment.LoadFromSidecar(filename)
		if err != nil {
			fmt.Printf("Error loading document: %v\n", err)
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
	authorFilter := fs.String("author", "", "Filter by author name")
	searchText := fs.String("search", "", "Search comment text (case-insensitive)")
	lineRange := fs.String("line-range", "", "Filter by line range (e.g., 10-30)")
	sortBy := fs.String("sort", "line", "Sort by: line, timestamp, author")
	format := fs.String("format", "text", "Output format: text, json, table")

	fs.Parse(args)

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Filter by resolved status (only show root comments based on resolved flag)
	filteredComments := comment.GetVisibleComments(doc.Comments, *showResolved)

	// Filter comments by type if specified
	if *typeFilter != "" {
		filteredComments = filterCommentsByType(filteredComments, *typeFilter)
	}

	// Apply author filter
	if *authorFilter != "" {
		filteredComments = filterByAuthor(filteredComments, *authorFilter)
	}

	// Apply text search filter
	if *searchText != "" {
		filteredComments = filterBySearch(filteredComments, *searchText)
	}

	// Apply line range filter
	if *lineRange != "" {
		filtered, err := filterByLineRange(filteredComments, *lineRange)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		filteredComments = filtered
	}

	// Sort comments
	sortComments(filteredComments, *sortBy)

	// Output based on format
	switch *format {
	case "json":
		if err := outputJSON(filteredComments, doc.Positions, doc.Comments); err != nil {
			fmt.Printf("Error outputting JSON: %v\n", err)
			os.Exit(1)
		}
		return

	case "table":
		outputTable(filteredComments, doc.Positions, doc.Comments)
		return

	case "text":
		// Original text format (below)

	default:
		fmt.Printf("Error: Unknown format '%s'. Valid formats: text, json, table\n", *format)
		os.Exit(1)
	}

	// Build threads to identify root vs reply
	threads := comment.BuildThreads(doc.Comments)

	// List comments (original text format)
	statusText := "unresolved"
	if *showResolved {
		statusText = "total"
	}

	// Build filter description
	filterDesc := ""
	if *typeFilter != "" {
		filterDesc += fmt.Sprintf(" with type [%s]", *typeFilter)
	}
	if *authorFilter != "" {
		filterDesc += fmt.Sprintf(" by @%s", *authorFilter)
	}
	if *searchText != "" {
		filterDesc += fmt.Sprintf(" matching '%s'", *searchText)
	}
	if *lineRange != "" {
		filterDesc += fmt.Sprintf(" in lines %s", *lineRange)
	}

	fmt.Printf("Found %d %s comment(s)%s in %s\n\n", len(filteredComments), statusText, filterDesc, filename)

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
	author := fs.String("author", "", "Author name (required)")
	commentType := fs.String("type", "", "Comment type: Q, S, B, T, E (auto-prefixes text)")

	fs.Parse(args)

	if *text == "" {
		fmt.Println("Error: --text flag is required")
		fmt.Println("Usage: comments add <file> --line N --author \"name\" --text \"your comment\"")
		os.Exit(1)
	}

	if *line == 0 {
		fmt.Println("Error: --line flag is required")
		fmt.Println("Usage: comments add <file> --line N --author \"name\" --text \"your comment\"")
		os.Exit(1)
	}

	if *author == "" {
		fmt.Println("Error: --author flag is required")
		fmt.Println("Usage: comments add <file> --line N --author \"name\" --text \"your comment\"")
		os.Exit(1)
	}

	// Auto-prefix text with type if specified
	commentText := *text
	if *commentType != "" {
		commentText = "[" + *commentType + "] " + *text
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Create new comment with type metadata
	var newComment *comment.Comment
	if *commentType != "" {
		newComment = comment.NewCommentWithType(*author, *line, commentText, *commentType)
	} else {
		newComment = comment.NewComment(*author, *line, commentText)
	}
	doc.Comments = append(doc.Comments, newComment)
	doc.Positions[newComment.ID] = comment.Position{Line: *line}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
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
	author := fs.String("author", "", "Author name (required)")

	fs.Parse(args)

	if *text == "" {
		fmt.Println("Error: --text flag is required")
		fmt.Println("Usage: comments reply <file> --thread ID --author \"name\" --text \"your reply\"")
		os.Exit(1)
	}

	if *thread == "" {
		fmt.Println("Error: --thread flag is required")
		fmt.Println("Usage: comments reply <file> --thread ID --author \"name\" --text \"your reply\"")
		os.Exit(1)
	}

	if *author == "" {
		fmt.Println("Error: --author flag is required")
		fmt.Println("Usage: comments reply <file> --thread ID --author \"name\" --text \"your reply\"")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
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

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
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

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
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

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
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

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
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

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
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

func suggestCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("suggest", flag.ExitOnError)
	line := fs.Int("line", 0, "Line number (required)")
	author := fs.String("author", "", "Author name (required)")
	text := fs.String("text", "", "Suggestion description (required)")
	suggestionType := fs.String("type", "line", "Suggestion type: line, char-range, multi-line, diff-hunk")
	startLine := fs.Int("start-line", 0, "Start line (for multi-line)")
	endLine := fs.Int("end-line", 0, "End line (for multi-line)")
	byteOffset := fs.Int("offset", 0, "Byte offset (for char-range)")
	length := fs.Int("length", 0, "Length in bytes (for char-range)")
	original := fs.String("original", "", "Original text to replace")
	proposed := fs.String("proposed", "", "Proposed replacement text (required)")

	fs.Parse(args)

	// Validate required flags
	if *author == "" {
		fmt.Println("Error: --author flag is required")
		os.Exit(1)
	}
	if *text == "" {
		fmt.Println("Error: --text flag is required")
		os.Exit(1)
	}
	if *proposed == "" {
		fmt.Println("Error: --proposed flag is required")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Create suggestion based on type
	var selection *comment.Selection
	var sugType comment.SuggestionType

	switch *suggestionType {
	case "line":
		if *line == 0 {
			fmt.Println("Error: --line flag is required for line suggestions")
			os.Exit(1)
		}
		sugType = comment.SuggestionLine
		end := *endLine
		if end == 0 {
			end = *line
		}
		selection = &comment.Selection{
			StartLine: *line,
			EndLine:   end,
			Original:  *original,
		}

	case "char-range":
		if *byteOffset == 0 && *line == 0 {
			fmt.Println("Error: --offset or --line flag is required for char-range suggestions")
			os.Exit(1)
		}
		sugType = comment.SuggestionCharRange
		offset := *byteOffset
		if offset == 0 && *line > 0 {
			// Calculate offset from line number
			offset = comment.CalculateByteOffset(doc.Content, *line, 0)
		}
		selection = &comment.Selection{
			ByteOffset: offset,
			Length:     *length,
			Original:   *original,
		}

	case "multi-line":
		if *startLine == 0 || *endLine == 0 {
			fmt.Println("Error: --start-line and --end-line flags are required for multi-line suggestions")
			os.Exit(1)
		}
		sugType = comment.SuggestionMultiLine
		selection = &comment.Selection{
			StartLine: *startLine,
			EndLine:   *endLine,
			Original:  *original,
		}

	case "diff-hunk":
		if *line == 0 {
			fmt.Println("Error: --line flag is required for diff-hunk suggestions")
			os.Exit(1)
		}
		sugType = comment.SuggestionDiffHunk
		selection = &comment.Selection{
			StartLine: *line,
		}

	default:
		fmt.Printf("Error: Unknown suggestion type '%s'. Valid types: line, char-range, multi-line, diff-hunk\n", *suggestionType)
		os.Exit(1)
	}

	// Create suggestion comment
	suggestion := comment.NewComment(*author, selection.StartLine, *text)
	suggestion.SuggestionType = sugType
	suggestion.Selection = selection
	suggestion.ProposedText = *proposed
	suggestion.AcceptanceState = comment.AcceptancePending
	suggestion.Type = "S" // Mark as suggestion type

	// Add to document
	doc.Comments = append(doc.Comments, suggestion)
	doc.Positions[suggestion.ID] = comment.Position{Line: selection.StartLine}

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Suggestion added to line %d by @%s\n", selection.StartLine, *author)
	fmt.Printf("  Suggestion ID: %s\n", suggestion.ID)
	fmt.Printf("  Type: %s\n", sugType)
}

func acceptCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("accept", flag.ExitOnError)
	suggestionID := fs.String("suggestion", "", "Suggestion ID (required)")
	preview := fs.Bool("preview", false, "Preview changes without applying")

	fs.Parse(args)

	if *suggestionID == "" {
		fmt.Println("Error: --suggestion flag is required")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Find suggestion
	var suggestion *comment.Comment
	for _, c := range doc.Comments {
		if c.ID == *suggestionID {
			suggestion = c
			break
		}
	}

	if suggestion == nil {
		fmt.Printf("Error: Suggestion '%s' not found\n", *suggestionID)
		os.Exit(1)
	}

	if !suggestion.IsSuggestion() {
		fmt.Printf("Error: Comment '%s' is not a suggestion\n", *suggestionID)
		os.Exit(1)
	}

	// Preview if requested
	if *preview {
		newContent, err := comment.ApplySuggestion(doc.Content, suggestion)
		if err != nil {
			fmt.Printf("Error applying suggestion: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Preview of changes:")
		fmt.Println("==================")
		fmt.Println(newContent)
		return
	}

	// Apply suggestion
	newContent, err := comment.ApplySuggestion(doc.Content, suggestion)
	if err != nil {
		fmt.Printf("Error applying suggestion: %v\n", err)
		os.Exit(1)
	}

	// Update document content
	doc.Content = newContent

	// Mark suggestion as accepted
	suggestion.AcceptanceState = comment.AcceptanceAccepted

	// Recalculate positions
	comment.RecalculatePositionsAfterEdit(
		suggestion.Selection.StartLine,
		suggestion.Selection.EndLine,
		len(strings.Split(suggestion.ProposedText, "\n")),
		doc.Positions,
	)
	comment.UpdatePositionsByteOffsets(doc.Content, doc.Positions)

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Suggestion %s accepted and applied\n", *suggestionID)
}

func rejectCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("reject", flag.ExitOnError)
	suggestionID := fs.String("suggestion", "", "Suggestion ID (required)")

	fs.Parse(args)

	if *suggestionID == "" {
		fmt.Println("Error: --suggestion flag is required")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Find suggestion
	var suggestion *comment.Comment
	for _, c := range doc.Comments {
		if c.ID == *suggestionID {
			suggestion = c
			break
		}
	}

	if suggestion == nil {
		fmt.Printf("Error: Suggestion '%s' not found\n", *suggestionID)
		os.Exit(1)
	}

	if !suggestion.IsSuggestion() {
		fmt.Printf("Error: Comment '%s' is not a suggestion\n", *suggestionID)
		os.Exit(1)
	}

	// Mark suggestion as rejected
	suggestion.AcceptanceState = comment.AcceptanceRejected

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Suggestion %s rejected\n", *suggestionID)
}

func batchAcceptCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("batch-accept", flag.ExitOnError)
	jsonInput := fs.String("json", "", "JSON file path (use '-' for stdin)")
	filterAuthor := fs.String("author", "", "Accept all suggestions by author")
	filterType := fs.String("type", "", "Accept all suggestions of type (S)")
	checkConflicts := fs.Bool("check-conflicts", true, "Check for conflicts before applying")

	fs.Parse(args)

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	var suggestionsToAccept []*comment.Comment

	// Get suggestions from JSON or filters
	if *jsonInput != "" {
		// Read JSON list of suggestion IDs
		var input []byte
		if *jsonInput == "-" {
			input, err = io.ReadAll(os.Stdin)
		} else {
			input, err = os.ReadFile(*jsonInput)
		}
		if err != nil {
			fmt.Printf("Error reading JSON: %v\n", err)
			os.Exit(1)
		}

		var suggestionIDs []string
		if err := json.Unmarshal(input, &suggestionIDs); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			os.Exit(1)
		}

		// Find suggestions by ID
		for _, id := range suggestionIDs {
			for _, c := range doc.Comments {
				if c.ID == id && c.IsSuggestion() && c.IsPending() {
					suggestionsToAccept = append(suggestionsToAccept, c)
					break
				}
			}
		}
	} else if *filterAuthor != "" || *filterType != "" {
		// Filter by author or type
		for _, c := range doc.Comments {
			if !c.IsSuggestion() || !c.IsPending() {
				continue
			}
			if *filterAuthor != "" && c.Author != *filterAuthor {
				continue
			}
			if *filterType != "" && c.Type != *filterType {
				continue
			}
			suggestionsToAccept = append(suggestionsToAccept, c)
		}
	} else {
		fmt.Println("Error: --json, --author, or --type flag is required")
		os.Exit(1)
	}

	if len(suggestionsToAccept) == 0 {
		fmt.Println("No pending suggestions found matching criteria")
		os.Exit(0)
	}

	// Check for conflicts
	if *checkConflicts {
		conflicts := comment.DetectConflicts(suggestionsToAccept)
		if comment.HasConflicts(conflicts) {
			fmt.Printf("⚠ Warning: Found %d conflicts between suggestions\n", len(conflicts))
			for i, conflict := range conflicts {
				if conflict.Type == comment.ConflictNone || conflict.Type == comment.ConflictAdjacent {
					continue
				}
				fmt.Printf("  %d. %s: %s vs %s - %s\n", i+1, conflict.Type,
					conflict.Suggestion1.ID, conflict.Suggestion2.ID, conflict.Description)
			}
			fmt.Println("\nFiltering to non-conflicting suggestions...")
			suggestionsToAccept = comment.FilterNonConflicting(suggestionsToAccept)
			fmt.Printf("Proceeding with %d non-conflicting suggestions\n", len(suggestionsToAccept))
		}
	}

	// Sort bottom-to-top for safe application
	comment.SortSuggestionsByPosition(suggestionsToAccept)

	// Apply all suggestions
	newContent, applied, err := comment.ApplyMultipleSuggestions(doc.Content, suggestionsToAccept)
	if err != nil {
		fmt.Printf("Error applying suggestions: %v\n", err)
		fmt.Printf("Successfully applied %d of %d suggestions before error\n", len(applied), len(suggestionsToAccept))
		os.Exit(1)
	}

	// Update document
	doc.Content = newContent

	// Mark all applied suggestions as accepted
	for _, id := range applied {
		for _, c := range doc.Comments {
			if c.ID == id {
				c.AcceptanceState = comment.AcceptanceAccepted
				break
			}
		}
	}

	// Recalculate all positions
	comment.UpdatePositionsByteOffsets(doc.Content, doc.Positions)

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Accepted and applied %d suggestions\n", len(applied))
	for i, id := range applied {
		fmt.Printf("  %d. %s\n", i+1, id)
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
  batch-add <file> [flags]    Add multiple comments from JSON
  reply <file> [flags]        Reply to a comment thread
  batch-reply <file> [flags]  Reply to multiple threads from JSON
  resolve <file> [flags]      Mark a thread as resolved
  suggest <file> [flags]      Add an edit suggestion to a specific line
  accept <file> [flags]       Accept a suggestion and apply changes
  reject <file> [flags]       Reject a suggestion
  batch-accept <file> [flags] Accept multiple suggestions at once
  export <file> [flags]       Export comments to JSON format
  publish <file> [flags]      Output clean markdown without comments
  help                        Show this help message

List Command Flags:
  --type <type>               Filter by comment type: Q, S, B, T, E
  --resolved                  Show resolved comments (default: false, only shows unresolved)
  --author <name>             Filter by author name
  --search <text>             Search comment text (case-insensitive)
  --line-range <range>        Filter by line range (e.g., 10-30)
  --sort <field>              Sort by: line (default), timestamp, author
  --format <format>           Output format: text (default), json, table

Add Command Flags:
  --line <number>             Line number (required)
  --text <text>               Comment text (required)
  --author <name>             Author name (required)
  --type <type>               Comment type: Q, S, B, T, E (auto-prefixes text)

Batch-Add Command Flags:
  --json <file|->             JSON file path or '-' for stdin (required)
                              Note: Each comment in JSON must include "author" field

Reply Command Flags:
  --thread <id>               Thread ID (required)
  --text <text>               Reply text (required)
  --author <name>             Author name (required)

Batch-Reply Command Flags:
  --json <file|->             JSON file path or '-' for stdin (required)
                              Note: Each reply in JSON must include "thread" and "author" fields

Resolve Command Flags:
  --thread <id>               Thread ID (required)

Suggest Command Flags:
  --line <number>             Line number (required for line/diff-hunk types)
  --author <name>             Author name (required)
  --text <text>               Suggestion description (required)
  --type <type>               Suggestion type: line (default), char-range, multi-line, diff-hunk
  --original <text>           Original text to replace (required)
  --proposed <text>           Proposed replacement text (required)
  --start-line <number>       Start line (for multi-line type)
  --end-line <number>         End line (for multi-line type)
  --offset <number>           Byte offset (for char-range type)
  --length <number>           Length in bytes (for char-range type)

Accept Command Flags:
  --suggestion <id>           Suggestion ID (required)
  --preview                   Preview changes without applying

Reject Command Flags:
  --suggestion <id>           Suggestion ID (required)

Batch-Accept Command Flags:
  --json <file|->             JSON file path or '-' for stdin (suggestion IDs)
  --author <name>             Accept all suggestions from this author
  --type <type>               Accept all suggestions of this type
  --check-conflicts           Check for conflicts before accepting (default: true)

Export Command Flags:
  --format <format>           Export format: json (default: json)
  --output <file>             Output file (default: stdout)

Publish Command Flags:
  --output <file>             Output file (default: stdout)

Examples:
  # Interactive mode
  comments view document.md

  # List with filters (can combine multiple filters!)
  comments list document.md                              # Show only unresolved comments
  comments list document.md --resolved                   # Show all comments (including resolved)
  comments list document.md --type Q                     # Show only unresolved questions
  comments list document.md --author claude              # Show comments by claude
  comments list document.md --search "API"               # Search for "API" in comment text
  comments list document.md --line-range 10-50           # Comments between lines 10-50
  comments list document.md --author alice --type Q      # Alice's questions
  comments list document.md --format table               # Pretty table output
  comments list document.md --format json > output.json  # Export filtered results

  # Single comment (author required for CLI)
  comments add document.md --line 10 --author "claude" --text "This needs review"
  comments add document.md --line 15 --author "bot" --text "Great point!"
  comments add document.md --line 20 --author "reviewer" --type Q --text "Is this correct?"

  # Batch add comments from JSON (each comment must have author)
  comments batch-add document.md --json reviews.json
  echo '[{"line":10,"author":"claude","text":"Fix this"},{"line":20,"author":"bot","text":"Add example","type":"S"}]' | \
    comments batch-add document.md --json -

  # Thread operations (author required for CLI)
  comments reply document.md --thread c123 --author "claude" --text "I agree"
  comments batch-reply document.md --json replies.json
  echo '[{"thread":"c123","author":"claude","text":"LGTM"}]' | \
    comments batch-reply document.md --json -
  comments resolve document.md --thread c123

  # Suggestions - propose edits with track-changes workflow
  # Simple line suggestion
  comments suggest document.md --line 10 --author "editor" \
    --text "Simplify this sentence" \
    --original "The system allows users to collaborate effectively" \
    --proposed "The system enables collaboration"

  # Multi-line suggestion
  comments suggest document.md --type multi-line --start-line 5 --end-line 8 \
    --author "writer" --text "Restructure intro" \
    --original "Line 1\nLine 2\nLine 3\nLine 4" \
    --proposed "New line 1\nNew line 2"

  # Accept/reject suggestions
  comments accept document.md --suggestion c123 --preview  # Preview changes first
  comments accept document.md --suggestion c123            # Apply the changes
  comments reject document.md --suggestion c456            # Reject suggestion

  # Batch accept suggestions
  comments batch-accept document.md --author "copywriter"  # Accept all from author
  comments batch-accept document.md --type "line"          # Accept all line suggestions

  # Export comments for programmatic access
  comments export document.md                    # Print JSON to stdout
  comments export document.md --output comments.json  # Save to file

  # Publish clean markdown (strip all comments)
  comments publish document.md                   # Print to stdout
  comments publish document.md --output final.md # Save to file

Batch-Add JSON Format:
  [
    {
      "line": 10,
      "author": "alice",       // Required
      "text": "Add examples",
      "type": "S"              // Optional: Q, S, B, T, E
    },
    {
      "line": 25,
      "author": "bob",         // Required
      "text": "Great point!"
    }
  ]

Batch-Reply JSON Format:
  [
    {
      "thread": "c123",        // Required: Thread ID
      "author": "claude",      // Required
      "text": "This looks good to me"
    },
    {
      "thread": "c456",
      "author": "alice",
      "text": "I agree with this approach"
    }
  ]

Keyboard shortcuts (in view mode):
  j/k or ↓/↑      Navigate comments
  c               Enter line selection mode
  q or Ctrl+C     Quit

For more information, visit: https://github.com/rcliao/comments
`
	fmt.Print(usage)
}
