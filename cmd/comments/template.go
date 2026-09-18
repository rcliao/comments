package main

import (
	"flag"
	"fmt"
	"sort"

	"github.com/rcliao/comments/pkg/comment"
)

// templateCommand handles `comments template list|show <name>`
func templateCommand(args []string) error {
	if len(args) == 0 {
		return failf("Usage: comments template list | comments template show <name>")
	}

	switch args[0] {
	case "list":
		templates, err := comment.ListTemplates()
		if err != nil {
			return failf("Error listing templates: %v", err)
		}
		names := make([]string, 0, len(templates))
		for name := range templates {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Printf("Available templates (project templates in %s override built-ins):\n", comment.ProjectTemplateDir)
		for _, name := range names {
			t, err := comment.LoadTemplate(name)
			desc := ""
			if err == nil {
				desc = t.Description
			}
			fmt.Printf("  %-12s (%s)  %s\n", name, templates[name], desc)
		}

	case "show":
		if len(args) < 2 {
			return failf("Usage: comments template show <name>")
		}
		t, err := comment.LoadTemplate(args[1])
		if err != nil {
			return failf("Error: %v", err)
		}
		fmt.Print(t.Brief().Text())

	default:
		return failf("Unknown template subcommand: %s (use list or show)", args[0])
	}
	return nil
}

// validateCommand handles `comments validate <file> --template <name> [--json]`
// Exit codes: 0 = conforms, 1 = violations or error.
func validateCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	templateName := fs.String("template", "", "Template name (defaults to frontmatter, sidecar, or bundle)")
	jsonOut := fs.Bool("json", false, "Output violations as JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	t, doc, err := loadTemplateForDoc(filename, *templateName)
	if err != nil {
		return err
	}
	report := comment.BuildValidationReport(doc.Content, filename, t)
	violations, wordReport := report.Violations, report.SectionWords

	if *jsonOut {
		if err := printJSON(report); err != nil {
			return err
		}
	} else {
		// Markers are deliberate: an agent flags an ambiguity it refuses to
		// guess at, then adds a blocking comment at that line. Lumping them in
		// with structural defects made "validate until it is clean" unreachable
		// for any doc that uses one — two independent drafting agents read the
		// same output and could not tell which violations they were meant to fix.
		var structural, markers []comment.Violation
		for _, v := range violations {
			if v.Rule == "unresolved_marker" || v.Rule == "too_many_markers" {
				markers = append(markers, v)
			} else {
				structural = append(structural, v)
			}
		}

		switch {
		case len(violations) == 0:
			fmt.Printf("✓ %s conforms to template %q\n", filename, t.Name)
		case len(structural) == 0:
			fmt.Printf("✓ %s conforms to template %q — structure is clean\n\n", filename, t.Name)
			fmt.Printf("%d intentional marker(s), each needs a blocking review comment:\n", len(markers))
			for _, v := range markers {
				fmt.Printf("  [%s] %s\n", v.Rule, v.Message)
			}
			fmt.Printf("\nAnnotate each with: comments add %s --anchor <marker text> --blocking --type Q --author <agent> --text <question>\n", filename)
		default:
			fmt.Printf("✗ %s has %d structural violation(s) against template %q — fix these:\n\n", filename, len(structural), t.Name)
			for _, v := range structural {
				fmt.Printf("  [%s] %s\n", v.Rule, v.Message)
			}
			if len(markers) > 0 {
				fmt.Printf("\n%d intentional marker(s), expected — leave them for the human:\n", len(markers))
				for _, v := range markers {
					fmt.Printf("  [%s] %s\n", v.Rule, v.Message)
				}
			}
		}
		// Per-section counts on success AND failure: trimming is informed,
		// not blind
		fmt.Println("\nSection words:")
		for _, row := range wordReport {
			cap := ""
			if row.Max > 0 {
				cap = fmt.Sprintf(" / %d", row.Max)
				if row.Words > row.Max {
					cap += "  ← over"
				}
			}
			fmt.Printf("  %-28s %d%s\n", row.Section, row.Words, cap)
		}
	}

	if len(violations) > 0 {
		return exitSilent(1)
	}
	return nil
}

// loadTemplateForDoc resolves the template (flag > frontmatter > legacy sidecar
// > unambiguous bundle collection) and loads the doc.
func loadTemplateForDoc(filename, templateName string) (*comment.Template, *comment.DocumentWithComments, error) {
	doc, err := loadDocument(filename)
	if err != nil {
		return nil, nil, failf("Error loading document: %v", err)
	}
	t, _, err := comment.ResolveTemplateForDocument(filename, doc.Content, templateName, doc.Template)
	if err != nil {
		return nil, nil, failf("Error: %v", err)
	}
	if t == nil {
		return nil, nil, failf("Error: no template specified; use --template, comments.template frontmatter, or a bundle collection with one template\nList templates with: comments template list")
	}
	return t, doc, nil
}
