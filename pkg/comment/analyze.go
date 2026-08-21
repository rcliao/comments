package comment

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
)

var findingID = regexp.MustCompile(`(?i)^\s*(F\d+)\b`)

// Analysis is a deterministic artifact report. It exposes the coverage and
// evidence facts an agent needs without pretending to judge whether prose is
// true. Semantic disagreements remain ordinary review threads.
type Analysis struct {
	File               string             `json:"file"`
	Template           string             `json:"template,omitempty"`
	Ready              bool               `json:"ready"`
	StructureUnchecked bool               `json:"structure_unchecked,omitempty"`
	Violations         []Violation        `json:"violations"`
	CitationViolations []Violation        `json:"citation_violations"`
	Questions          []AnalysisQuestion `json:"questions"`
	Findings           []AnalysisFinding  `json:"findings"`
	Against            string             `json:"against,omitempty"`
	Coverage           []FindingCoverage  `json:"coverage,omitempty"`
}

type AnalysisQuestion struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Line     int      `json:"line"`
	Findings []string `json:"findings"`
}

type AnalysisFinding struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Questions []string `json:"questions,omitempty"`
}

type FindingCoverage struct {
	Finding    AnalysisFinding     `json:"finding"`
	Status     string              `json:"status"` // cited, excluded, uncovered
	References []CoverageReference `json:"references"`
}

