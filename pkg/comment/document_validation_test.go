package comment

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateDocumentPreservesStructuralThenCitationOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(dir, "research.md")
	content := "# R\n\n## Research Question\n\nQ1. Covered?\nQ2. Missing?\n\n## Findings\n\n### F1 [Q1]\n\nSee pkg/nope.go:9.\n\n[NEEDS CLARIFICATION: one]\n[NEEDS CLARIFICATION: two]\n"
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	tpl := &Template{
		Name: "check",
		Doc:  TemplateDocRules{CheckCitations: true},
		Sections: []TemplateSection{
			{Heading: "Research Question", Required: true, EnumeratesQuestions: true},
			{Heading: "Summary", Required: true},
			{Heading: "Findings", Required: true, AnswersQuestions: true},
		},
		Markers: TemplateMarkers{NeedsClarification: "[NEEDS CLARIFICATION:", Max: 1},
	}

	var got []string
	for _, v := range ValidateDocument(content, docPath, tpl) {
		got = append(got, v.Rule)
	}
	want := []string{"missing_section", "uncovered_question", "unresolved_marker", "unresolved_marker", "too_many_markers", "unresolvable_citation"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rules/order = %v, want %v", got, want)
	}
}
