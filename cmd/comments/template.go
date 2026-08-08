package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

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
			// Subsection bounds belong in the brief, not only in the validator:
			// an agent cannot follow a cap it is never shown, and a rule that
			// only surfaces as a late violation gets satisfied by trimming
			// rather than by writing tighter in the first place.
			switch {
			case s.MinSubsections > 0 && s.MaxSubsections > 0:
				flags = append(flags, fmt.Sprintf("%d-%d subsections", s.MinSubsections, s.MaxSubsections))
			case s.MinSubsections > 0:
				flags = append(flags, fmt.Sprintf(">=%d subsections", s.MinSubsections))
			case s.MaxSubsections > 0:
				flags = append(flags, fmt.Sprintf("<=%d subsections", s.MaxSubsections))
			}
			if s.MaxSubsectionWords > 0 {
				flags = append(flags, fmt.Sprintf("max %d words each", s.MaxSubsectionWords))
			}
			if s.EnumeratesQuestions {
				flags = append(flags, "enumerate sub-questions Q1., Q2., ...")
			}
			if s.AnswersQuestions {
				flags = append(flags, "tag each subsection [Q1]")
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

		// The marker budget is enforced by validate but was invisible here, so
		// agents guessed at it. Spend markers on what genuinely needs the human.
		if t.Markers.Max > 0 {
			fmt.Printf("\nAmbiguity markers: at most %d %s ...] per document.\n",
				t.Markers.Max, t.MarkerPrefix())
			fmt.Println("  Each marker is reported by validate and seeded as a blocking thread;")
			fmt.Println("  decide the rest yourself and record them as assumptions in the doc.")
		}

	default:
		return failf("Unknown template subcommand: %s (use list or show)", args[0])
	}
	return nil
}

// validateCommand handles `comments validate <file> --template <name> [--json]`
// Exit codes: 0 = conforms, 1 = violations or error.
func validateCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	templateName := fs.String("template", "", "Template name (defaults to template recorded in sidecar)")
	jsonOut := fs.Bool("json", false, "Output violations as JSON")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	t, doc, err := loadTemplateForDoc(filename, *templateName)
	if err != nil {
		return err
	}
	violations := comment.ValidateTemplate(doc.Content, t)
	// Citation resolvability needs the filesystem, so it runs here where the
	// document path is known rather than inside the pure content check.
	if t.Doc.CheckCitations {
		violations = append(violations, comment.ValidateCitations(doc.Content, filename)...)
	}
	wordReport := comment.SectionWordReport(doc.Content, t)

	if *jsonOut {
		payload := map[string]any{
			"file":          filename,
			"template":      t.Name,
			"conforms":      len(violations) == 0,
			"violations":    violations,
			"section_words": wordReport,
		}
		encoded, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(encoded))
	} else {
		// Markers are deliberate: an agent flags an ambiguity it refuses to
		// guess at, and seed turns each into a blocking thread. Lumping them in
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
			fmt.Printf("%d intentional marker(s), each becomes a blocking review thread:\n", len(markers))
			for _, v := range markers {
				fmt.Printf("  [%s] %s\n", v.Rule, v.Message)
			}
			fmt.Printf("\nSeed them with: comments seed %s --template %s --markers-only\n", filename, t.Name)
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

// seedCommand handles `comments seed <file> --template <name> [--author name]`
func seedCommand(filename string, args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	templateName := fs.String("template", "", "Template name (defaults to template recorded in sidecar)")
	author := fs.String("author", "template", "Author for seeded threads")
	markersOnly := fs.Bool("markers-only", false, "Seed only NEEDS CLARIFICATION markers, not generic criteria (agent posts specific callouts instead)")
	if err := fs.Parse(args); err != nil {
		return exitSilent(2)
	}

	t, doc, err := loadTemplateForDoc(filename, *templateName)
	if err != nil {
		return err
	}
	added := comment.SeedTemplateThreads(doc, t, *author, *markersOnly)

	if err := comment.SaveToSidecar(filename, doc); err != nil {
		return failf("Error saving document: %v", err)
	}

	if len(added) == 0 {
		fmt.Printf("✓ Nothing to seed — all template review threads already exist (template %q recorded)\n", t.Name)
		return nil
	}
	fmt.Printf("✓ Seeded %d review thread(s); template %q recorded — the gate now enforces its structure:\n", len(added), t.Name)
	for _, c := range added {
		marker := ""
		if c.Blocking {
			marker = " [blocking]"
		}
		fmt.Printf("  %s (line %d)%s %s\n", c.ID, c.Line, marker, comment.DecorateType(c.Text))
	}
	fmt.Println("\nReview = resolving these threads. Check progress with: comments gate", filename)
	return nil
}

// loadTemplateForDoc resolves the template (flag > sidecar record) and loads the doc
func loadTemplateForDoc(filename, templateName string) (*comment.Template, *comment.DocumentWithComments, error) {
	doc, err := loadDocument(filename)
	if err != nil {
		return nil, nil, failf("Error loading document: %v", err)
	}
	name := templateName
	if name == "" {
		name = doc.Template
	}
	if name == "" {
		return nil, nil, failf("Error: no template specified (--template <name>) and none recorded in sidecar\nList templates with: comments template list")
	}
	t, err := comment.LoadTemplateForDoc(name, filename)
	if err != nil {
		return nil, nil, failf("Error: %v", err)
	}
	return t, doc, nil
}