type CoverageReference struct {
	Raw          string `json:"raw"`
	DocumentLine int    `json:"document_line"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	Exclusion    bool   `json:"exclusion"`
}

// AnalyzeDocument builds the mechanical report for one artifact. Template may
// be nil, in which case the report is intentionally not ready: an untemplated
// document has no declared structural contract.
func AnalyzeDocument(content, docPath string, t *Template, againstContent, againstPath string) Analysis {
	absPath, _ := filepath.Abs(docPath)
	result := Analysis{
		File:       absPath,
		Ready:      true,
		Violations: []Violation{},
		Questions:  []AnalysisQuestion{},
		Findings:   []AnalysisFinding{},
	}

	if t == nil {
		result.StructureUnchecked = true
		result.Ready = false
		result.CitationViolations = ValidateCitations(content, docPath)
		result.Violations = append(result.Violations, result.CitationViolations...)
	} else {
		result.Template = t.Name
		result.Violations = ValidateDocument(content, docPath, t)
		if !t.Doc.CheckCitations {
			result.Violations = append(result.Violations, ValidateCitations(content, docPath)...)
		}
		for _, v := range result.Violations {
			if v.Rule == "unresolvable_citation" || v.Rule == "ambiguous_citation" {
				result.CitationViolations = append(result.CitationViolations, v)
			}
		}
		if len(result.Violations) > 0 {
			result.Ready = false
		}
	}

	result.Questions, result.Findings = analyzeQuestionMap(content)

	if againstPath != "" {
		absAgainst, _ := filepath.Abs(againstPath)
		result.Against = absAgainst
		_, againstFindings := analyzeQuestionMap(againstContent)
		result.Coverage = analyzeFindingCoverage(content, docPath, againstPath, againstFindings)
		if len(againstFindings) == 0 {
			result.Ready = false
			result.Violations = append(result.Violations, Violation{
				Rule:    "against_has_no_findings",
				Message: fmt.Sprintf("%s has no subsections under Findings", againstPath),
			})
		}
		for _, c := range result.Coverage {
			if c.Status == "uncovered" {
				result.Ready = false
			}
		}
	}

	if result.CitationViolations == nil {
		result.CitationViolations = []Violation{}
	}
	return result
}

func analyzeQuestionMap(content string) ([]AnalysisQuestion, []AnalysisFinding) {
	structure := markdown.ParseDocument(content)
	questionSection := sectionNamed(structure, "Research Question")
	findingsSection := sectionNamed(structure, "Findings")
	lines := strings.Split(content, "\n")

	questions := []AnalysisQuestion{}
	if questionSection != nil {
		for _, q := range parseSubQuestions(lines, questionSection) {
			questions = append(questions, AnalysisQuestion{ID: q.ID, Text: q.Text, Line: q.Line, Findings: []string{}})
		}
	}

	findings := []AnalysisFinding{}
	if findingsSection != nil {
		for i, child := range findingsSection.Children {
			id := fmt.Sprintf("F%d", i+1)
			if m := findingID.FindStringSubmatch(child.Title); m != nil {
				id = strings.ToUpper(m[1])
			}
			finding := AnalysisFinding{
				ID:        id,
				Title:     child.Title,
				StartLine: child.StartLine,
				EndLine:   child.EndLine,
				Questions: questionTags(child.Title),
			}
			findings = append(findings, finding)
		}
	}

	byQuestion := map[string][]string{}
	for _, f := range findings {
		for _, q := range f.Questions {
			byQuestion[q] = append(byQuestion[q], f.ID)
		}
	}
	for i := range questions {
		questions[i].Findings = byQuestion[questions[i].ID]
		if questions[i].Findings == nil {
			questions[i].Findings = []string{}
		}
	}
	return questions, findings
}

func questionTags(title string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range claimedQuestions.FindAllStringSubmatch(title, -1) {
		for _, part := range strings.Split(m[1], ",") {
			if n := questionID.FindString(part); n != "" {
				id := "Q" + n
				if !seen[id] {
					seen[id] = true
					out = append(out, id)
				}
			}
		}
	}
	return out
}

func sectionNamed(structure *markdown.DocumentStructure, name string) *markdown.Section {
	var matches []*markdown.Section
	for _, section := range structure.SectionsByID {
		if strings.EqualFold(strings.TrimSpace(section.Title), name) {
			matches = append(matches, section)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].StartLine < matches[j].StartLine })
	return matches[0]
}

func analyzeFindingCoverage(content, docPath, againstPath string, findings []AnalysisFinding) []FindingCoverage {
	coverage := make([]FindingCoverage, len(findings))
	for i, f := range findings {
		coverage[i] = FindingCoverage{Finding: f, Status: "uncovered", References: []CoverageReference{}}
	}

	want, _ := filepath.Abs(againstPath)
	docStructure := markdown.ParseDocument(content)
	for _, ref := range markdown.ParseReferences(content) {
		if ref.Path == "" || ref.Line == 0 {
			continue
		}
		resolved, ok := markdown.ResolveReference(filepath.Dir(docPath), ref.Path)
		if !ok {
			continue
		}
		resolved, _ = filepath.Abs(resolved)
		if filepath.Clean(resolved) != filepath.Clean(want) {
			continue
		}
		end := ref.EndLine
		if end == 0 {
			end = ref.Line
		}
		exclusion := isExclusionLine(docStructure, ref.LineNum)
		for i := range coverage {
			f := coverage[i].Finding
			if ref.Line > f.EndLine || end < f.StartLine {
				continue
			}
			coverage[i].References = append(coverage[i].References, CoverageReference{
				Raw:          ref.Raw,
				DocumentLine: ref.LineNum,
				StartLine:    ref.Line,
				EndLine:      end,
				Exclusion:    exclusion,
			})
			if !exclusion {
				coverage[i].Status = "cited"
			} else if coverage[i].Status == "uncovered" {
				coverage[i].Status = "excluded"
			}
		}
	}
	return coverage
}

func isExclusionLine(structure *markdown.DocumentStructure, line int) bool {
	section := structure.SectionsByLine[line]
	for section != nil {
		title := strings.ToLower(section.Title)
		if strings.Contains(title, "not doing") || strings.Contains(title, "non-goal") ||
			strings.Contains(title, "out of scope") || strings.Contains(title, "excluded") {
			return true
		}
		if section.ParentID == "" {
			break
		}
		section = structure.SectionsByID[section.ParentID]
	}
	return false
}
