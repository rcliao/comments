package comment

// Phase 3 of docs/plan-agent-surface.md: reproduction of the external field
// report — a section over its cap only via its subsections must violate.

import (
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
