package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
)

// gateCommentJSON is the canonical comment view plus gate-specific document
// context lines.
type gateCommentJSON struct {
	comment.CommentView
	Context []string `json:"context,omitempty"`
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
	// StructureUnchecked marks a commented doc with no template recorded: the
	// gate checked comment state only, so a silent pass is not a structural pass.
	StructureUnchecked bool `json:"structure_unchecked,omitempty"`
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
func gateCommand(target string, args []string) error {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON decision")
	strict := fs.Bool("strict", false, "Fail on any unresolved comment or pending suggestion, not just blocking ones")
	contextSize := fs.Int("context", 2, "Lines of document context around each comment (0 to disable)")
	templateName := fs.String("template", "", "Also validate structure against this template (defaults to frontmatter, sidecar, or bundle)")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	files, err := comment.FindGateTargets(target)
	if err != nil {
		return failf("Error: %v", err)
	}
	if len(files) == 0 {
		return failf("Error: no markdown files with comment sidecars found under %s", target)
	}

	output := gateOutputJSON{Decision: comment.DecisionApproved, Strict: *strict}

	for _, file := range files {
		doc, err := loadDocument(file)
		if err != nil {
			return failf("Error loading %s: %v", file, err)
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

		// Explicit flag wins; otherwise frontmatter, legacy sidecar, then bundle.
		t, _, err := comment.ResolveTemplateForDocument(file, doc.Content, *templateName, doc.Template)
		if err != nil {
			return failf("Error: %v", err)
		}
		if t != nil {
			fileJSON.Template = t.Name
			fileJSON.Violations = comment.ValidateManagedDocument(doc.Content, file, t)
			if len(fileJSON.Violations) > 0 {
				fileJSON.Decision = comment.DecisionChangesRequested
			}
		} else if len(doc.Threads) > 0 {
			// A doc with no discoverable template passes the structural half of the
			// gate by default, which reads identically to passing it on merit.
			// Shipped RPI artifacts have gone out hundreds of words over their
			// caps this way, so say it out loud.
			fileJSON.StructureUnchecked = true
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
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
	} else {
		printGateText(output)
	}

	if output.Decision == comment.DecisionChangesRequested {
		return exitSilent(comment.GateExitCode)
	}
	return nil
}

// signoffCommand records a completed human review pass on a document.
func signoffCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("signoff", flag.ContinueOnError)
	author := fs.String("author", os.Getenv("USER"), "Reviewer name (defaults to $USER)")
	decision := fs.String("decision", "", "Override decision: approved or changes_requested (default: derived from gate)")
	note := fs.String("note", "", "Optional review note")
	strict := fs.Bool("strict", false, "Derive decision using strict gate rules")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	if *author == "" {
		return failf("Error: --author is required (or set $USER)")
	}
	if *decision != "" && *decision != comment.DecisionApproved && *decision != comment.DecisionChangesRequested && *decision != comment.DecisionCommented {
		return failf("Error: invalid --decision %q (use %s, %s or %s)", *decision, comment.DecisionApproved, comment.DecisionChangesRequested, comment.DecisionCommented)
	}

	doc, err := loadDocument(filename)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	record := comment.AddReviewRecord(doc, *author, *decision, *note, *strict)
	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	// A verdict also stores the reviewed content as this reviewer's baseline
	// (what "changed since your signoff" marks diff against). Best-effort: the
	// signoff is already in the sidecar, so a baseline failure must not report
	// a landed signoff as failed. Gate on the RECORD's decision — the flag may
	// be empty and derived by the gate.
	if comment.BaselineUpdatesOn(record.Decision) {
		if err := comment.SaveReviewBaseline(filename, record.Author, doc.Content); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save review baseline: %v\n", err)
		}
	}

	fmt.Printf("✓ Review recorded: %s by @%s\n", record.Decision, record.Author)
	if record.Decision == comment.DecisionChangesRequested {
		result := comment.EvaluateGate(doc, *strict)
		fmt.Printf("  %d blocking comment(s) remain — agents waiting on request_review will now see this feedback\n", len(result.Blocking))
	}
	return nil
}

func toGateJSON(comments []*comment.Comment, docContent string, contextSize int) []gateCommentJSON {
	out := []gateCommentJSON{}
	lines := strings.Split(docContent, "\n")
	for _, c := range comments {
		item := gateCommentJSON{CommentView: comment.NewCommentView(c)}
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

// printStructureUnchecked names docs whose structure the gate never checked, so
// an approval cannot be mistaken for a structural pass.
func printStructureUnchecked(output gateOutputJSON) {
	for _, file := range output.Files {
		if file.StructureUnchecked {
			fmt.Printf("  ⚠ %s: structure unchecked — no template recorded.\n"+
				"    Add comments.template frontmatter or pass: comments gate %s --template <name>\n", file.File, file.File)
		}
	}
}

func printGateText(output gateOutputJSON) {
	if output.Decision == comment.DecisionApproved {
		fmt.Printf("✓ Gate passed: approved (%d file(s) checked)\n", len(output.Files))
		if output.Summary.NonBlocking > 0 || output.Summary.PendingSuggestions > 0 {
			fmt.Printf("  Note: %d non-blocking comment(s) and %d pending suggestion(s) remain\n",
				output.Summary.NonBlocking, output.Summary.PendingSuggestions)
		}
		printStructureUnchecked(output)
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
	printStructureUnchecked(output)
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
