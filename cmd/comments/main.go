package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
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
		"get":      "Usage: comments get <file> [flags]",
		"add":      "Usage: comments add <file> [flags]",
		"reply":    "Usage: comments reply <file> [flags]",
		"suggest":  "Usage: comments suggest <file> [flags]",
		"validate": "Usage: comments validate <file> --template <name>",
		"analyze":  "Usage: comments analyze <file> [--against <research.md>] [--json]",
		"context":  "Usage: comments context <file> [flags]",
		"new":      "Usage: comments new <name> --template <name> [flags]",
		"gate":     "Usage: comments gate <file-or-dir> [flags]",
		"watch":    "Usage: comments watch <file-or-dir> [flags]",
		"reanchor": "Usage: comments reanchor <file> --comment ID --line N | --json <file|->",
		"inbox":    "Usage: comments inbox <file-or-dir> [flags]",
		"serve":    "Usage: comments serve <file-or-dir> [flags]",
	}
	if usage, needsFile := fileUsage[command]; needsFile && len(args) < 2 {
		return failf("%s", usage)
	}

	switch command {
	case "view":
		// View command can be called with or without a filename
		return viewCommand(args[1:])
	case "get":
		return getCommand(args[1], args[2:])
	case "add":
		return addCommand(args[1], args[2:])
	case "reply":
		return replyCommand(args[1], args[2:])
	case "suggest":
		return suggestCommand(args[1], args[2:])
	case "template":
		return templateCommand(args[1:])
	case "validate":
		return validateCommand(args[1], args[2:])
	case "analyze":
		return analyzeCommand(args[1], args[2:])
	case "context":
		return contextCommand(args[1], args[2:])
	case "new":
		return newCommand(args[1], args[2:])
	case "bundle":
		return bundleCommand(args[1:])
	case "gate":
		return gateCommand(args[1], args[2:])
	case "watch":
		return watchCommand(args[1], args[2:])
	case "reanchor":
		return reanchorCommand(args[1], args[2:])
	case "inbox":
		return inboxCommand(args[1], args[2:])
	case "doctor":
		return doctorCommand(args[1:])
	case "serve":
		return serveCommand(args[1], args[2:])
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
	// Bubbletea v2: the alt screen is a View field (set in tui.Model.View),
	// no longer a program option
	p := tea.NewProgram(model)

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
		case comment.DecisionCommented:
			fmt.Println("💬 Review submitted: commented — replies handed to the agent, iteration continues")
		}
	}
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

func suggestCommand(filename string, args []string) error {
	// Parse flags
	fs := flag.NewFlagSet("suggest", flag.ContinueOnError)
	startLine := fs.Int("start-line", 0, "Start line (use either line range or section)")
	endLine := fs.Int("end-line", 0, "End line (use either line range or section)")
	section := fs.String("section", "", "Section path (use either line range or section)")
	anchorFlag := fs.String("anchor", "", "Anchor the range by quoting its FIRST line; --original's line count sets the end")
	author := fs.String("author", "", "Author name (required)")
	text := fs.String("text", "", "Suggestion description (required)")
	original := fs.String("original", "", "Original text to replace")
	proposed := fs.String("proposed", "", "Proposed replacement text (required)")
	jsonOut := fs.Bool("json-out", false, "Output machine-readable JSON")

	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	const suggestUsage = "Usage: comments suggest <file> --start-line N --end-line M --author \"name\" --text \"desc\" --proposed \"new text\"\n" +
		"   or: comments suggest <file> --section \"Section Path\" --author \"name\" --text \"desc\" --proposed \"new text\"\n" +
		"   or: comments suggest <file> --anchor \"quoted first line\" --author \"name\" --text \"desc\" --original \"old\" --proposed \"new\""

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

	// Exactly one of line range / section / anchor locates the suggestion
	locGiven := 0
	for _, ok := range []bool{*startLine != 0, *section != "", *anchorFlag != ""} {
		if ok {
			locGiven++
		}
	}
	if locGiven == 0 {
		return failf("Error: one of --start-line/--end-line, --section or --anchor is required\n%s", suggestUsage)
	}
	if locGiven > 1 {
		return failf("Error: --start-line, --section and --anchor are mutually exclusive")
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

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}
	result, err := comment.AddSuggestion(doc, comment.SuggestionSpec{
		Author: *author, Text: resolvedText, StartLine: *startLine, EndLine: *endLine,
		Section: *section, Anchor: *anchorFlag, OriginalText: resolvedOriginal, ProposedText: resolvedProposed,
	})
	if err != nil {
		return failf("Error: %v\n%s", err, suggestUsage)
	}
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}
	if *jsonOut {
		return printJSON(result)
	}
	where := fmt.Sprintf("lines %d-%d", result.StartLine, result.EndLine)
	if result.SectionPath != "" {
		where = fmt.Sprintf("%s (Lines %d-%d)", result.SectionPath, result.StartLine, result.EndLine)
	}
	fmt.Printf("✓ Suggestion added to %s by @%s\n  Suggestion ID: %s\n", where, *author, result.SuggestionID)
	return nil
}

