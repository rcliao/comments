package comment

// Phase 3 of docs/plan-agent-surface.md: reproduction of the external field
// report — a section over its cap only via its subsections must violate.

import (
	"fmt"
	"strings"
	"testing"
)

func TestSectionCapCountsSubsections(t *testing.T) {
	tpl := &Template{
		Name: "t",
		Sections: []TemplateSection{
			{Heading: "Design", Required: true, MaxWords: 50},
		},
	}
	// Preamble tiny; the overage lives inside a subsection
	content := "# Doc\n\n## Design\n\nshort preamble\n\n### Detail\n\n" +
		strings.Repeat("word ", 100) + "\n"
	violations := ValidateTemplate(content, tpl)
	found := false
	for _, v := range violations {
		if v.Rule == "over_length" && v.Section == "Design" {
			found = true
		}
	}
	if !found {
		t.Fatalf("subsection words must count toward the section cap; violations: %+v", violations)
	}
}

// Body sections are where review iteration accumulates: measured across the
// shipped RPI corpus, every over-cap doc was over because a few subsections
// grew, not because the whole section drifted up. A section-wide cap alone
// tells the agent to "trim somewhere", which is how padding survives.
func TestSubsectionWordCap(t *testing.T) {
	tpl := &Template{
		Name: "t",
		Sections: []TemplateSection{
			{Heading: "Findings", Required: true, MaxSubsectionWords: 50},
		},
	}
	content := "# Doc\n\n## Findings\n\n### F1 — fine\n\n" +
		strings.Repeat("word ", 10) + "\n\n### F2 — bloated\n\n" +
		strings.Repeat("word ", 80) + "\n"

	violations := ValidateTemplate(content, tpl)

	var named []string
	for _, v := range violations {
		if v.Rule == "subsection_over_length" {
			named = append(named, v.Message)
		}
	}
	if len(named) != 1 {
		t.Fatalf("expected exactly the bloated subsection to fail, got %d: %+v", len(named), violations)
	}
	// The message must name the offending subsection, or the agent cannot tell
	// which one to rewrite.
	if !strings.Contains(named[0], "F2 — bloated") {
		t.Errorf("violation must name the subsection, got: %s", named[0])
	}
	if strings.Contains(named[0], "F1") {
		t.Errorf("the compliant subsection must not be reported: %s", named[0])
	}
}

func TestMaxSubsections(t *testing.T) {
	tpl := &Template{
		Name: "t",
		Sections: []TemplateSection{
			{Heading: "Findings", Required: true, MinSubsections: 2, MaxSubsections: 3},
		},
	}
	body := func(n int) string {
		s := "# Doc\n\n## Findings\n"
		for i := 1; i <= n; i++ {
			s += fmt.Sprintf("\n### F%d\n\nfact\n", i)
		}
		return s
	}

	if v := ValidateTemplate(body(3), tpl); len(v) != 0 {
		t.Errorf("3 subsections is at the cap and must pass, got %+v", v)
	}

	found := false
	for _, v := range ValidateTemplate(body(4), tpl) {
		if v.Rule == "max_subsections" {
			found = true
			// Splitting to satisfy a word cap must not be the escape hatch.
			if !strings.Contains(v.Message, "do not split to fit") {
				t.Errorf("max_subsections should warn against splitting, got: %s", v.Message)
			}
		}
	}
	if !found {
		t.Error("4 subsections must violate a max of 3")
	}
}

// A cap of 0 means unlimited; adding the fields must not change templates that
// do not set them.
func TestSubsectionCapsOptOut(t *testing.T) {
	tpl := &Template{
		Name:     "t",
		Sections: []TemplateSection{{Heading: "Findings", Required: true}},
	}
	content := "# Doc\n\n## Findings\n\n### F1\n\n" + strings.Repeat("word ", 500) + "\n"
	for _, v := range ValidateTemplate(content, tpl) {
		if v.Rule == "subsection_over_length" || v.Rule == "max_subsections" {
			t.Errorf("unset caps must not fire: %+v", v)
		}
	}
}

// A template that asks for peekable file:line evidence in its review criteria
// must also enable the check that verifies the reference resolves. The criteria
// alone are a promise; check_citations is what keeps it. Pinned because this
// repo's hand-maintained lists drift.
func TestEvidenceTemplatesEnforceTheirPromises(t *testing.T) {
	for _, name := range []string{"research", "plan", "design-doc", "as-built"} {
		tpl, err := LoadTemplate(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !tpl.Doc.CheckCitations {
			t.Errorf("%s asks for peekable evidence but does not set check_citations", name)
		}
		if tpl.Doc.Style.MaxParagraphWords == 0 || tpl.Doc.Style.MaxSentenceWords == 0 {
			t.Errorf("%s has no style caps; review artifacts should be scannable", name)
		}
	}
}

// Citations are exempt from word caps: measured at ~12% of a section's budget
// (scripts/eval/logs/cap-pilot-2026-08-11.json), the cite-every-claim rule was
// costing a fact per section. Evidence must never compete with content.
func TestWordCapsExemptCitations(t *testing.T) {
	tpl := &Template{
		Name:     "t",
		Doc:      TemplateDocRules{MaxWords: 12},
		Sections: []TemplateSection{{Heading: "Design", Required: true, MaxWords: 7}},
	}
	// Seven real words in the section, every claim cited — ten, and over cap,
	// only if the three citation tokens are counted.
	content := "# Doc\n\n## Design\n\nThe gate exits ten (pkg/comment/gate.go:59)\n" +
		"when blocking `cmd/comments/gate.go:114-116` remain thread:c1abc\n"

	for _, v := range ValidateTemplate(content, tpl) {
		if v.Rule == "over_length" || v.Rule == "doc_over_length" {
			t.Errorf("citations must not count toward caps, got %s: %s", v.Rule, v.Message)
		}
	}

	report := SectionWordReport(content, tpl)
	var design *SectionWordCount
	for i := range report {
		if report[i].Section == "Design" {
			design = &report[i]
		}
	}
	if design == nil {
		t.Fatal("Design missing from word report")
	}
	if design.Words != 7 {
		t.Errorf("Design counts %d words, want 7 (citations exempt)", design.Words)
	}
}
