package comment

import (
	"os"
	"path/filepath"
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

func TestHeadingMatchesPath(t *testing.T) {
	cases := []struct {
		name    string
		heading string
		path    string
		want    bool
	}{
		{"exact title match", "Problem", "Problem", true},
		{"suffix false positive rejected", "Problem", "Big Problem", false},
		{"prefix mismatch rejected", "Problem", "Problems", false},
		{"Design does not match Redesign", "Design", "Redesign", false},
		{"single segment matches last path segment", "Options Considered", "Design > Options Considered", true},
		{"segment boundary respected in path", "Options Considered", "Design > More Options Considered", false},
		{"multi-segment suffix matches whole trailing segments", "Impl > Details", "Doc > Impl > Details", true},
		{"multi-segment suffix exact path", "Impl > Details", "Impl > Details", true},
		{"multi-segment partial first segment rejected", "Impl > Details", "Doc > MyImpl > Details", false},
		{"multi-segment partial last segment rejected", "Impl > Details", "Doc > Impl > More Details", false},
		{"heading longer than path rejected", "Doc > Impl > Details", "Impl > Details", false},
		{"case sensitive", "problem", "Problem", false},
		{"middle segment alone does not match full path tail mismatch", "Impl", "Doc > Impl > Details", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := headingMatchesPath(tc.heading, tc.path); got != tc.want {
				t.Errorf("headingMatchesPath(%q, %q) = %v, want %v", tc.heading, tc.path, got, tc.want)
			}
		})
	}
}

func TestValidateTemplateNoSuffixFalsePositive(t *testing.T) {
	// "Big Problem" must NOT satisfy the required "Problem" section.
	doc := `# My Design

## Big Problem

Things are slow.

## Options Considered

### Option A

A.

### Option B

B.

## Unresolved Questions

None.
`
	violations := ValidateTemplate(doc, testTemplate(t))
	found := false
	for _, v := range violations {
		if v.Rule == "missing_section" && v.Section == "Problem" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_section for %q (Big Problem must not match), got %v", "Problem", violations)
	}
}

const zoneBoundaryTemplateYAML = `template: zone-test
version: 1
sections:
  - heading: "Risks"
    required: true
    zone: human
    review_criteria:
      - "Are the risks real?"
`

// Doc where the near-miss section comes first, so the old suffix matching
// would have latched onto "More Risks" before reaching "Risks".
const zoneBoundaryDoc = `# Doc

## More Risks

Not the human zone.

## Risks

The human zone.
`

func TestSectionZoneSegmentBoundary(t *testing.T) {
	tmpl, err := parseTemplate([]byte(zoneBoundaryTemplateYAML))
	if err != nil {
		t.Fatal(err)
	}
	// Line 5 is inside "More Risks" — must NOT be captured by zone:human "Risks".
	if zone := SectionZone(zoneBoundaryDoc, tmpl, 5); zone != "" {
		t.Errorf("expected no zone inside %q, got %q", "More Risks", zone)
	}
	// Line 9 is inside "Risks" — must be human.
	if zone := SectionZone(zoneBoundaryDoc, tmpl, 9); zone != ZoneHuman {
		t.Errorf("expected human zone inside %q, got %q", "Risks", zone)
	}
}

func TestComputeSeedTargetsSegmentBoundary(t *testing.T) {
	tmpl, err := parseTemplate([]byte(zoneBoundaryTemplateYAML))
	if err != nil {
		t.Fatal(err)
	}
	targets := ComputeSeedTargets(zoneBoundaryDoc, tmpl, false)
	if len(targets) != 1 {
		t.Fatalf("expected 1 seed target, got %d: %v", len(targets), targets)
	}
	// "Risks" heading is line 7; the old suffix match would anchor at
	// "More Risks" (line 3).
	if targets[0].Line != 7 {
		t.Errorf("criteria must anchor at the Risks heading (line 7), got %d", targets[0].Line)
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

func TestLoadTemplateForDocFindsProjectTemplates(t *testing.T) {
	// Project layout: <tmp>/proj/.comments/templates/proj-tmpl.yaml with the
	// document nested two levels down. Discovery must walk up from the doc's
	// directory, independent of the process cwd.
	tmp := t.TempDir()
	tmplDir := filepath.Join(tmp, "proj", ".comments", "templates")
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		t.Fatal(err)
	}
	yaml := "template: proj-tmpl\nversion: 1\ndescription: project-local\n"
	if err := os.WriteFile(filepath.Join(tmplDir, "proj-tmpl.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	docDir := filepath.Join(tmp, "proj", "specs", "001-feature")
	if err := os.MkdirAll(docDir, 0755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(docDir, "spec.md")

	tmpl, err := LoadTemplateForDoc("proj-tmpl", docPath)
	if err != nil {
		t.Fatalf("LoadTemplateForDoc failed: %v", err)
	}
	if tmpl.Name != "proj-tmpl" || tmpl.Description != "project-local" {
		t.Errorf("wrong template loaded: %+v", tmpl)
	}

	// Built-ins still resolve for the same document
	if _, err := LoadTemplateForDoc("design-doc", docPath); err != nil {
		t.Errorf("built-in fallback failed: %v", err)
	}

	// Unknown names list both project and built-in templates
	_, err = LoadTemplateForDoc("nope", docPath)
	if err == nil || !strings.Contains(err.Error(), "proj-tmpl") {
		t.Errorf("error should list project template, got: %v", err)
	}

	names, err := ListTemplatesForDoc(docPath)
	if err != nil {
		t.Fatalf("ListTemplatesForDoc failed: %v", err)
	}
	if names["proj-tmpl"] != "project" {
		t.Errorf("proj-tmpl source = %q, want project", names["proj-tmpl"])
	}
	if names["design-doc"] != "built-in" {
		t.Errorf("design-doc source = %q, want built-in", names["design-doc"])
	}
}

func TestLoadTemplateForDocNoProjectDir(t *testing.T) {
	// A document outside any project: only built-ins resolve
	docPath := filepath.Join(t.TempDir(), "doc.md")
	if _, err := LoadTemplateForDoc("design-doc", docPath); err != nil {
		t.Errorf("built-in load failed without project dir: %v", err)
	}
	if _, err := LoadTemplateForDoc("proj-tmpl", docPath); err == nil {
		t.Error("expected error for unknown template outside a project")
	}
}
