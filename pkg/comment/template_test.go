package comment

import (
	"strings"
	"testing"
)

const testTemplateYAML = `template: test-doc
version: 1
description: test
doc:
  max_words: 500
sections:
  - heading: "Problem"
    required: true
    max_words: 20
    zone: human
    review_criteria:
      - "Is the problem clear?"
  - heading: "Options Considered"
    required: true
    min_subsections: 2
  - heading: "Unresolved Questions"
    required: true
    zone: human
markers:
  needs_clarification: "[NEEDS CLARIFICATION"
`

const conformingDoc = `# My Design

## Problem

Things are slow.

## Options Considered

### Option A

Fast thing.

### Option B

Slow thing.

## Unresolved Questions

None.
`

func testTemplate(t *testing.T) *Template {
	tmpl, err := parseTemplate([]byte(testTemplateYAML))
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func TestValidateTemplateConforming(t *testing.T) {
	violations := ValidateTemplate(conformingDoc, testTemplate(t))
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %v", violations)
	}
}

func TestValidateTemplateViolations(t *testing.T) {
	doc := `# My Design

## Problem

` + strings.Repeat("word ", 30) + `

## Options Considered

### Only Option

Just one.

Something [NEEDS CLARIFICATION: which?] here.
`
	violations := ValidateTemplate(doc, testTemplate(t))

	rules := map[string]bool{}
	for _, v := range violations {
		rules[v.Rule] = true
	}
	for _, expected := range []string{"missing_section", "over_length", "min_subsections", "unresolved_marker"} {
		if !rules[expected] {
			t.Errorf("expected violation rule %q, got %v", expected, violations)
		}
	}
}

func TestComputeSeedTargets(t *testing.T) {
	doc := strings.Replace(conformingDoc, "Things are slow.", "Things are [NEEDS CLARIFICATION: how slow?] slow.", 1)
	targets := ComputeSeedTargets(doc, testTemplate(t), false)

	var criteria, markers int
	for _, target := range targets {
		switch target.Type {
		case "T":
			criteria++
			if target.Line != 3 {
				t.Errorf("criteria should anchor at Problem heading (line 3), got %d", target.Line)
			}
		case "Q":
			markers++
			if !target.Blocking {
				t.Error("marker threads must be blocking")
			}
		}
	}
	if criteria != 1 || markers != 1 {
		t.Errorf("expected 1 criteria + 1 marker target, got %d + %d", criteria, markers)
	}
}

func TestSeedTemplateThreadsIdempotent(t *testing.T) {
	doc := &DocumentWithComments{Content: conformingDoc}
	tmpl := testTemplate(t)

	first := SeedTemplateThreads(doc, tmpl, "template", false)
	if len(first) != 1 {
		t.Fatalf("expected 1 seeded thread, got %d", len(first))
	}
	if doc.Template != "test-doc" {
		t.Errorf("seeding should record template name, got %q", doc.Template)
	}

	second := SeedTemplateThreads(doc, tmpl, "template", false)
	if len(second) != 0 {
		t.Errorf("re-seeding should add nothing, got %d", len(second))
	}
}

func TestSectionZone(t *testing.T) {
	tmpl := testTemplate(t)
	// Line 5 = "Things are slow." inside Problem (zone: human)
	if zone := SectionZone(conformingDoc, tmpl, 5); zone != ZoneHuman {
		t.Errorf("expected human zone for Problem body, got %q", zone)
	}
	// Line 9 = Option A area, no zone declared
	if zone := SectionZone(conformingDoc, tmpl, 9); zone != "" {
		t.Errorf("expected no zone for Options Considered, got %q", zone)
	}
}

func TestLoadBuiltinTemplates(t *testing.T) {
	for _, name := range []string{"design-doc", "adr", "rfc"} {
		tmpl, err := LoadTemplate(name)
		if err != nil {
			t.Errorf("built-in template %q failed to load: %v", name, err)
			continue
		}
		if tmpl.Name != name {
			t.Errorf("template %q has mismatched name %q", name, tmpl.Name)
		}
	}
}

func TestSeedMarkersOnly(t *testing.T) {
	doc := strings.Replace(conformingDoc, "Things are slow.", "Things are [NEEDS CLARIFICATION: how slow?] slow.", 1)
	targets := ComputeSeedTargets(doc, testTemplate(t), true)

	if len(targets) != 1 || targets[0].Type != "Q" {
		t.Errorf("markers-only should seed only the marker Q thread, got %v", targets)
	}
}
