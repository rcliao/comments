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
	for _, name := range []string{"design-doc", "adr", "rfc", "mini", "research", "plan"} {
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

const cappedMarkersYAML = `template: capped
version: 1
sections:
  - heading: "Problem"
    required: true
markers:
  needs_clarification: "[NEEDS CLARIFICATION"
  max: 2
`

func TestMarkerCap(t *testing.T) {
	tmpl, err := parseTemplate([]byte(cappedMarkersYAML))
	if err != nil {
		t.Fatal(err)
	}

	countRule := func(vs []Violation, rule string) int {
		n := 0
		for _, v := range vs {
			if v.Rule == rule {
				n++
			}
		}
		return n
	}

	atCap := "# D\n\n## Problem\n\nA [NEEDS CLARIFICATION: x?]\nB [NEEDS CLARIFICATION: y?]\n"
	vs := ValidateTemplate(atCap, tmpl)
	if got := countRule(vs, "too_many_markers"); got != 0 {
		t.Errorf("2 markers at cap 2 must not trip the cap, got %d violation(s)", got)
	}
	if got := countRule(vs, "unresolved_marker"); got != 2 {
		t.Errorf("individual markers still report: want 2 unresolved_marker, got %d", got)
	}

	overCap := atCap + "C [NEEDS CLARIFICATION: z?]\n"
	vs = ValidateTemplate(overCap, tmpl)
	if got := countRule(vs, "too_many_markers"); got != 1 {
		t.Fatalf("3 markers over cap 2 must add one too_many_markers violation, got %d", got)
	}
	for _, v := range vs {
		if v.Rule == "too_many_markers" && !strings.Contains(v.Message, "record them as assumptions") {
			t.Errorf("cap message should tell the agent to downgrade to assumptions, got %q", v.Message)
		}
	}

	// Uncapped template (max absent) never trips regardless of count
	uncapped := testTemplate(t)
	if got := countRule(ValidateTemplate(overCap, uncapped), "too_many_markers"); got != 0 {
		t.Errorf("template without markers.max must not cap, got %d", got)
	}
}

func TestMiniTemplateWorkflow(t *testing.T) {
	tmpl, err := LoadTemplate("mini")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Markers.Max != 2 {
		t.Errorf("mini should cap markers at 2, got %d", tmpl.Markers.Max)
	}

	conforming := "# Fix\n\n## Problem\n\nList table preview splits multibyte runes.\n\n## Change\n\nUse the rune-safe truncate helper in outputTable.\n\n## Definition of Done\n\n- automated: go test ./cmd/... passes with a CJK-text fixture\n"
	if vs := ValidateTemplate(conforming, tmpl); len(vs) != 0 {
		t.Errorf("conforming mini doc should validate clean, got %v", vs)
	}

	missing := "# Fix\n\n## Problem\n\nX.\n\n## Change\n\nY.\n"
	vs := ValidateTemplate(missing, tmpl)
	found := false
	for _, v := range vs {
		if v.Rule == "missing_section" && strings.Contains(v.Message, "Definition of Done") {
			found = true
		}
	}
	if !found {
		t.Errorf("mini must require Definition of Done, got %v", vs)
	}
}

func TestRPITemplates(t *testing.T) {
	research, err := LoadTemplate("research")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := LoadTemplate("plan")
	if err != nil {
		t.Fatal(err)
	}
	if research.Markers.Max != 3 || plan.Markers.Max != 1 {
		t.Errorf("marker caps: research want 3 got %d, plan want 1 got %d",
			research.Markers.Max, plan.Markers.Max)
	}

	// A conforming research doc decomposes its question and tags each finding
	// with the clause it answers; coverage is part of conformance now.
	conformingResearch := "# R\n\n## Research Question\n\nQ1. What?\nQ2. Why?\n\n## Summary\n\nThis.\n\n## Findings\n\n### F1 [Q1]\n\nOne, per a.go:1.\n\n### F2 [Q2]\n\nTwo.\n\n## Code References\n\n- a.go:1\n\n## Open Questions\n\nNone.\n"
	if vs := ValidateTemplate(conformingResearch, research); len(vs) != 0 {
		t.Errorf("conforming research doc should validate clean, got %v", vs)
	}
	// findings as one wall of prose (no subsections) must fail
	prose := "# R\n\n## Research Question\n\nQ1. What?\n\n## Summary\n\nThis.\n\n## Findings\n\nJust prose.\n\n## Code References\n\n- a.go:1\n\n## Open Questions\n\nNone.\n"
	if vs := ValidateTemplate(prose, research); !hasRule(vs, "min_subsections") {
		t.Errorf("prose findings must fail min_subsections, got %v", vs)
	}

	conformingPlan := "# P\n\n## Overview\n\nBuild X.\n\n## Current State\n\nY exists (y.go:3).\n\n## Desired End State\n\nX works; verify by test.\n\n## What We're NOT Doing\n\nNo Z.\n\n## Implementation Phases\n\n### Phase 1\n\nDo A. Success: automated go test; manual eyeball.\n\n### Phase 2\n\nDo B. Success: automated lint; manual review.\n"
	if vs := ValidateTemplate(conformingPlan, plan); len(vs) != 0 {
		t.Errorf("conforming plan doc should validate clean, got %v", vs)
	}
	// two markers exceed the plan cap of 1
	twoMarkers := strings.Replace(conformingPlan, "Do A.", "Do A [NEEDS CLARIFICATION: a?]\nand [NEEDS CLARIFICATION: b?].", 1)
	if vs := ValidateTemplate(twoMarkers, plan); !hasRule(vs, "too_many_markers") {
		t.Errorf("2 markers must exceed plan cap of 1, got %v", vs)
	}
}

func hasRule(vs []Violation, rule string) bool {
	for _, v := range vs {
		if v.Rule == rule {
			return true
		}
	}
	return false
}
