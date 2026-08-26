package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rcliao/comments/pkg/comment"
)

// analyzeCommand reports deterministic research and plan coverage. It is
// advisory by design: ready=false is data for an agent loop, not a second gate.
func analyzeCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	against := fs.String("against", "", "Research document to check plan coverage against")
	templateName := fs.String("template", "", "Template name (defaults to frontmatter, sidecar, or bundle)")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return failf("Error: invalid path: %v", err)
	}
	doc, err := loadDocument(absPath)
	if err != nil {
		return failf("Error loading document: %v", err)
	}

	var template *comment.Template
	template, _, err = comment.ResolveTemplateForDocument(absPath, doc.Content, *templateName, doc.Template)
	if err != nil {
		return failf("Error: %v", err)
	}

	var againstContent, againstPath string
	if *against != "" {
		againstPath, err = filepath.Abs(*against)
		if err != nil {
			return failf("Error: invalid --against path: %v", err)
		}
		data, err := os.ReadFile(againstPath)
		if err != nil {
			return failf("Error reading --against document: %v", err)
		}
		againstContent = string(data)
	}

	result := comment.AnalyzeDocument(doc.Content, absPath, template, againstContent, againstPath)
	if *jsonOut {
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return failf("Error encoding JSON: %v", err)
		}
		fmt.Println(string(encoded))
		return nil
	}

	state := "ready"
	if !result.Ready {
		state = "not ready"
	}
	fmt.Printf("Analysis: %s — %s\n", state, filename)
	if result.StructureUnchecked {
		fmt.Println("  structure unchecked: add comments.template frontmatter or pass --template before treating this artifact as ready")
	}
	for _, q := range result.Questions {
		fmt.Printf("  %s line %d → %v\n", q.ID, q.Line, q.Findings)
	}
	for _, c := range result.Coverage {
		fmt.Printf("  %-9s %s (lines %d-%d)\n", c.Status, c.Finding.Title, c.Finding.StartLine, c.Finding.EndLine)
	}
	for _, v := range result.Violations {
		fmt.Printf("  [%s] %s\n", v.Rule, v.Message)
	}
	return nil
}
