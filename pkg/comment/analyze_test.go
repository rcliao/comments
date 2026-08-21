package comment

import (
	"os"
	"path/filepath"
	"testing"
)

const analyzeResearch = `# Research

## Research Question

Q1. How does the path work?
Q2. Where are its boundaries?

## Findings

### Renamed cascade [Q1] [Q2]

First body.

### F7 — security boundary [Q2]

Second body.

### Final concern [Q2]

Third body.
`

func writeAnalysisFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAnalyzeQuestionMapIncludesMultipleClaimsAndRenamedFindings(t *testing.T) {
	path := writeAnalysisFixture(t, "research.md", analyzeResearch)
	got := AnalyzeDocument(analyzeResearch, path, &Template{Name: "test"}, "", "")
	if !got.Ready || len(got.Questions) != 2 || len(got.Findings) != 3 {
		t.Fatalf("unexpected analysis: %+v", got)
	}
	if got.Findings[0].ID != "F1" || got.Findings[1].ID != "F7" {
		t.Fatalf("renamed finding fallback / explicit IDs wrong: %+v", got.Findings)
	}
	if len(got.Questions[0].Findings) != 1 || got.Questions[0].Findings[0] != "F1" {
		t.Fatalf("Q1 map = %+v, want F1", got.Questions[0].Findings)
	}
	if len(got.Questions[1].Findings) != 3 {
		t.Fatalf("Q2 should map to all three findings: %+v", got.Questions[1].Findings)
	}
}

func TestAnalyzeAgainstClassifiesRangesExclusionsAndUncovered(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	researchPath := filepath.Join(dir, "research.md")
	if err := os.WriteFile(researchPath, []byte(analyzeResearch), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := `# Plan

## Current State

The first two findings are used (research.md:10-16).

## What We're NOT Doing

The final concern is excluded (research.md:18).
`
	planPath := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}

	got := AnalyzeDocument(plan, planPath, &Template{Name: "test"}, analyzeResearch, researchPath)
	if !got.Ready || len(got.Coverage) != 3 {
		t.Fatalf("unexpected coverage: %+v", got)
	}
	if got.Coverage[0].Status != "cited" || got.Coverage[1].Status != "cited" || got.Coverage[2].Status != "excluded" {
		t.Fatalf("statuses = %q, %q, %q", got.Coverage[0].Status, got.Coverage[1].Status, got.Coverage[2].Status)
	}
	if got.Coverage[0].References[0].EndLine != 16 {
		t.Fatalf("range end not preserved: %+v", got.Coverage[0].References)
	}
}

func TestAnalyzeAgainstCountsFenceCommentTrailAndReportsUncovered(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	researchPath := filepath.Join(dir, "research.md")
	if err := os.WriteFile(researchPath, []byte(analyzeResearch), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := "# Plan\n\n## Current State\n\n```dbml\nfield int // research.md:18\n```\n"
	planPath := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(planPath, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}

	got := AnalyzeDocument(plan, planPath, &Template{Name: "test"}, analyzeResearch, researchPath)
	if got.Ready {
		t.Fatal("two uncited findings must keep analysis unready")
	}
	if got.Coverage[0].Status != "uncovered" || got.Coverage[1].Status != "uncovered" || got.Coverage[2].Status != "cited" {
		t.Fatalf("fence/uncovered statuses wrong: %+v", got.Coverage)
	}
}

func TestAnalyzeReportsCitationViolationsEvenWithoutTemplateOptIn(t *testing.T) {
	path := writeAnalysisFixture(t, "plan.md", "# Plan\n\nMissing pkg/nope.go:99.\n")
	got := AnalyzeDocument("# Plan\n\nMissing pkg/nope.go:99.\n", path, &Template{Name: "test"}, "", "")
	if got.Ready || len(got.CitationViolations) != 1 || got.CitationViolations[0].Rule != "unresolvable_citation" {
		t.Fatalf("broken citation not reported: %+v", got)
	}
}
