package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/rcliao/comments/pkg/comment"
)

func newCommand(name string, args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	templateName := fs.String("template", "", "Template that determines the collection and document shape")
	title := fs.String("title", "", "Document title (defaults to the slug)")
	description := fs.String("description", "", "One-sentence concept description")
	from := fs.String("from", "", "Related source document to record as informed_by")
	startDir := fs.String("bundle", ".", "Path used to discover .comments/bundle.yaml")
	jsonOut := fs.Bool("json", false, "Output the created document as JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}
	result, err := comment.CreateBundleDocument(comment.NewDocumentOptions{
		Name: name, Template: *templateName, Title: *title,
		Description: *description, From: *from, StartDir: *startDir,
	})
	if err != nil {
		return failf("Error: %v", err)
	}
	if *jsonOut {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}
	rel, relErr := filepath.Rel(".", result.Path)
	if relErr != nil {
		rel = result.Path
	}
	fmt.Printf("✓ Created %s concept %s with template %q\n", result.Type, rel, result.Template)
	fmt.Printf("  Review threads: %s\n", result.Sidecar)
	return nil
}

func contextCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	mode := fs.String("for", "drafting", "Context mode: drafting, review, coverage-scout, evidence-verifier, human-review")
	includeBody := fs.Bool("include-body", false, "Include document bodies")
	includeThreads := fs.Bool("include-threads", false, "Include review threads")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}
	result, err := comment.BuildDocumentContext(filename, comment.ContextOptions{
		For: *mode, IncludeBody: *includeBody, IncludeThreads: *includeThreads,
	})
	if err != nil {
		return failf("Error: %v", err)
	}
	if *jsonOut {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
		return nil
	}
	fmt.Printf("Context for %s (%s) — %s mode\n", result.Document.Path, result.Document.Type, result.Mode)
	fmt.Printf("  template: %s  review: %s (%d blocking)\n", result.Document.Template, result.Document.Review.Decision, result.Document.Review.Blocking)
	printContextRelations("Related", result.Related)
	printContextRelations("Backlinks", result.Backlinks)
	printContextRelations("Suggested", result.Suggestions)
	if len(result.Sources) > 0 {
		fmt.Println("Sources:")
		for _, source := range result.Sources {
			fmt.Printf("  - %s\n", source.Resource)
		}
	}
	return nil
}

func printContextRelations(label string, relations []comment.ContextRelation) {
	if len(relations) == 0 {
		return
	}
	fmt.Printf("%s:\n", label)
	for _, relation := range relations {
		kind := relation.Relation
		if kind == "" {
			kind = "related"
		}
		fmt.Printf("  - %s [%s] — %s\n", relation.Document.Path, kind, relation.Reason)
	}
}

func bundleCommand(args []string) error {
	if len(args) == 0 || args[0] != "index" {
		return failf("Usage: comments bundle index [path]")
	}
	start := "."
	if len(args) > 1 {
		start = args[1]
	}
	bundle, err := comment.FindBundle(start)
	if err != nil {
		return failf("Error: %v", err)
	}
	if err := comment.WriteBundleIndexes(bundle); err != nil {
		return failf("Error: %v", err)
	}
	fmt.Printf("✓ Indexed OKF bundle %q at %s\n", bundle.Config.Bundle, bundle.RootPath)
	return nil
}
