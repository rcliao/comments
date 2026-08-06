package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/tui"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// dispatch routes to the command handlers. Handlers return errors (optionally
// carrying an exit code via exitError) instead of exiting themselves.
func dispatch(args []string) error {
	if len(args) < 1 {
		return &exitError{code: 1, msg: strings.TrimRight(usageText, "\n")}
	}

	command := args[0]

	// Commands that take a positional <file> argument and its usage line
	fileUsage := map[string]string{
		"list":        "Usage: comments list <file> [flags]",
		"get":         "Usage: comments get <file> [flags]",
		"add":         "Usage: comments add <file> [flags]",
		"batch-add":   "Usage: comments batch-add <file> [flags]",
		"reply":       "Usage: comments reply <file> [flags]",
		"batch-reply": "Usage: comments batch-reply <file> [flags]",
		"resolve":     "Usage: comments resolve <file> [flags]",
		"suggest":     "Usage: comments suggest <file> [flags]",
		"accept":      "Usage: comments accept <file> [flags]",
		"reject":      "Usage: comments reject <file> [flags]",
		"validate":    "Usage: comments validate <file> --template <name>",
		"seed":        "Usage: comments seed <file> --template <name>",
		"gate":        "Usage: comments gate <file-or-dir> [flags]",
		"signoff":     "Usage: comments signoff <file> [flags]",
		"watch":       "Usage: comments watch <file-or-dir> [flags]",
	}
	if usage, needsFile := fileUsage[command]; needsFile && len(args) < 2 {
		return failf("%s", usage)
	}

	switch command {
	case "view":
		// View command can be called with or without a filename
		return viewCommand(args[1:])
	case "list":
		return listCommand(args[1], args[2:])
	case "get":
		return getCommand(args[1], args[2:])
	case "add":
		return addCommand(args[1], args[2:])
	case "batch-add":
		return batchAddCommand(args[1], args[2:])
	case "reply":
		return replyCommand(args[1], args[2:])
	case "batch-reply":
		return batchReplyCommand(args[1], args[2:])
	case "resolve":
		return resolveCommand(args[1], args[2:])
	case "suggest":
		return suggestCommand(args[1], args[2:])
	case "accept":
		return acceptCommand(args[1], args[2:])
	case "reject":
		return rejectCommand(args[1], args[2:])
	case "template":
		return templateCommand(args[1:])
	case "validate":
		return validateCommand(args[1], args[2:])
	case "seed":
		return seedCommand(args[1], args[2:])
	case "gate":
		return gateCommand(args[1], args[2:])
	case "signoff":
		return signoffCommand(args[1], args[2:])
	case "watch":
		return watchCommand(args[1], args[2:])
	case "serve-mcp":
		return serveMCPCommand()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return failf("Unknown command: %s\n\n%s", command, strings.TrimRight(usageText, "\n"))
	}
}

func viewCommand(args []string) error {
	// Parse flags; the filename is positional and may come before the flags
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	themeFlag := fs.String("theme", "", "Color theme: nord (default), dracula, gruvbox, ansi")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	filename := ""
	if rest := fs.Args(); len(rest) > 0 {
		filename = rest[0]
		if err := fs.Parse(rest[1:]); err != nil { // allow `view <file> --theme <name>` ordering
			return exitSilent(2)
		}
	}

	// Theme selection: --theme flag wins over COMMENTS_THEME env var
	themeName := *themeFlag
	if themeName == "" {
		themeName = os.Getenv("COMMENTS_THEME")
	}
	if themeName != "" && !tui.SetTheme(themeName) {
		fmt.Fprintf(os.Stderr, "Unknown theme %q. Valid themes: %s. Using default (%s).\n",
			themeName, strings.Join(tui.ThemeNames(), ", "), tui.DefaultThemeName)
	}

	var model tui.Model

	if filename == "" {
		// No filename provided - start with file picker
		model = tui.NewModel()
	} else {
		// Filename provided - load it directly
		doc, err := loadDocument(filename)
		if err != nil {
			return failf("Error loading document: %v", err)
		}

		// Create model with pre-loaded file
		model = tui.NewModelWithFile(doc, filename)
	}

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())

	final, err := p.Run()
	if err != nil {
		return failf("Error running TUI: %v", err)
	}
	// Verdict exit codes: view doubles as the interactive gate
	if fm, ok := final.(tui.Model); ok {
		switch fm.VerdictDecision {
		case comment.DecisionApproved:
			fmt.Println("✓ Review submitted: approved")
		case comment.DecisionChangesRequested:
			fmt.Println("✗ Review submitted: changes requested")
			return exitSilent(comment.GateExitCode)
		}
	}
	return nil
}

func listCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
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

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
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
			return failf("Error: %v", err)
		}
		filteredComments = filtered
	}

	// Apply section filter
	if *sectionFilter != "" {
		// Validate section exists
		if err := comment.ValidateSectionPath(doc.Content, *sectionFilter); err != nil {
			return failf("Error: %v", err)
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
			return failf("Error outputting JSON: %v", err)
		}
		return nil

	case "table":
		outputTable(filteredComments, doc.Threads)
		return nil

	case "text":
		// If --with-context is specified with text format, use context format
		if *withContext {
			output := formatListWithContext(filteredComments, doc.Content)
			fmt.Print(output)
			return nil
		}
		// Original text format (below)

	default:
		return failf("Error: Unknown format '%s'. Valid formats: text, json, table", *format)
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
		switch status {
		case "orphaned":
			statusIndicator = " ⚠️  ORPHANED"
			if thread.OrphanedReason != "" {
				statusIndicator += fmt.Sprintf(" (%s)", thread.OrphanedReason)
			}
		case "completed":
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
	return nil
}

func getCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	threadID := fs.String("thread", "", "Thread ID to get (required)")
	withReplies := fs.Bool("with-replies", true, "Include replies in output (default: true)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *threadID == "" {
		return failf("Error: --thread flag is required\nUsage: comments get <file> --thread <thread-id>")
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Compute section metadata for all comments if not already present
	comment.ComputeSectionsForComments(doc)

	// Find the comment anywhere in the nested thread tree (root, reply,
	// or reply-to-reply at any depth)
	foundComment := doc.FindCommentByID(*threadID)

	if foundComment == nil {
		var b strings.Builder
		fmt.Fprintf(&b, "Error: Thread with ID '%s' not found\n", *threadID)
		b.WriteString("\nAvailable threads:")
		for i, thread := range doc.Threads {
			fmt.Fprintf(&b, "\n  [%d] %s (Line %d) - @%s", i+1, thread.ID, thread.Line, thread.Author)
		}
		return failf("%s", b.String())
	}

	// Get context and format output
	ctx := getCommentContext(foundComment, doc.Content)
	output := formatCommentWithContext(foundComment, ctx, *withReplies)

	fmt.Print(output)
	return nil
}

// filterCommentsByType filters comments by type prefix ([Q], [S], [B], [T], [E])
func filterCommentsByType(comments []*comment.Comment, typePrefix string) []*comment.Comment {
	filtered := make([]*comment.Comment, 0)
	targetPrefix := "[" + typePrefix + "]"

	for _, c := range comments {
		if strings.HasPrefix(c.Text, targetPrefix) {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

func addCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	text := fs.String("text", "", "Comment text (required)")
	line := fs.Int("line", 0, "Line number (use either --line or --section)")
	section := fs.String("section", "", "Section path (use either --line or --section)")
	author := fs.String("author", "", "Author name (required)")
	commentType := fs.String("type", "", "Comment type: Q, S, B, T, E (auto-prefixes text)")
	priority := fs.String("priority", "medium", "Priority: low, medium, high (default: medium)")
	blocking := fs.Bool("blocking", false, "Mark comment as blocking (must be resolved before gate passes)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	const addUsage = "Usage: comments add <file> --line N --author \"name\" --text \"your comment\"\n" +
		"   or: comments add <file> --section \"Section Path\" --author \"name\" --text \"your comment\""

	if *text == "" {
		return failf("Error: --text flag is required\n%s", addUsage)
	}

	if *author == "" {
		return failf("Error: --author flag is required\n%s", addUsage)
	}

	// Validate that either line or section is provided (but not both)
	if *line == 0 && *section == "" {
		return failf("Error: either --line or --section flag is required\n%s", addUsage)
	}

	if *line != 0 && *section != "" {
		return failf("Error: cannot specify both --line and --section\nUse either --line N or --section \"Section Path\", not both")
	}

	// Resolve text input (supports @filename)
	resolvedText, err := resolveTextInput(*text)
	if err != nil {
		return failf("Error: %v", err)
	}

	// Auto-prefix text with type if specified
	commentText := resolvedText
	if *commentType != "" {
		commentText = "[" + *commentType + "] " + resolvedText
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Determine the line number to use
	targetLine := *line
	if *section != "" {
		// Validate section exists
		if err := comment.ValidateSectionPath(doc.Content, *section); err != nil {
			return failf("Error: %v", err)
		}

		// Resolve section to line number (use section start line)
		startLine, _, err := comment.ResolveSectionToLines(doc.Content, *section, false)
		if err != nil {
			return failf("Error resolving section: %v", err)
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
	newComment.Blocking = *blocking

	// Compute section metadata for the new comment
	comment.UpdateCommentSection(newComment, doc.Content)

	doc.Threads = append(doc.Threads, newComment)

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	// Display success message
	if newComment.SectionPath != "" {
		fmt.Printf("✓ Comment added to %s (Line %d) by @%s\n", newComment.SectionPath, targetLine, *author)
	} else {
		fmt.Printf("✓ Comment added to line %d by @%s\n", targetLine, *author)
	}
	fmt.Printf("  Comment ID: %s\n", newComment.ID)
	return nil
}

// availableThreadsMsg lists a document's root threads for not-found error messages
func availableThreadsMsg(doc *comment.DocumentWithComments) string {
	var b strings.Builder
	b.WriteString("\nAvailable threads:")
	for _, t := range doc.Threads {
		fmt.Fprintf(&b, "\n  %s (Line %d, %d replies)", t.ID, t.Line, t.CountReplies())
	}
	return b.String()
}

func replyCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("reply", flag.ContinueOnError)
	text := fs.String("text", "", "Reply text (required)")
	thread := fs.String("thread", "", "Thread ID (required)")
	author := fs.String("author", "", "Author name (required)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	const replyUsage = "Usage: comments reply <file> --thread ID --author \"name\" --text \"your reply\""

	if *text == "" {
		return failf("Error: --text flag is required\n%s", replyUsage)
	}

	if *thread == "" {
		return failf("Error: --thread flag is required\n%s", replyUsage)
	}

	if *author == "" {
		return failf("Error: --author flag is required\n%s", replyUsage)
	}

	// Resolve text input (supports @filename)
	resolvedText, err := resolveTextInput(*text)
	if err != nil {
		return failf("Error: %v", err)
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Add reply to thread using helper
	if err := comment.AddReplyToThread(doc.Threads, *thread, *author, resolvedText); err != nil {
		return failf("Error: %v\n%s", err, availableThreadsMsg(doc))
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	fmt.Printf("✓ Reply added to thread %s by @%s\n", *thread, *author)
	return nil
}

func resolveCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	thread := fs.String("thread", "", "Thread ID (required)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *thread == "" {
		return failf("Error: --thread flag is required\nUsage: comments resolve <file> --thread ID")
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Resolve the thread
	if err := comment.ResolveThread(doc.Threads, *thread); err != nil {
		return failf("Error: %v\n%s", err, availableThreadsMsg(doc))
	}

	// Save to sidecar
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	fmt.Printf("✓ Thread %s marked as resolved\n", *thread)
	return nil
}

func suggestCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("suggest", flag.ContinueOnError)
	startLine := fs.Int("start-line", 0, "Start line (use either line range or section)")
	endLine := fs.Int("end-line", 0, "End line (use either line range or section)")
	section := fs.String("section", "", "Section path (use either line range or section)")
	author := fs.String("author", "", "Author name (required)")
	text := fs.String("text", "", "Suggestion description (required)")
	original := fs.String("original", "", "Original text to replace")
	proposed := fs.String("proposed", "", "Proposed replacement text (required)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	const suggestUsage = "Usage: comments suggest <file> --start-line N --end-line M --author \"name\" --text \"desc\" --proposed \"new text\"\n" +
		"   or: comments suggest <file> --section \"Section Path\" --author \"name\" --text \"desc\" --proposed \"new text\""

	// Validate required flags
	if *author == "" {
		return failf("Error: --author flag is required\n%s", suggestUsage)
	}
	if *text == "" {
		return failf("Error: --text flag is required\n%s", suggestUsage)
	}
	if *proposed == "" {
		return failf("Error: --proposed flag is required\n%s", suggestUsage)
	}

	// Validate that either line range or section is provided (but not both)
	if *startLine == 0 && *section == "" {
		return failf("Error: either --start-line/--end-line or --section flag is required\n%s", suggestUsage)
	}

	if *startLine != 0 && *section != "" {
		return failf("Error: cannot specify both line range and section\nUse either --start-line/--end-line or --section, not both")
	}

	// Resolve text inputs (supports @filename)
	resolvedText, err := resolveTextInput(*text)
	if err != nil {
		return failf("Error resolving --text: %v", err)
	}

	resolvedOriginal, err := resolveTextInput(*original)
	if err != nil {
		return failf("Error resolving --original: %v", err)
	}

	resolvedProposed, err := resolveTextInput(*proposed)
	if err != nil {
		return failf("Error resolving --proposed: %v", err)
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Determine the line range to use
	targetStartLine := *startLine
	targetEndLine := *endLine
	if *section != "" {
		// Validate section exists
		if err := comment.ValidateSectionPath(doc.Content, *section); err != nil {
			return failf("Error: %v", err)
		}

		// Resolve section to line range
		start, end, err := comment.ResolveSectionToLines(doc.Content, *section, false)
		if err != nil {
			return failf("Error resolving section: %v", err)
		}
		targetStartLine = start
		targetEndLine = end
	}

	// Validate line range
	if targetEndLine == 0 {
		targetEndLine = targetStartLine
	}
	if targetStartLine > targetEndLine {
		return failf("Error: start line (%d) must be <= end line (%d)", targetStartLine, targetEndLine)
	}

	// Create suggestion using helper
	suggestion := comment.NewSuggestion(*author, targetStartLine, targetEndLine, resolvedText, resolvedOriginal, resolvedProposed)

	// Compute section metadata
	comment.UpdateCommentSection(suggestion, doc.Content)

	// Add to document
	doc.Threads = append(doc.Threads, suggestion)

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	if suggestion.SectionPath != "" {
		fmt.Printf("✓ Suggestion added to %s (Lines %d-%d) by @%s\n", suggestion.SectionPath, targetStartLine, targetEndLine, *author)
	} else {
		fmt.Printf("✓ Suggestion added to lines %d-%d by @%s\n", targetStartLine, targetEndLine, *author)
	}
	fmt.Printf("  Suggestion ID: %s\n", suggestion.ID)
	return nil
}

func acceptCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	suggestionID := fs.String("suggestion", "", "Suggestion ID (required)")
	preview := fs.Bool("preview", false, "Preview changes without applying")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *suggestionID == "" {
		return failf("Error: --suggestion flag is required")
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
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
		return failf("Error: Suggestion '%s' not found", *suggestionID)
	}

	if !suggestion.IsSuggestion {
		return failf("Error: Comment '%s' is not a suggestion", *suggestionID)
	}

	// Preview if requested
	if *preview {
		newContent, err := comment.ApplySuggestion(doc.Content, suggestion)
		if err != nil {
			return failf("Error applying suggestion: %v", err)
		}
		fmt.Println("Preview of changes:")
		fmt.Println("==================")
		fmt.Println(newContent)
		return nil
	}

	// Apply suggestion
	newContent, err := comment.ApplySuggestion(doc.Content, suggestion)
	if err != nil {
		return failf("Error applying suggestion: %v", err)
	}

	// Update document content
	doc.Content = newContent

	// Mark suggestion as accepted using helper
	if err := comment.AcceptSuggestion(doc.Threads, *suggestionID); err != nil {
		return failf("Error marking suggestion as accepted: %v", err)
	}

	// Recalculate comment line numbers (line-only tracking)
	linesAdded := len(strings.Split(suggestion.ProposedText, "\n"))
	comment.RecalculateCommentLines(doc.Threads, suggestion.StartLine, suggestion.EndLine, linesAdded)

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	fmt.Printf("✓ Suggestion %s accepted and applied\n", *suggestionID)
	return nil
}

func rejectCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("reject", flag.ContinueOnError)
	suggestionID := fs.String("suggestion", "", "Suggestion ID (required)")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *suggestionID == "" {
		return failf("Error: --suggestion flag is required")
	}

	// Load document
	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	// Mark suggestion as rejected using helper
	if err := comment.RejectSuggestion(doc.Threads, *suggestionID); err != nil {
		return failf("Error: %v", err)
	}

	// Save
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	fmt.Printf("✓ Suggestion %s rejected\n", *suggestionID)
	return nil
}

func printUsage() {
	fmt.Print(usageText)
}

const usageText = `comments - CLI tool for collaborative document commenting

Usage:
  comments <command> [arguments]

Commands:
  view <file> [flags]         Open interactive TUI viewer
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
  gate <file-or-dir> [flags]  Evaluate review gate (exit 0 = approved, 10 = changes requested)
  signoff <file> [flags]      Record a completed human review pass
  template list|show <name>   List or inspect doc templates (guardrails for agent-written docs)
  validate <file> [flags]     Check document structure against a template (exit 1 on violations)
  seed <file> [flags]         Create review threads from a template's criteria and markers
  watch <file-or-dir> [flags] Emit review-state change events as NDJSON (poll-based; --until exits on a matching event)
  serve-mcp                   Start Model Context Protocol server (for LLM integration)
  help                        Show this help message

View Command Flags:
  --theme <name>              Color theme: nord (default), dracula, gruvbox, ansi
                              (COMMENTS_THEME env var also works; the flag wins)

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
  --blocking                  Mark as blocking (must be resolved before gate passes)

Gate Command Flags:
  --json                      Output machine-readable JSON decision
  --strict                    Fail on any unresolved comment or pending suggestion
  --context <n>               Lines of context around each comment (default: 2)
  --template <name>           Also validate structure against a template (defaults to sidecar record)

Validate/Seed Command Flags:
  --template <name>           Template name (defaults to template recorded in sidecar)
  --json                      (validate) Output violations as JSON
  --author <name>             (seed) Author for seeded threads (default: template)
  --markers-only              (seed) Seed only NEEDS CLARIFICATION markers (agent posts specific callouts)

Signoff Command Flags:
  --author <name>             Reviewer name (defaults to $USER)
  --decision <decision>       Override: approved or changes_requested (default: derived from gate)
  --note <text>               Optional review note
  --strict                    Derive decision using strict gate rules

Watch Command Flags:
  --interval <duration>       Poll interval (default: 1s)
  --until <events>            Exit 0 after emitting a matching event; comma-separated event
                              types (e.g. signoff or signoff,gate_changed)

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
  comments suggest document.md --start-line 5 --end-line 8 \
    --author "claude" --text "Restructure intro" \
    --original "old text" --proposed "new text"

  # Accept/reject suggestions
  comments accept document.md --suggestion c123 --preview  # Preview changes first
  comments accept document.md --suggestion c123            # Apply the changes
  comments reject document.md --suggestion c456            # Reject suggestion

  # Status management - track TODOs and handle document changes
  comments list document.md --status orphaned              # View comments orphaned by edits
  comments list document.md --priority high                # View high-priority TODOs
  comments list document.md --status active --priority high # Active high-priority items

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
