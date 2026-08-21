package comment

import (
	"strings"
	"testing"
)

// coverageTemplate opts both ends in, the way research.yaml does.
func coverageTemplate() *Template {
	return &Template{
		Name: "t",
		Sections: []TemplateSection{
			{Heading: "Research Question", Required: true, EnumeratesQuestions: true},
			{Heading: "Findings", Required: true, AnswersQuestions: true},
		},
	}
}

func rules(violations []Violation) []string {
	out := make([]string, 0, len(violations))
	for _, v := range violations {
		out = append(out, v.Rule)
	}
	return out
}

func TestQuestionCoverageAllAnswered(t *testing.T) {
	content := `# Doc

## Research Question

Q1. What mechanism produces orphans?
Q2. What does the tool do about them?
Q3. What would reducing them require?

## Findings

### The cascade [Q1]

fact

### What the tool records [Q2]

fact

### The documented ceiling [Q3]

fact
`
	if v := ValidateTemplate(content, coverageTemplate()); len(v) != 0 {
		t.Errorf("fully covered question must pass, got %v", rules(v))
	}
}

// The exact defect this check exists for: a three-clause question answered in
// two. Every other check passed such a document.
func TestQuestionCoverageCatchesOmission(t *testing.T) {
	content := `# Doc

## Research Question

Q1. What mechanism produces orphans?
Q2. What does the tool do about them?
Q3. What would reducing them require?

## Findings

### The cascade [Q1]

fact

### What the tool records [Q2]

fact
`
	violations := ValidateTemplate(content, coverageTemplate())
	var uncovered []Violation
	for _, v := range violations {
		if v.Rule == "uncovered_question" {
			uncovered = append(uncovered, v)
		}
	}
	if len(uncovered) != 1 {
		t.Fatalf("expected exactly Q3 to be reported, got %v", rules(violations))
	}
	// Naming the clause is the point: "something is missing" is not actionable.
	if !strings.Contains(uncovered[0].Message, "Q3") {
		t.Errorf("violation must name the uncovered sub-question: %s", uncovered[0].Message)
	}
	if !strings.Contains(uncovered[0].Message, "reducing them require") {
		t.Errorf("violation must quote the sub-question text: %s", uncovered[0].Message)
	}
}

func TestQuestionCoverageRequiresEnumeration(t *testing.T) {
	content := `# Doc

## Research Question

What mechanism produces orphans, what does the tool do, and what would reducing them require?

## Findings

### The cascade

fact
`
	found := false
	for _, v := range ValidateTemplate(content, coverageTemplate()) {
		if v.Rule == "questions_not_enumerated" {
			found = true
		}
	}
	if !found {
		t.Error("a question left as prose must be reported: it cannot be checked for coverage")
	}
}

// A renumbering typo silently manufactures the appearance of coverage.
func TestQuestionCoverageUnknownClaim(t *testing.T) {
	content := `# Doc

## Research Question

Q1. What mechanism produces orphans?

## Findings

### The cascade [Q1]

fact

### Something else [Q7]

fact
`
	found := false
	for _, v := range ValidateTemplate(content, coverageTemplate()) {
		if v.Rule == "unknown_question" && strings.Contains(v.Message, "Q7") {
			found = true
		}
	}
	if !found {
		t.Error("a claim on a sub-question that was never asked must be reported")
	}
}

func TestQuestionCoverageMultipleClaimsPerFinding(t *testing.T) {
	content := `# Doc

## Research Question

Q1. First?
Q2. Second?

## Findings

### One finding covering both [Q1, Q2]

fact

### Another [Q1]

fact
`
	if v := ValidateTemplate(content, coverageTemplate()); len(v) != 0 {
		t.Errorf("one finding may answer several sub-questions, got %v", rules(v))
	}
}

func TestQuestionCoverageSeparateClaimsPerFinding(t *testing.T) {
	content := `# Doc

## Research Question

Q1. First?
Q2. Second?

## Findings

### One finding covering both [Q1] [Q2]

fact
`
	if v := ValidateTemplate(content, coverageTemplate()); len(v) != 0 {
		t.Errorf("separate tags should claim both sub-questions, got %v", rules(v))
	}
}

// Templates that do not opt in must be completely unaffected.
func TestQuestionCoverageOptOut(t *testing.T) {
	tpl := &Template{
		Name: "t",
		Sections: []TemplateSection{
			{Heading: "Research Question", Required: true},
			{Heading: "Findings", Required: true},
		},
	}
	content := "# Doc\n\n## Research Question\n\nprose only\n\n## Findings\n\n### A\n\nfact\n"
	for _, v := range ValidateTemplate(content, tpl) {
		if strings.Contains(v.Rule, "question") {
			t.Errorf("coverage checks must not fire when the template opts out: %+v", v)
		}
	}
}

func TestSubQuestionListSyntaxes(t *testing.T) {
	// Agents write lists several ways; all should parse.
	for _, form := range []string{"Q1. text", "- Q1) text", "* Q1: text", "  Q1 . text"} {
		content := "# D\n\n## Research Question\n\n" + form + "\n\n## Findings\n\n### A [Q1]\n\nfact\n"
		if v := ValidateTemplate(content, coverageTemplate()); len(v) != 0 {
			t.Errorf("form %q should parse as Q1, got %v", form, rules(v))
		}
	}
}
