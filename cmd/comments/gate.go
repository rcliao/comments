package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
)

// gateCommentJSON is the JSON shape for a comment in gate output.
// Matches the field naming used by `list --format json`.
type gateCommentJSON struct {
	ID          string   `json:"id"`
	Author      string   `json:"author"`
	Line        int      `json:"line"`
	Timestamp   string   `json:"timestamp"`
	Text        string   `json:"text"`
	Type        string   `json:"type,omitempty"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Blocking    bool     `json:"blocking"`
	ReplyCount  int      `json:"reply_count"`
	SectionPath string   `json:"section_path,omitempty"`
	Context     []string `json:"context,omitempty"`
}

type gateFileJSON struct {
	File               string                `json:"file"`
	Decision           string                `json:"decision"`
	Blocking           []gateCommentJSON     `json:"blocking"`
	NonBlocking        []gateCommentJSON     `json:"non_blocking"`
	PendingSuggestions []gateCommentJSON     `json:"pending_suggestions"`
	Template           string                `json:"template,omitempty"`
	Violations         []comment.Violation   `json:"violations,omitempty"`
	LastReview         *comment.ReviewRecord `json:"last_review,omitempty"`
}

type gateOutputJSON struct {
	Decision string         `json:"decision"`
	Strict   bool           `json:"strict"`
	Files    []gateFileJSON `json:"files"`
	Summary  struct {
		Blocking           int `json:"blocking"`
		NonBlocking        int `json:"non_blocking"`
		PendingSuggestions int `json:"pending_suggestions"`
		Violations         int `json:"violations"`
	} `json:"summary"`
}

// gateCommand evaluates the review gate for a file or directory of markdown files.
// Exit codes: 0 = approved, 10 = changes requested, 1 = error.
func gateCommand(target string, args []string) {
	fs := flag.NewFlagSet("gate", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON decision")
	strict := fs.Bool("strict", false, "Fail on any unresolved comment or pending suggestion, not just blocking ones")
	contextSize := fs.Int("context", 2, "Lines of document context around each comment (0 to disable)")
	templateName := fs.String("template", "", "Also validate structure against this template (defaults to template recorded in sidecar)")
	_ = fs.Parse(args)

	files, err := comment.FindGateTargets(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no markdown files with comment sidecars found under %s\n", target)
		os.Exit(1)
	}

	output := gateOutputJSON{Decision: comment.DecisionApproved, Strict: *strict}

	for _, file := range files {
		doc, err := comment.LoadFromSidecar(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", file, err)
			os.Exit(1)
		}
		result := comment.EvaluateGate(doc, *strict)

		fileJSON := gateFileJSON{
			File:               file,
			Decision:           result.Decision,
			Blocking:           toGateJSON(result.Blocking, doc.Content, *contextSize),
			NonBlocking:        toGateJSON(result.NonBlocking, doc.Content, *contextSize),
			PendingSuggestions: toGateJSON(result.PendingSuggestions, doc.Content, *contextSize),
			LastReview:         result.LastReview,
		}

		// Template structural validation: explicit flag wins, else sidecar record
		tName := *templateName
		if tName == "" {
			tName = doc.Template
		}
		if tName != "" {
			t, err := comment.LoadTemplate(tName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fileJSON.Template = t.Name
			fileJSON.Violations = comment.ValidateTemplate(doc.Content, t)
			if len(fileJSON.Violations) > 0 {
				fileJSON.Decision = comment.DecisionChangesRequested
			}
		}
		output.Files = append(output.Files, fileJSON)
		output.Summary.Blocking += len(result.Blocking)
		output.Summary.NonBlocking += len(result.NonBlocking)
		output.Summary.PendingSuggestions += len(result.PendingSuggestions)
		output.Summary.Violations += len(fileJSON.Violations)
		if fileJSON.Decision == comment.DecisionChangesRequested {
			output.Decision = comment.DecisionChangesRequested
		}
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(encoded))
	} else {
		printGateText(output)
	}

	if output.Decision == comment.DecisionChangesRequested {
		os.Exit(comment.GateExitCode)
	}
}

// signoffCommand records a completed human review pass on a document.
func signoffCommand(filename string, args []string) {
	fs := flag.NewFlagSet("signoff", flag.ExitOnError)
	author := fs.String("author", os.Getenv("USER"), "Reviewer name (defaults to $USER)")
	decision := fs.String("decision", "", "Override decision: approved or changes_requested (default: derived from gate)")
	note := fs.String("note", "", "Optional review note")
	strict := fs.Bool("strict", false, "Derive decision using strict gate rules")
	_ = fs.Parse(args)

	if *author == "" {
		fmt.Println("Error: --author is required (or set $USER)")
		os.Exit(1)
	}
	if *decision != "" && *decision != comment.DecisionApproved && *decision != comment.DecisionChangesRequested {
		fmt.Printf("Error: invalid --decision %q (use %s or %s)\n", *decision, comment.DecisionApproved, comment.DecisionChangesRequested)
		os.Exit(1)
	}

	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}

	record := comment.AddReviewRecord(doc, *author, *decision, *note, *strict)
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Review recorded: %s by @%s\n", record.Decision, record.Author)
	if record.Decision == comment.DecisionChangesRequested {
		result := comment.EvaluateGate(doc, *strict)
		fmt.Printf("  %d blocking comment(s) remain — agents waiting on request_review will now see this feedback\n", len(result.Blocking))
	}
}

func toGateJSON(comments []*comment.Comment, docContent string, contextSize int) []gateCommentJSON {
	out := []gateCommentJSON{}
	lines := strings.Split(docContent, "\n")
	for _, c := range comments {
		item := gateCommentJSON{
			ID:          c.ID,
			Author:      c.Author,
			Line:        c.Line,
			Timestamp:   c.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Text:        c.Text,
			Type:        c.Type,
			Status:      c.GetStatus(),
			Priority:    c.GetPriority(),
			Blocking:    c.Blocking,
			ReplyCount:  c.CountReplies(),
			SectionPath: c.SectionPath,
		}
		if contextSize > 0 {
			start := max(1, c.Line-contextSize)
			end := min(len(lines), c.Line+contextSize)
			for i := start; i <= end; i++ {
				marker := "  "
				if i == c.Line {
					marker = "► "
				}
				item.Context = append(item.Context, fmt.Sprintf("%s%d │ %s", marker, i, lines[i-1]))
			}
		}
		out = append(out, item)
	}
	return out
}

func printGateText(output gateOutputJSON) {
	if output.Decision == comment.DecisionApproved {
		fmt.Printf("✓ Gate passed: approved (%d file(s) checked)\n", len(output.Files))
		if output.Summary.NonBlocking > 0 || output.Summary.PendingSuggestions > 0 {
			fmt.Printf("  Note: %d non-blocking comment(s) and %d pending suggestion(s) remain\n",
				output.Summary.NonBlocking, output.Summary.PendingSuggestions)
		}
		return
	}

	fmt.Printf("✗ Gate failed: changes requested (%d blocking, %d non-blocking, %d pending suggestions, %d template violations)\n\n",
		output.Summary.Blocking, output.Summary.NonBlocking, output.Summary.PendingSuggestions, output.Summary.Violations)
	for _, file := range output.Files {
		if file.Decision != comment.DecisionChangesRequested {
			continue
		}
		fmt.Printf("%s:\n", file.File)
		for _, v := range file.Violations {
			fmt.Printf("  [TEMPLATE:%s] %s\n", v.Rule, v.Message)
		}
		if len(file.Violations) > 0 {
			fmt.Println()
		}
		printGateComments("BLOCKING", file.Blocking)
		if output.Strict {
			printGateComments("unresolved", file.NonBlocking)
			printGateComments("pending suggestion", file.PendingSuggestions)
		}
	}
	fmt.Printf("Resolve blocking comments (comments resolve/reply) then re-run gate. Exit code %d.\n", comment.GateExitCode)
}

func printGateComments(label string, comments []gateCommentJSON) {
	for _, c := range comments {
		location := fmt.Sprintf("line %d", c.Line)
		if c.SectionPath != "" {
			location = fmt.Sprintf("%s (line %d)", c.SectionPath, c.Line)
		}
		fmt.Printf("  [%s] %s • @%s • %s\n", label, c.ID, c.Author, location)
		fmt.Printf("      %s\n", c.Text)
		for _, line := range c.Context {
			fmt.Printf("      %s\n", line)
		}
		fmt.Println()
	}
}
