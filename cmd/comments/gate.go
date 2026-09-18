package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/rcliao/comments/pkg/comment"
)

// gateCommentJSON is the canonical comment view plus gate-specific document
// context lines.
// The gate's shapes live in pkg/comment so every surface reports one decision.
type (
	gateCommentJSON = comment.GateThread
	gateOutputJSON  = comment.GateReport
)

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

	report, err := comment.BuildGateReport(target, *strict, *templateName, *contextSize)
	if err != nil {
		return failf("Error: %v", err)
	}
	output := *report

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
