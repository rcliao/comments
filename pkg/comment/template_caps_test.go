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