func printUsage() {
	fmt.Print(usageText)
}

const usageText = `comments - review agent-written markdown like a Google Doc, from the terminal

Usage:
  comments <command> [arguments]

The loop: an agent drafts a doc under a template and annotates it; a human
reviews it in 'comments view' and gives a verdict on exit; the agent reads its
inbox, responds, and waits again — until the gate passes.

Human — review and decide:
  view <file>                 Interactive TUI. Reply, resolve, accept or reject suggestions;
                              q submits the verdict (a approve / c request changes / r reply-pass,
                              n adds a note). The verdict is only ever written here or in 'serve'.
  serve <file-or-dir>         The same review in a token-protected local browser workspace

Agent — create a document under a template:
  new <name> --template T     Create an OKF concept in its template-guided bundle folder
  context <file>              The writing brief's neighborhood: related concepts, backlinks,
                              sources and review state (--for drafting|review|implementation|...)
  template list|show <name>   List templates, or print one as a writing brief
  validate <file>             Check structure against the template (exit 1 on violations)
  analyze <file>              Advisory question / evidence / research-to-plan coverage

Agent — annotate the draft:
  add <file>                  Add one comment (flags) or many (--json), placed by quoted
                              anchor, section path or line. Atomic. --blocking marks must-fix.

Agent — iterate with the human:
  watch <file-or-dir>         Emit review events as NDJSON; '--until signoff' blocks until the
                              human's verdict and prints it with their decision and note
  inbox <file-or-dir>         THE read: gate decision, every open thread (blocking first) with
                              replies and context, pending suggestions, template violations,
                              orphaned anchors, and lines changed since the last verdict
  get <file>                  Look something up: one comment with context (--thread ID), or
                              every thread including resolved ones. Also: get thread:doc.md#c1abc
  reply <file>                Reply to a thread; --resolve also closes it (refused for an agent
                              in a zone: human section). --json replies to many.
  suggest <file>              Propose an edit for the human to accept or reject in 'view'
  reanchor <file>             After editing a commented doc, migrate the anchors you displaced

Scripts and maintenance:
  gate <file-or-dir>          Exit-code contract: 0 = approved, 10 = changes requested
  doctor [path]               Install health: binary, MCP server, plugin, sidecars
  bundle index [path]         Regenerate OKF root and collection indexes
  serve-mcp                   MCP server over stdio: the agent commands above, as tools
  help                        Show this help

Flags (every command also has -h):
  add        --anchor TEXT | --section PATH | --line N   --author NAME --text TEXT
             [--type Q|S|B|T|E] [--priority low|medium|high] [--blocking]
             --json FILE|-   add many        --json-out   machine-readable result
  reply      --thread ID --author NAME --text TEXT [--resolve]
             --json FILE|-   reply to many   --json-out
  get        [--thread ID] [--unresolved] [--from CITING-DOC] [--json]
  inbox      [--since RFC3339] [--reviewer NAME] [--json]
  watch      [--until signoff[,gate_changed]] [--interval 1s]
  suggest    --anchor TEXT | --start-line N --end-line M   --author NAME --text TEXT
             --proposed TEXT [--original TEXT]
  reanchor   --comment ID --line N | --section PATH      --json FILE|-   --json-out
  gate       [--json] [--strict] [--template NAME] [--context N]
  validate   [--template NAME] [--json]
  analyze    [--against research.md] [--template NAME] [--json]
  new        --template NAME [--title T] [--description D] [--from related.md] [--json]
  context    [--for MODE] [--include-body] [--include-threads] [--json]
  view       [--theme nord|dracula|gruvbox|ansi]
  doctor     [--json] [--skip-mcp]
  --text, --original and --proposed accept @filename to read content from a file.

Examples:
  comments new cache-policy --template design-doc
  comments add doc.md --anchor "Page loads take four seconds" --author claude \
    --type Q --blocking --text "Measured or estimated?"
  comments add doc.md --json annotations.json
  comments view doc.md                          # the human reviews; q gives the verdict
  comments watch doc.md --until signoff         # the agent waits on it
  comments inbox doc.md --json                  # then reads everything that needs it
  comments reply doc.md --thread c7f3k --author claude --text "Fixed: cites the dashboard." --resolve
  comments gate doc.md                          # exit 0 = approved

JSON formats:
  add --json       [{"anchor": "quoted target line", "author": "claude", "text": "...",
                     "type": "Q", "priority": "high", "blocking": true},
                    {"section": "Doc Title > Proposed Design", "author": "claude", "text": "..."}]
                   Each comment takes exactly one of anchor, section or line. Prefer anchor.
  reply --json     [{"thread_id": "c7f3k", "author": "claude", "text": "...", "resolve": true}]
  reanchor --json  [{"comment_id": "c7f3k", "line": 42},
                    {"comment_id": "c9b21", "section": "Doc Title > Proposed Design"}]

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
