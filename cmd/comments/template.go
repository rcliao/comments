package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rcliao/comments/pkg/comment"
)

// templateCommand handles `comments template list|show <name>`
func templateCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: comments template list | comments template show <name>")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		templates, err := comment.ListTemplates()
		if err != nil {
			fmt.Printf("Error listing templates: %v\n", err)
			os.Exit(1)
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
			fmt.Println("Usage: comments template show <name>")
			os.Exit(1)
		}
		t, err := comment.LoadTemplate(args[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Template: %s\n%s\n\n", t.Name, t.Description)
		if t.Doc.MaxWords > 0 {
			fmt.Printf("Document cap: %d words\n\n", t.Doc.MaxWords)
		}
		fmt.Println("Sections:")
		for _, s := range t.Sections {
			flags := []string{}
			if s.Required {
				flags = append(flags, "required")
			}
			if s.MaxWords > 0 {
				flags = append(flags, fmt.Sprintf("max %d words", s.MaxWords))
			}
			if s.MinSubsections > 0 {
				flags = append(flags, fmt.Sprintf(">=%d subsections", s.MinSubsections))
			}
			if s.Zone != "" {
				flags = append(flags, "zone: "+s.Zone)
			}
			fmt.Printf("  ## %s", s.Heading)
			if len(flags) > 0 {
				fmt.Printf("  [%s]", strings.Join(flags, ", "))
			}
			fmt.Println()
			for _, c := range s.ReviewCriteria {
				fmt.Printf("     ✓ %s\n", c)
			}
		}

	default:
		fmt.Printf("Unknown template subcommand: %s (use list or show)\n", args[0])
		os.Exit(1)
	}
}

// validateCommand handles `comments validate <file> --template <name> [--json]`
// Exit codes: 0 = conforms, 1 = violations or error.
func validateCommand(filename string, args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	templateName := fs.String("template", "", "Template name (defaults to template recorded in sidecar)")
	jsonOut := fs.Bool("json", false, "Output violations as JSON")
	fs.Parse(args)

	t, doc := loadTemplateForDoc(filename, *templateName)
	violations := comment.ValidateTemplate(doc.Content, t)

	if *jsonOut {
		payload := map[string]any{
			"file":       filename,
			"template":   t.Name,
			"conforms":   len(violations) == 0,
			"violations": violations,
		}
		encoded, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(encoded))
	} else if len(violations) == 0 {
		fmt.Printf("✓ %s conforms to template %q\n", filename, t.Name)
	} else {
		fmt.Printf("✗ %s has %d violation(s) against template %q:\n\n", filename, len(violations), t.Name)
		for _, v := range violations {
			fmt.Printf("  [%s] %s\n", v.Rule, v.Message)
		}
	}

	if len(violations) > 0 {
		os.Exit(1)
	}
}

// seedCommand handles `comments seed <file> --template <name> [--author name]`
func seedCommand(filename string, args []string) {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	templateName := fs.String("template", "", "Template name (defaults to template recorded in sidecar)")
	author := fs.String("author", "template", "Author for seeded threads")
	markersOnly := fs.Bool("markers-only", false, "Seed only NEEDS CLARIFICATION markers, not generic criteria (agent posts specific callouts instead)")
	fs.Parse(args)

	t, doc := loadTemplateForDoc(filename, *templateName)
	added := comment.SeedTemplateThreads(doc, t, *author, *markersOnly)

	if err := comment.SaveToSidecar(filename, doc); err != nil {
		fmt.Printf("Error saving document: %v\n", err)
		os.Exit(1)
	}

	if len(added) == 0 {
		fmt.Printf("✓ Nothing to seed — all template review threads already exist (template %q recorded)\n", t.Name)
		return
	}
	fmt.Printf("✓ Seeded %d review thread(s) from template %q:\n", len(added), t.Name)
	for _, c := range added {
		marker := ""
		if c.Blocking {
			marker = " [blocking]"
		}
		fmt.Printf("  %s (line %d)%s %s\n", c.ID, c.Line, marker, c.Text)
	}
	fmt.Println("\nReview = resolving these threads. Check progress with: comments gate", filename)
}

// loadTemplateForDoc resolves the template (flag > sidecar record) and loads the doc
func loadTemplateForDoc(filename, templateName string) (*comment.Template, *comment.DocumentWithComments) {
	doc, err := comment.LoadFromSidecar(filename)
	if err != nil {
		fmt.Printf("Error loading document: %v\n", err)
		os.Exit(1)
	}
	name := templateName
	if name == "" {
		name = doc.Template
	}
	if name == "" {
		fmt.Println("Error: no template specified (--template <name>) and none recorded in sidecar")
		fmt.Println("List templates with: comments template list")
		os.Exit(1)
	}
	t, err := comment.LoadTemplate(name)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	return t, doc
}
