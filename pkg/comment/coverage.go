package comment

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
)

// Question coverage makes "did this document answer the whole question?" a
// structural property instead of a judgment call.
//
// The failure it exists to catch: a three-clause research question produced a
// document answering two clauses, which passed every check the tool had —
// word caps, citations, tone — because conformance and faithfulness say nothing
// about omission. A narrow-but-accurate doc scored perfectly while a third of
// the question silently vanished.
//
// The fix is to force the decomposition up front. A question written as
// enumerated sub-questions is a better question, and it turns coverage into
// something a validator can check without reading for meaning.
var (
	// enumeratedQuestion matches "Q1." / "- Q2)" / "Q3:" at the start of a line.
	enumeratedQuestion = regexp.MustCompile(`(?m)^\s*(?:[-*]\s*)?Q(\d+)\s*[.):]\s*(.*)$`)
	// claimedQuestions matches a "[Q1]" or "[Q1, Q3]" tag in a finding heading.
	claimedQuestions = regexp.MustCompile(`\[\s*(Q\d+(?:\s*,\s*Q?\d+)*)\s*\]`)
	questionID       = regexp.MustCompile(`\d+`)
)

// SubQuestion is one enumerated clause of a research question.
type SubQuestion struct {
	ID   string // "Q1"
	Text string
	Line int
}

// parseSubQuestions extracts the enumerated sub-questions from a section body.
func parseSubQuestions(lines []string, section *markdown.Section) []SubQuestion {
	var out []SubQuestion
	for i := section.StartLine; i < section.EndLine && i < len(lines); i++ {
		m := enumeratedQuestion.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		out = append(out, SubQuestion{
			ID:   "Q" + m[1],
			Text: strings.TrimSpace(m[2]),
			Line: i + 1,
		})
	}
	return out
}

// parseClaims returns the set of sub-question IDs claimed by a section's
// subsection headings, and where each claim was made.
func parseClaims(section *markdown.Section) map[string]int {
	claims := map[string]int{}
	for _, child := range section.Children {
		m := claimedQuestions.FindStringSubmatch(child.Title)
		if m == nil {
			continue
		}
		for _, part := range strings.Split(m[1], ",") {
			if n := questionID.FindString(part); n != "" {
				claims["Q"+n] = child.StartLine
			}
		}
	}
	return claims
}

// validateQuestionCoverage cross-checks the enumerated sub-questions against
// the findings that claim to answer them.
func validateQuestionCoverage(lines []string, asks, answers *markdown.Section, asksHeading, answersHeading string) []Violation {
	var violations []Violation

	questions := parseSubQuestions(lines, asks)
	if len(questions) == 0 {
		return []Violation{{
			Rule:    "questions_not_enumerated",
			Section: asksHeading,
			Line:    asks.StartLine,
			Message: fmt.Sprintf("section %q must enumerate its sub-questions as \"Q1. ...\", one per clause — a question that is not decomposed cannot be checked for coverage", asksHeading),
		}}
	}

	claims := parseClaims(answers)
	known := map[string]bool{}
	for _, q := range questions {
		known[q.ID] = true
	}

	for _, q := range questions {
		if _, ok := claims[q.ID]; !ok {
			text := q.Text
			if len(text) > 60 {
				text = text[:60] + "…"
			}
			violations = append(violations, Violation{
				Rule:    "uncovered_question",
				Section: answersHeading,
				Line:    q.Line,
				Message: fmt.Sprintf("%s is never answered: %q — add a subsection under %q tagged [%s], or drop the sub-question",
					q.ID, text, answersHeading, q.ID),
			})
		}
	}

	// A claim on a question that does not exist is usually a renumbering typo,
	// and it silently creates the illusion of coverage.
	for id, line := range claims {
		if !known[id] {
			violations = append(violations, Violation{
				Rule:    "unknown_question",
				Section: answersHeading,
				Line:    line,
				Message: fmt.Sprintf("a subsection claims to answer %s, which %q does not enumerate", id, asksHeading),
			})
		}
	}
	return violations
}
