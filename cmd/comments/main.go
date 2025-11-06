package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/markdown"
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

	case "get":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments get <file> [flags]")
			os.Exit(1)
		}
		getCommand(os.Args[2], os.Args[3:])

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

	case "status":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments status <file> [flags]")
			os.Exit(1)
		}
		statusCommand(os.Args[2], os.Args[3:])

	case "reattach":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments reattach <file> [flags]")
			os.Exit(1)
		}
		reattachCommand(os.Args[2], os.Args[3:])

	case "cleanup":
		if len(os.Args) < 3 {
			fmt.Println("Usage: comments cleanup <file> [flags]")
			os.Exit(1)
		}
		cleanupCommand(os.Args[2], os.Args[3:])

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
	sectionFilter := fs.String("section", "", "Filter by section path (includes nested sections)")
	statusFilter := fs.String("status", "", "Filter by status: active, orphaned, resolved, completed")
	priorityFilter := fs.String("priority", "", "Filter by priority: low, medium, high")
	sortBy := fs.String("sort", "line", "Sort by: line, timestamp, author")
	format := fs.String("format", "text", "Output format: text, json, table")
	withContext := fs.Bool("with-context", false, "Include document context for each comment")

	fs.Parse(args)

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Compute section metadata for all comments if not already present
	comment.ComputeSectionsForComments(doc)

	// Filter by resolved status (only show root comments based on resolved flag)
	filteredComments := comment.GetVisibleComments(doc.Threads, *showResolved)

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

	// Apply section filter
	if *sectionFilter != "" {
		// Validate section exists
		if err := comment.ValidateSectionPath(doc.Content, *sectionFilter); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Get all comments in this section (including nested sections)
		sectionComments := comment.GetCommentsInSection(doc, *sectionFilter)

		// Intersect with filtered comments (preserve other filters)
		commentSet := make(map[string]bool)
		for _, c := range sectionComments {
			commentSet[c.ID] = true
		}

		filtered := []*comment.Comment{}
		for _, c := range filteredComments {
			if commentSet[c.ID] {
				filtered = append(filtered, c)
			}
		}
		filteredComments = filtered
	}

	// Apply status filter
	if *statusFilter != "" {
		filtered := []*comment.Comment{}
		for _, c := range filteredComments {
			if c.GetStatus() == *statusFilter {
				filtered = append(filtered, c)
			}
		}
		filteredComments = filtered
	}

	// Apply priority filter
	if *priorityFilter != "" {
		filtered := []*comment.Comment{}
		for _, c := range filteredComments {
			if c.GetPriority() == *priorityFilter {
				filtered = append(filtered, c)
			}
		}
		filteredComments = filtered
	}

	// Sort comments
	sortComments(filteredComments, *sortBy)

	// Output based on format
	switch *format {
	case "json":
		if err := outputJSON(filteredComments, doc.Threads, doc.Content, *withContext); err != nil {
			fmt.Printf("Error outputting JSON: %v\n", err)
			os.Exit(1)
		}
		return

	case "table":
		outputTable(filteredComments, doc.Threads)
		return

	case "text":
		// If --with-context is specified with text format, use context format
		if *withContext {
			output := formatListWithContext(filteredComments, doc.Content)
			fmt.Print(output)
			return
		}
		// Original text format (below)

	default:
		fmt.Printf("Error: Unknown format '%s'. Valid formats: text, json, table\n", *format)
		os.Exit(1)
	}

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
	if *sectionFilter != "" {
		filterDesc += fmt.Sprintf(" in section '%s'", *sectionFilter)
	}
	if *statusFilter != "" {
		filterDesc += fmt.Sprintf(" with status [%s]", *statusFilter)
	}
	if *priorityFilter != "" {
		filterDesc += fmt.Sprintf(" with priority [%s]", *priorityFilter)
	}

	fmt.Printf("Found %d %s thread(s)%s in %s\n\n", len(filteredComments), statusText, filterDesc, filename)

	for i, thread := range filteredComments {
		// Build location string (show section path if available, otherwise just line)
		locationStr := fmt.Sprintf("Line %d", thread.Line)
		if thread.SectionPath != "" {
			locationStr = fmt.Sprintf("%s (Line %d)", thread.SectionPath, thread.Line)
		}

		// Priority indicator
		priorityIndicator := ""
		switch thread.GetPriority() {
		case "high":
			priorityIndicator = " [HIGH]"
		case "low":
			priorityIndicator = " [LOW]"
		// medium is default, no indicator needed
		}

		// Status indicator
		statusIndicator := ""
		status := thread.GetStatus()
		if status == "orphaned" {
			statusIndicator = " ⚠️  ORPHANED"
			if thread.OrphanedReason != "" {
				statusIndicator += fmt.Sprintf(" (%s)", thread.OrphanedReason)
			}
		} else if status == "completed" {
			statusIndicator = " ✓ COMPLETED"
		}

		// Show thread info with priority and status
		fmt.Printf("[%d] %s • @%s • %s%s%s\n", i+1, locationStr, thread.Author, thread.Timestamp.Format("2006-01-02 15:04"), priorityIndicator, statusIndicator)
		fmt.Printf("    Type: Root | Thread ID: %s | Status: %s\n", thread.ID, thread.GetStatus())

		// Show reply count and resolved status
		replyCount := thread.CountReplies()
		resolvedStatus := ""
		if thread.Resolved {
			resolvedStatus = " [RESOLVED]"
		}
		fmt.Printf("    Replies: %d%s\n", replyCount, resolvedStatus)

		fmt.Printf("    %s\n\n", thread.Text)
	}
}

func getCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	threadID := fs.String("thread", "", "Thread ID to get (required)")
	withReplies := fs.Bool("with-replies", true, "Include replies in output (default: true)")

	fs.Parse(args)

	if *threadID == "" {
		fmt.Println("Error: --thread flag is required")
		fmt.Println("Usage: comments get <file> --thread <thread-id>")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Compute section metadata for all comments if not already present
	comment.ComputeSectionsForComments(doc)

	// Find the thread
	var foundComment *comment.Comment
	for _, thread := range doc.Threads {
		if thread.ID == *threadID {
			foundComment = thread
			break
		}
		// Also search in replies
		for _, reply := range thread.Replies {
			if reply.ID == *threadID {
				foundComment = reply
				break
			}
		}
		if foundComment != nil {
			break
		}
	}

	if foundComment == nil {
		fmt.Printf("Error: Thread with ID '%s' not found\n", *threadID)
		fmt.Println("\nAvailable threads:")
		for i, thread := range doc.Threads {
			fmt.Printf("  [%d] %s (Line %d) - @%s\n", i+1, thread.ID, thread.Line, thread.Author)
		}
		os.Exit(1)
	}

	// Get context and format output
	ctx := getCommentContext(foundComment, doc.Content)
	output := formatCommentWithContext(foundComment, ctx, *withReplies)

	fmt.Print(output)
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
	line := fs.Int("line", 0, "Line number (use either --line or --section)")
	section := fs.String("section", "", "Section path (use either --line or --section)")
	author := fs.String("author", "", "Author name (required)")
	commentType := fs.String("type", "", "Comment type: Q, S, B, T, E (auto-prefixes text)")
	priority := fs.String("priority", "medium", "Priority: low, medium, high (default: medium)")

	fs.Parse(args)

	if *text == "" {
		fmt.Println("Error: --text flag is required")
		fmt.Println("Usage: comments add <file> --line N --author \"name\" --text \"your comment\"")
		fmt.Println("   or: comments add <file> --section \"Section Path\" --author \"name\" --text \"your comment\"")
		os.Exit(1)
	}

	if *author == "" {
		fmt.Println("Error: --author flag is required")
		fmt.Println("Usage: comments add <file> --line N --author \"name\" --text \"your comment\"")
		fmt.Println("   or: comments add <file> --section \"Section Path\" --author \"name\" --text \"your comment\"")
		os.Exit(1)
	}

	// Validate that either line or section is provided (but not both)
	if *line == 0 && *section == "" {
		fmt.Println("Error: either --line or --section flag is required")
		fmt.Println("Usage: comments add <file> --line N --author \"name\" --text \"your comment\"")
		fmt.Println("   or: comments add <file> --section \"Section Path\" --author \"name\" --text \"your comment\"")
		os.Exit(1)
	}

	if *line != 0 && *section != "" {
		fmt.Println("Error: cannot specify both --line and --section")
		fmt.Println("Use either --line N or --section \"Section Path\", not both")
		os.Exit(1)
	}

	// Resolve text input (supports @filename)
	resolvedText, err := resolveTextInput(*text)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Auto-prefix text with type if specified
	commentText := resolvedText
	if *commentType != "" {
		commentText = "[" + *commentType + "] " + resolvedText
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Determine the line number to use
	targetLine := *line
	if *section != "" {
		// Validate section exists
		if err := comment.ValidateSectionPath(doc.Content, *section); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Resolve section to line number (use section start line)
		startLine, _, err := comment.ResolveSectionToLines(doc.Content, *section, false)
		if err != nil {
			fmt.Printf("Error resolving section: %v\n", err)
			os.Exit(1)
		}
		targetLine = startLine
	}

	// Create new comment with type metadata
	var newComment *comment.Comment
	if *commentType != "" {
		newComment = comment.NewCommentWithType(*author, targetLine, commentText, *commentType)
	} else {
		newComment = comment.NewComment(*author, targetLine, commentText)
	}

	// Set priority
	newComment.Priority = *priority
	newComment.Status = "active"

	// Compute section metadata for the new comment
	comment.UpdateCommentSection(newComment, doc.Content)

	doc.Threads = append(doc.Threads, newComment)

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	// Display success message
	if newComment.SectionPath != "" {
		fmt.Printf("✓ Comment added to %s (Line %d) by @%s\n", newComment.SectionPath, targetLine, *author)
	} else {
		fmt.Printf("✓ Comment added to line %d by @%s\n", targetLine, *author)
	}
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

	// Resolve text input (supports @filename)
	resolvedText, err := resolveTextInput(*text)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Add reply to thread using helper
	if err := comment.AddReplyToThread(doc.Threads, *thread, *author, resolvedText); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nAvailable threads:")
		for _, t := range doc.Threads {
			fmt.Printf("  %s (Line %d, %d replies)\n", t.ID, t.Line, t.CountReplies())
		}
		os.Exit(1)
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Reply added to thread %s by @%s\n", *thread, *author)
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

	// Resolve the thread
	if err := comment.ResolveThread(doc.Threads, *thread); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nAvailable threads:")
		for _, t := range doc.Threads {
			fmt.Printf("  %s (Line %d, %d replies)\n", t.ID, t.Line, t.CountReplies())
		}
		os.Exit(1)
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Thread %s marked as resolved\n", *thread)
}

func suggestCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("suggest", flag.ExitOnError)
	startLine := fs.Int("start-line", 0, "Start line (use either line range or section)")
	endLine := fs.Int("end-line", 0, "End line (use either line range or section)")
	section := fs.String("section", "", "Section path (use either line range or section)")
	author := fs.String("author", "", "Author name (required)")
	text := fs.String("text", "", "Suggestion description (required)")
	original := fs.String("original", "", "Original text to replace")
	proposed := fs.String("proposed", "", "Proposed replacement text (required)")

	fs.Parse(args)

	// Validate required flags
	if *author == "" {
		fmt.Println("Error: --author flag is required")
		fmt.Println("Usage: comments suggest <file> --start-line N --end-line M --author \"name\" --text \"desc\" --proposed \"new text\"")
		fmt.Println("   or: comments suggest <file> --section \"Section Path\" --author \"name\" --text \"desc\" --proposed \"new text\"")
		os.Exit(1)
	}
	if *text == "" {
		fmt.Println("Error: --text flag is required")
		fmt.Println("Usage: comments suggest <file> --start-line N --end-line M --author \"name\" --text \"desc\" --proposed \"new text\"")
		fmt.Println("   or: comments suggest <file> --section \"Section Path\" --author \"name\" --text \"desc\" --proposed \"new text\"")
		os.Exit(1)
	}
	if *proposed == "" {
		fmt.Println("Error: --proposed flag is required")
		fmt.Println("Usage: comments suggest <file> --start-line N --end-line M --author \"name\" --text \"desc\" --proposed \"new text\"")
		fmt.Println("   or: comments suggest <file> --section \"Section Path\" --author \"name\" --text \"desc\" --proposed \"new text\"")
		os.Exit(1)
	}

	// Validate that either line range or section is provided (but not both)
	if *startLine == 0 && *section == "" {
		fmt.Println("Error: either --start-line/--end-line or --section flag is required")
		fmt.Println("Usage: comments suggest <file> --start-line N --end-line M --author \"name\" --text \"desc\" --proposed \"new text\"")
		fmt.Println("   or: comments suggest <file> --section \"Section Path\" --author \"name\" --text \"desc\" --proposed \"new text\"")
		os.Exit(1)
	}

	if *startLine != 0 && *section != "" {
		fmt.Println("Error: cannot specify both line range and section")
		fmt.Println("Use either --start-line/--end-line or --section, not both")
		os.Exit(1)
	}

	// Resolve text inputs (supports @filename)
	resolvedText, err := resolveTextInput(*text)
	if err != nil {
		fmt.Printf("Error resolving --text: %v\n", err)
		os.Exit(1)
	}

	resolvedOriginal, err := resolveTextInput(*original)
	if err != nil {
		fmt.Printf("Error resolving --original: %v\n", err)
		os.Exit(1)
	}

	resolvedProposed, err := resolveTextInput(*proposed)
	if err != nil {
		fmt.Printf("Error resolving --proposed: %v\n", err)
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Determine the line range to use
	targetStartLine := *startLine
	targetEndLine := *endLine
	if *section != "" {
		// Validate section exists
		if err := comment.ValidateSectionPath(doc.Content, *section); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Resolve section to line range
		start, end, err := comment.ResolveSectionToLines(doc.Content, *section, false)
		if err != nil {
			fmt.Printf("Error resolving section: %v\n", err)
			os.Exit(1)
		}
		targetStartLine = start
		targetEndLine = end
	}

	// Validate line range
	if targetEndLine == 0 {
		targetEndLine = targetStartLine
	}
	if targetStartLine > targetEndLine {
		fmt.Printf("Error: start line (%d) must be <= end line (%d)\n", targetStartLine, targetEndLine)
		os.Exit(1)
	}

	// Create suggestion using helper
	suggestion := comment.NewSuggestion(*author, targetStartLine, targetEndLine, resolvedText, resolvedOriginal, resolvedProposed)

	// Compute section metadata
	comment.UpdateCommentSection(suggestion, doc.Content)

	// Add to document
	doc.Threads = append(doc.Threads, suggestion)

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	if suggestion.SectionPath != "" {
		fmt.Printf("✓ Suggestion added to %s (Lines %d-%d) by @%s\n", suggestion.SectionPath, targetStartLine, targetEndLine, *author)
	} else {
		fmt.Printf("✓ Suggestion added to lines %d-%d by @%s\n", targetStartLine, targetEndLine, *author)
	}
	fmt.Printf("  Suggestion ID: %s\n", suggestion.ID)
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

	// Find suggestion in all comments (threads + replies)
	allComments := doc.GetAllComments()
	var suggestion *comment.Comment
	for _, c := range allComments {
		if c.ID == *suggestionID {
			suggestion = c
			break
		}
	}

	if suggestion == nil {
		fmt.Printf("Error: Suggestion '%s' not found\n", *suggestionID)
		os.Exit(1)
	}

	if !suggestion.IsSuggestion {
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

	// Mark suggestion as accepted using helper
	if err := comment.AcceptSuggestion(doc.Threads, *suggestionID); err != nil {
		fmt.Printf("Error marking suggestion as accepted: %v\n", err)
		os.Exit(1)
	}

	// Recalculate comment line numbers (line-only tracking)
	linesAdded := len(strings.Split(suggestion.ProposedText, "\n"))
	comment.RecalculateCommentLines(doc.Threads, suggestion.StartLine, suggestion.EndLine, linesAdded)

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

	// Mark suggestion as rejected using helper
	if err := comment.RejectSuggestion(doc.Threads, *suggestionID); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

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
	filterAuthor := fs.String("author", "", "Accept all suggestions by author")

	fs.Parse(args)

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	var suggestionsToAccept []*comment.Comment

	// Get suggestions by author filter
	if *filterAuthor != "" {
		suggestionsToAccept = comment.GetSuggestionsByAuthor(doc.Threads, *filterAuthor)
	} else {
		// Get all pending suggestions
		suggestionsToAccept = comment.GetPendingSuggestions(doc.Threads)
	}

	if len(suggestionsToAccept) == 0 {
		fmt.Println("No pending suggestions found matching criteria")
		os.Exit(0)
	}

	fmt.Printf("Found %d pending suggestion(s) to accept\n", len(suggestionsToAccept))

	// Apply each suggestion sequentially
	acceptedCount := 0
	for _, suggestion := range suggestionsToAccept {
		// Apply suggestion
		newContent, err := comment.ApplySuggestion(doc.Content, suggestion)
		if err != nil {
			fmt.Printf("⚠ Warning: Failed to apply suggestion %s: %v\n", suggestion.ID, err)
			continue
		}

		// Update document content
		doc.Content = newContent

		// Mark as accepted
		if err := comment.AcceptSuggestion(doc.Threads, suggestion.ID); err != nil {
			fmt.Printf("⚠ Warning: Failed to mark suggestion %s as accepted: %v\n", suggestion.ID, err)
			continue
		}

		// Recalculate comment lines after this edit
		linesAdded := len(strings.Split(suggestion.ProposedText, "\n"))
		comment.RecalculateCommentLines(doc.Threads, suggestion.StartLine, suggestion.EndLine, linesAdded)

		acceptedCount++
		fmt.Printf("  ✓ Accepted and applied %s\n", suggestion.ID)
	}

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Successfully accepted and applied %d of %d suggestions\n", acceptedCount, len(suggestionsToAccept))
}

func statusCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	commentID := fs.String("comment", "", "Comment ID to update (required)")
	newStatus := fs.String("status", "", "New status: active, orphaned, resolved, completed (required)")

	fs.Parse(args)

	if *commentID == "" {
		fmt.Println("Error: --comment flag is required")
		fmt.Println("Usage: comments status <file> --comment <id> --status <status>")
		os.Exit(1)
	}

	if *newStatus == "" {
		fmt.Println("Error: --status flag is required")
		fmt.Println("Valid statuses: active, orphaned, resolved, completed")
		os.Exit(1)
	}

	// Validate status value
	validStatuses := map[string]bool{
		"active":    true,
		"orphaned":  true,
		"resolved":  true,
		"completed": true,
	}
	if !validStatuses[*newStatus] {
		fmt.Printf("Error: Invalid status '%s'\n", *newStatus)
		fmt.Println("Valid statuses: active, orphaned, resolved, completed")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Find the comment
	foundComment := doc.FindCommentByID(*commentID)
	if foundComment == nil {
		fmt.Printf("Error: Comment '%s' not found\n", *commentID)
		os.Exit(1)
	}

	// Update status
	oldStatus := foundComment.GetStatus()
	foundComment.Status = *newStatus

	// If changing from orphaned to active, clear orphaned metadata
	if oldStatus == "orphaned" && *newStatus == "active" {
		foundComment.OrphanedReason = ""
		foundComment.OrphanedAt = nil
	}

	// Save changes
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving changes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated comment %s status: %s → %s\n", *commentID, oldStatus, *newStatus)
}

func reattachCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("reattach", flag.ExitOnError)
	commentID := fs.String("comment", "", "Comment ID to reattach (required)")
	newLine := fs.Int("line", 0, "New line number to attach to (required)")
	sectionPath := fs.String("section", "", "Section path to attach to (alternative to --line)")

	fs.Parse(args)

	if *commentID == "" {
		fmt.Println("Error: --comment flag is required")
		fmt.Println("Usage: comments reattach <file> --comment <id> --line <num>")
		fmt.Println("   or: comments reattach <file> --comment <id> --section <path>")
		os.Exit(1)
	}

	if *newLine == 0 && *sectionPath == "" {
		fmt.Println("Error: either --line or --section flag is required")
		os.Exit(1)
	}

	if *newLine != 0 && *sectionPath != "" {
		fmt.Println("Error: cannot specify both --line and --section")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Find the comment
	foundComment := doc.FindCommentByID(*commentID)
	if foundComment == nil {
		fmt.Printf("Error: Comment '%s' not found\n", *commentID)
		os.Exit(1)
	}

	// Determine target line
	targetLine := *newLine
	if *sectionPath != "" {
		// Find section
		docStructure := markdown.ParseDocument(doc.Content)
		section := docStructure.FindSection(*sectionPath)
		if section == nil {
			fmt.Printf("Error: Section '%s' not found\n", *sectionPath)
			fmt.Println("\nAvailable sections:")
			for _, sec := range docStructure.Sections {
				fmt.Printf("  - %s\n", sec.GetFullPath(docStructure.SectionsByID))
			}
			os.Exit(1)
		}
		targetLine = section.StartLine
		foundComment.SectionID = section.ID
		foundComment.SectionPath = section.GetFullPath(docStructure.SectionsByID)
	}

	// Validate line number
	lines := strings.Split(doc.Content, "\n")
	if targetLine < 1 || targetLine > len(lines) {
		fmt.Printf("Error: Line %d out of bounds (document has %d lines)\n", targetLine, len(lines))
		os.Exit(1)
	}

	// Reattach comment
	oldLine := foundComment.Line
	foundComment.Line = targetLine
	foundComment.Status = "active"
	foundComment.OrphanedReason = ""
	foundComment.OrphanedAt = nil

	// Save changes
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving changes: %v\n", err)
		os.Exit(1)
	}

	locationStr := fmt.Sprintf("line %d", targetLine)
	if *sectionPath != "" {
		locationStr = fmt.Sprintf("section '%s' (line %d)", *sectionPath, targetLine)
	}
	fmt.Printf("Reattached comment %s: line %d → %s\n", *commentID, oldLine, locationStr)
}

func cleanupCommand(filename string, args []string) {
	// Parse flags
	fs := flag.NewFlagSet("cleanup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Show what would be cleaned up without actually doing it")
	statusFilter := fs.String("status", "completed", "Status to clean up (completed or resolved)")

	fs.Parse(args)

	// Validate status
	if *statusFilter != "completed" && *statusFilter != "resolved" {
		fmt.Println("Error: --status must be 'completed' or 'resolved'")
		os.Exit(1)
	}

	// Load document
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	// Find comments to clean up
	var toCleanup []*comment.Comment
	for _, c := range doc.GetAllComments() {
		if c.GetStatus() == *statusFilter {
			toCleanup = append(toCleanup, c)
		}
	}

	if len(toCleanup) == 0 {
		fmt.Printf("No %s comments to clean up\n", *statusFilter)
		return
	}

	// Show what will be cleaned up
	fmt.Printf("Found %d %s comment(s) to clean up:\n\n", len(toCleanup), *statusFilter)
	for i, c := range toCleanup {
		fmt.Printf("[%d] %s • @%s • Line %d\n", i+1, c.ID, c.Author, c.Line)
		fmt.Printf("    %s\n\n", c.Text)
	}

	if *dryRun {
		fmt.Println("Dry run - no changes made")
		return
	}

	// Archive to separate file
	sidecarPath := comment.GetSidecarPath(filename)
	archivePath := strings.TrimSuffix(sidecarPath, ".json") + fmt.Sprintf(".archived.%s", *statusFilter)

	// Load or create archive
	var archive comment.StorageFormat
	if archiveData, err := os.ReadFile(archivePath); err == nil {
		json.Unmarshal(archiveData, &archive)
	} else {
		archive = comment.StorageFormat{
			Version:       comment.StorageVersion,
			Threads:       []*comment.Comment{},
			DocumentHash:  doc.DocumentHash,
			LastValidated: doc.LastValidated,
		}
	}

	// Remove from doc.Threads and add to archive
	cleanupIDs := make(map[string]bool)
	for _, c := range toCleanup {
		cleanupIDs[c.ID] = true
		// Only add root comments to archive (replies are nested)
		for _, thread := range doc.Threads {
			if thread.ID == c.ID {
				archive.Threads = append(archive.Threads, thread)
				break
			}
		}
	}

	// Filter out cleaned up threads
	var remainingThreads []*comment.Comment
	for _, thread := range doc.Threads {
		if !cleanupIDs[thread.ID] {
			remainingThreads = append(remainingThreads, thread)
		}
	}
	doc.Threads = remainingThreads

	// Save updated sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving sidecar: %v\n", err)
		os.Exit(1)
	}

	// Save archive
	archiveBytes, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		fmt.Printf("Error creating archive: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(archivePath, archiveBytes, 0644); err != nil {
		fmt.Printf("Error writing archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Cleaned up %d %s comment(s)\n", len(toCleanup), *statusFilter)
	fmt.Printf("✓ Archived to: %s\n", archivePath)
}

func printUsage() {
	usage := `comments - CLI tool for collaborative document commenting

Usage:
  comments <command> [arguments]

Commands:
  view <file>                 Open interactive TUI viewer
  list <file> [flags]         List all comments in a file
  get <file> [flags]          Get detailed comment with context
  add <file> [flags]          Add a comment to a specific line
  batch-add <file> [flags]    Add multiple comments from JSON
  reply <file> [flags]        Reply to a comment thread
  batch-reply <file> [flags]  Reply to multiple threads from JSON
  resolve <file> [flags]      Mark a thread as resolved
  suggest <file> [flags]      Add an edit suggestion to a specific line
  accept <file> [flags]       Accept a suggestion and apply changes
  reject <file> [flags]       Reject a suggestion
  batch-accept <file> [flags] Accept multiple suggestions at once
  status <file> [flags]       Update comment status (active/orphaned/resolved/completed)
  reattach <file> [flags]     Reattach an orphaned comment to a new line/section
  cleanup <file> [flags]      Archive completed/resolved comments
  export <file> [flags]       Export comments to JSON format
  publish <file> [flags]      Output clean markdown without comments
  help                        Show this help message

List Command Flags:
  --type <type>               Filter by comment type: Q, S, B, T, E
  --resolved                  Show resolved comments (default: false, only shows unresolved)
  --author <name>             Filter by author name
  --search <text>             Search comment text (case-insensitive)
  --line-range <range>        Filter by line range (e.g., 10-30)
  --section <path>            Filter by section path (includes nested sections)
  --status <status>           Filter by status: active, orphaned, resolved, completed
  --priority <priority>       Filter by priority: low, medium, high
  --sort <field>              Sort by: line (default), timestamp, author, priority
  --format <format>           Output format: text (default), json, table
  --with-context              Include document context for each comment

Get Command Flags:
  --thread <id>               Thread ID to retrieve (required)
  --with-replies              Include replies in output (default: true)

Add Command Flags:
  --line <number>             Line number (use either --line or --section)
  --section <path>            Section path (use either --line or --section)
  --text <text>               Comment text (required, supports @filename)
  --author <name>             Author name (required)
  --type <type>               Comment type: Q, S, B, T, E (auto-prefixes text)
  --priority <priority>       Priority: low, medium, high (default: medium)

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

Status Command Flags:
  --comment <id>              Comment ID to update (required)
  --status <status>           New status: active, orphaned, resolved, completed (required)

Reattach Command Flags:
  --comment <id>              Comment ID to reattach (required)
  --line <number>             New line number (use either --line or --section)
  --section <path>            Section path (use either --line or --section)

Cleanup Command Flags:
  --status <status>           Status to clean up: completed (default) or resolved
  --dry-run                   Preview what would be cleaned up without doing it

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
  comments list document.md --with-context               # Show all comments with document context
  comments list document.md --type Q --with-context      # Show questions with context (great for LLMs!)

  # Get detailed comment with context
  comments get document.md --thread c123                 # Get comment with full context
  comments get document.md --thread c456 --with-replies=false  # Get without replies

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

  # Status management - track TODOs and handle document changes
  comments list document.md --status orphaned              # View comments orphaned by edits
  comments list document.md --priority high                # View high-priority TODOs
  comments list document.md --status active --priority high # Active high-priority items
  comments status document.md --comment c123 --status completed  # Mark TODO as done
  comments reattach document.md --comment c456 --line 42   # Reattach orphaned comment
  comments reattach document.md --comment c789 --section "Introduction"  # Reattach to section
  comments cleanup document.md --dry-run                   # Preview cleanup
  comments cleanup document.md --status completed          # Archive completed TODOs

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

// resolveTextInput resolves text input that may be a file reference (@filename)
// If the input starts with '@', reads the file at the specified path
// Otherwise, returns the input as-is
func resolveTextInput(input string) (string, error) {
	if len(input) == 0 {
		return input, nil
	}

	// Check if input is a file reference
	if input[0] != '@' {
		return input, nil
	}

	// Extract filename (skip the @ prefix)
	filename := input[1:]
	if filename == "" {
		return "", fmt.Errorf("invalid file reference: '@' must be followed by a filename")
	}

	// Read file contents
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", filename, err)
	}

	return string(content), nil
}
