package comment

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/rcliao/comments/pkg/markdown"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*.yaml
var builtinTemplates embed.FS

// ProjectTemplateDir is where project-specific templates live, relative to the
// project root. The project root is discovered by walking up from a start
// directory (the document's directory, or cwd) until a .comments/templates
// directory is found — like git repo discovery — so template resolution does
// not depend on the process cwd (the MCP server may be launched from anywhere).
const ProjectTemplateDir = ".comments/templates"

// findProjectTemplateDir walks up from startDir to the filesystem root looking
// for a .comments/templates directory. Returns "" when none exists.
func findProjectTemplateDir(startDir string) string {
	if startDir == "" {
		startDir = "."
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ProjectTemplateDir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Zone ownership values for template sections
const (
	ZoneHuman = "human" // agents must not resolve threads in this section
	ZoneAgent = "agent"
)

// Template is a document guardrail: it constrains structure at write time,
// powers `comments validate` at gate time, and supplies self-review criteria.
type Template struct {
	Name        string            `yaml:"template"`
	Version     int               `yaml:"version"`
	Description string            `yaml:"description"`
	Doc         TemplateDocRules  `yaml:"doc"`
	Sections    []TemplateSection `yaml:"sections"`
	Markers     TemplateMarkers   `yaml:"markers"`
}

type TemplateDocRules struct {
	MaxWords int `yaml:"max_words"` // 0 = unlimited
	// CheckCitations turns on file:line resolvability checking for documents
	// under this template. It needs the filesystem, so it is applied by callers
	// via ValidateCitations rather than inside the pure ValidateTemplate.
	CheckCitations bool `yaml:"check_citations"`
	// Style bounds prose shape so a reviewer can scan rather than read through.
	Style TemplateStyle `yaml:"style"`
}

type TemplateSection struct {
	Heading        string `yaml:"heading"` // matched against section title or path suffix
	Required       bool   `yaml:"required"`
	MaxWords       int    `yaml:"max_words"`       // 0 = unlimited
	MinSubsections int    `yaml:"min_subsections"` // e.g. Options Considered needs >= 2
	// MaxSubsections and MaxSubsectionWords bound the BODY sections (Findings,
	// Implementation Phases) where review iteration actually accumulates. A
	// section-wide cap alone lets a doc answer feedback by growing one
	// subsection until the whole section bursts; measured across the shipped
	// corpus, every over-cap doc was over because of a handful of oversized
	// subsections, not uniform growth. 0 = unlimited.
	MaxSubsections     int `yaml:"max_subsections"`
	MaxSubsectionWords int `yaml:"max_subsection_words"`
	// EnumeratesQuestions marks the section that decomposes the question into
	// "Q1. ..." clauses; AnswersQuestions marks the section whose subsections
	// claim them back with a [Q1] heading tag. Set both to make coverage a
	// checkable property — see coverage.go for why omission needs its own check.
	EnumeratesQuestions bool     `yaml:"enumerates_questions"`
	AnswersQuestions    bool     `yaml:"answers_questions"`
	Zone                string   `yaml:"zone"`              // "human" or "agent" (default agent)
	ReviewCriteria      []string `yaml:"review_criteria"`   // agent self-review prompts
	CriteriaBlocking    *bool    `yaml:"criteria_blocking"` // nil = default true
}

type TemplateMarkers struct {
	NeedsClarification string `yaml:"needs_clarification"` // default "[NEEDS CLARIFICATION:" (colon distinguishes real markers from prose mentions)
	// Max caps how many markers a doc may carry (0 = unlimited). Above the
	// cap the agent must decide the less consequential ambiguities itself and
	// record them as assumptions, keeping open questions to the few that
	// genuinely need the human (Spec Kit's max-3 convention).
	Max int `yaml:"max"`
}

// Violation is a single structural check failure
type Violation struct {
	Rule    string `json:"rule"` // missing_section, section_order, over_length, subsection_over_length, min_subsections, max_subsections, unresolved_marker, too_many_markers, doc_over_length
	Section string `json:"section,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

func (t *Template) markerPrefix() string {
	return t.MarkerPrefix()
}

// MarkerPrefix is the ambiguity-marker opener this template recognizes, so
// callers rendering the brief show the same string validate enforces.
func (t *Template) MarkerPrefix() string {
	if t.Markers.NeedsClarification != "" {
		return t.Markers.NeedsClarification
	}
	return "[NEEDS CLARIFICATION:"
}

// ListTemplates returns built-in and project template names (project wins on
// conflict). Project templates are discovered by walking up from cwd; use
// ListTemplatesForDoc when a document path is available.
func ListTemplates() (map[string]string, error) {
	return ListTemplatesForDoc("")
}

// ListTemplatesForDoc returns built-in and project template names, discovering
// the project template dir by walking up from the document's directory (or cwd
// when docPath is empty). Project templates win on name conflict.
func ListTemplatesForDoc(docPath string) (map[string]string, error) {
	names := map[string]string{} // name -> source
	entries, err := builtinTemplates.ReadDir("templates")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		names[strings.TrimSuffix(e.Name(), ".yaml")] = "built-in"
	}
	if projDir := findProjectTemplateDir(startDirForDoc(docPath)); projDir != "" {
		if projEntries, err := os.ReadDir(projDir); err == nil {
			for _, e := range projEntries {
				if strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml") {
					base := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".yaml"), ".yml")
					names[base] = "project"
				}
			}
		}
	}
	return names, nil
}

// startDirForDoc maps a document path to the directory template discovery
// starts from: the document's directory, or cwd when no path is given.
func startDirForDoc(docPath string) string {
	if docPath == "" {
		return "."
	}
	return filepath.Dir(docPath)
}

// LoadTemplate loads a template by name, preferring project templates over
// built-ins. The project template dir is discovered by walking up from cwd;
// use LoadTemplateForDoc when a document path is available so resolution works
// regardless of the process cwd.
func LoadTemplate(name string) (*Template, error) {
	return LoadTemplateForDoc(name, "")
}

// LoadTemplateForDoc loads a template by name for a specific document,
// discovering project templates by walking up from the document's directory
// (or cwd when docPath is empty), then falling back to built-ins.
func LoadTemplateForDoc(name, docPath string) (*Template, error) {
	if projDir := findProjectTemplateDir(startDirForDoc(docPath)); projDir != "" {
		for _, candidate := range []string{
			filepath.Join(projDir, name+".yaml"),
			filepath.Join(projDir, name+".yml"),
		} {
			if data, err := os.ReadFile(candidate); err == nil {
				return parseTemplate(data)
			}
		}
	}
	data, err := builtinTemplates.ReadFile("templates/" + name + ".yaml")
	if err != nil {
		available, _ := ListTemplatesForDoc(docPath)
		names := make([]string, 0, len(available))
		for n := range available {
			names = append(names, n)
		}
		return nil, fmt.Errorf("template %q not found (available: %s)", name, strings.Join(names, ", "))
	}
	return parseTemplate(data)
}

func parseTemplate(data []byte) (*Template, error) {
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("invalid template YAML: %w", err)
	}
	if t.Name == "" {
		return nil, fmt.Errorf("template is missing the 'template:' name field")
	}
	return &t, nil
}

// headingMatchesPath reports whether a template heading matches a section path
// (" > "-separated) on whole-segment boundaries. A single-segment heading like
// "Problem" matches only a section titled exactly "Problem" (not "Big Problem");
// a multi-segment heading like "Impl > Details" matches only whole trailing
// segments of the path ("A > Impl > Details" yes, "A > Impl > More Details" no).
func headingMatchesPath(heading, path string) bool {
	want := strings.Split(heading, " > ")
	have := strings.Split(path, " > ")
	if len(want) > len(have) {
		return false
	}
	offset := len(have) - len(want)
	for i, seg := range want {
		if have[offset+i] != seg {
			return false
		}
	}
	return true
}

// findTemplateSection locates the document section matching a template heading.
// Matches by exact title, or by whole-segment path suffix for nested headings
// ("A > B"). All template matching (validation and zone lookup) goes
// through this single helper so the rule cannot drift between sites.
func findTemplateSection(structure *markdown.DocumentStructure, heading string) *markdown.Section {
	var found *markdown.Section
	var walk func(sections []*markdown.Section, parentPath string)
	walk = func(sections []*markdown.Section, parentPath string) {
		for _, s := range sections {
			path := s.Title
			if parentPath != "" {
				path = parentPath + " > " + s.Title
			}
			if found == nil && headingMatchesPath(heading, path) {
				found = s
			}
			walk(s.Children, path)
		}
	}
	walk(structure.Sections, "")
	return found
}

// countSectionWords counts words in a section's own body (excluding child sections)
// countSectionWords counts the section's WHOLE subtree — subsections
// included. It previously stopped at the first child heading, which made
// caps hollow for any section organized into subsections: a 1064-word
// "Proposed Design" counted only its preamble and passed an 800 cap while
// the doc total caught the overage (external field report, reproduced in
// TestSectionCapCountsSubsections).
func countSectionWords(lines []string, section *markdown.Section) int {
	end := min(section.EndLine, len(lines))
	if section.StartLine >= end { // skip heading line itself
		return 0
	}
	return countWords(strings.Join(lines[section.StartLine:end], "\n"))
}

// countWords is the ONE word counter behind every cap: citation tokens are
// exempt (see markdown.StripCitations), so citing evidence never competes with
// content for a section's budget.
//
// A word must carry a letter or digit. Stripping a citation orphans whatever
// punctuation hugged it — `(gate.go:59).` leaves a lone `.` — and counting
// that would hand the budget straight back. The same rule drops standalone
// typography (an em-dash between clauses), which was never a word either.
func countWords(text string) int {
	n := 0
	for _, field := range strings.Fields(markdown.StripCitations(text)) {
		if strings.IndexFunc(field, func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r)
		}) >= 0 {
			n++
		}
	}
	return n
}

// SectionWordCount is one row of the per-section word report: every template
// section's current count against its cap (0 = uncapped), plus the doc total
// as a synthetic row. Reported on success AND failure so trimming is
// informed, not blind (docs/research-agent-surface.md).
type SectionWordCount struct {
	Section string `json:"section"`
	Words   int    `json:"words"`
	Max     int    `json:"max,omitempty"`
}

// SectionWordReport computes the word count of every template section present
// in the document (whole subtree), with the doc total first.
func SectionWordReport(content string, t *Template) []SectionWordCount {
	body := markdown.MaskFrontmatter(content)
	structure := markdown.ParseDocument(body)
	countBody := body
	if t.Name == "plan" {
		countBody = MaskPlanStatusLog(body)
	}
	lines := strings.Split(countBody, "\n")
	report := []SectionWordCount{{Section: "(document)", Words: countWords(countBody), Max: t.Doc.MaxWords}}
	for _, ts := range t.Sections {
		if section := findTemplateSection(structure, ts.Heading); section != nil {
			report = append(report, SectionWordCount{
				Section: ts.Heading,
				Words:   countSectionWords(lines, section),
				Max:     ts.MaxWords,
			})
		}
	}
	return report
}

// ValidateTemplate checks document structure against a template.
// Returns violations; empty means the document conforms.
func ValidateTemplate(content string, t *Template) []Violation {
	violations := []Violation{}
	body := markdown.MaskFrontmatter(content)
	structure := markdown.ParseDocument(body)
	lines := strings.Split(body, "\n")
	countBody := body
	if t.Name == "plan" {
		countBody = MaskPlanStatusLog(body)
	}
	countLines := strings.Split(countBody, "\n")

	// Whole-doc word cap
	if t.Doc.MaxWords > 0 {
		total := countWords(countBody)
		if total > t.Doc.MaxWords {
			violations = append(violations, Violation{
				Rule:    "doc_over_length",
				Message: fmt.Sprintf("document is %d words (max %d) — trim before review", total, t.Doc.MaxWords),
			})
		}
	}

	// Section rules, tracking order of required sections
	lastLine := 0
	for _, ts := range t.Sections {
		section := findTemplateSection(structure, ts.Heading)
		if section == nil {
			if ts.Required {
				violations = append(violations, Violation{
					Rule:    "missing_section",
					Section: ts.Heading,
					Message: fmt.Sprintf("required section %q is missing", ts.Heading),
				})
			}
			continue
		}
		if ts.Required {
			if section.StartLine < lastLine {
				violations = append(violations, Violation{
					Rule:    "section_order",
					Section: ts.Heading,
					Line:    section.StartLine,
					Message: fmt.Sprintf("section %q is out of order (expected after previous required section)", ts.Heading),
				})
			} else {
				lastLine = section.StartLine
			}
		}
		if ts.MaxWords > 0 {
			if words := countSectionWords(countLines, section); words > ts.MaxWords {
				violations = append(violations, Violation{
					Rule:    "over_length",
					Section: ts.Heading,
					Line:    section.StartLine,
					Message: fmt.Sprintf("section %q is %d words (max %d)", ts.Heading, words, ts.MaxWords),
				})
			}
		}
		if ts.MinSubsections > 0 && len(section.Children) < ts.MinSubsections {
			violations = append(violations, Violation{
				Rule:    "min_subsections",
				Section: ts.Heading,
				Line:    section.StartLine,
				Message: fmt.Sprintf("section %q has %d subsection(s), needs at least %d", ts.Heading, len(section.Children), ts.MinSubsections),
			})
		}
		if ts.MaxSubsections > 0 && len(section.Children) > ts.MaxSubsections {
			violations = append(violations, Violation{
				Rule:    "max_subsections",
				Section: ts.Heading,
				Line:    section.StartLine,
				Message: fmt.Sprintf("section %q has %d subsection(s), max %d — merge or drop the weakest, do not split to fit",
					ts.Heading, len(section.Children), ts.MaxSubsections),
			})
		}
		// Named per subsection so trimming is targeted: a bare "section is over"
		// tells the agent to trim somewhere, which is how padding survives.
		if ts.MaxSubsectionWords > 0 {
			for _, child := range section.Children {
				if words := countSectionWords(countLines, child); words > ts.MaxSubsectionWords {
					violations = append(violations, Violation{
						Rule:    "subsection_over_length",
						Section: ts.Heading,
						Line:    child.StartLine,
						Message: fmt.Sprintf("%q is %d words (max %d) — rewrite it tighter or split it, do not trim elsewhere",
							child.Title, words, ts.MaxSubsectionWords),
					})
				}
			}
		}
	}

	// Question coverage: cross-check the decomposed question against the
	// findings claiming to answer it. Only runs when the template opts in by
	// naming both ends.
	var asksSec, answersSec *markdown.Section
	var asksHeading, answersHeading string
	for _, ts := range t.Sections {
		if ts.EnumeratesQuestions {
			if s := findTemplateSection(structure, ts.Heading); s != nil {
				asksSec, asksHeading = s, ts.Heading
			}
		}
		if ts.AnswersQuestions {
			if s := findTemplateSection(structure, ts.Heading); s != nil {
				answersSec, answersHeading = s, ts.Heading
			}
		}
	}
	if asksSec != nil && answersSec != nil {
		violations = append(violations, validateQuestionCoverage(lines, asksSec, answersSec, asksHeading, answersHeading)...)
	}

	violations = append(violations, validateStyle(lines, t.Doc.Style)...)

	// Ambiguity markers: every occurrence is a violation until removed
	marker := t.markerPrefix()
	markerCount := 0
	for i, line := range lines {
		if n := strings.Count(line, marker); n > 0 {
			markerCount += n // cap counts questions, not lines
			violations = append(violations, Violation{
				Rule:    "unresolved_marker",
				Line:    i + 1,
				Message: fmt.Sprintf("line %d: unresolved %s marker", i+1, marker),
			})
		}
	}
	if t.Markers.Max > 0 && markerCount > t.Markers.Max {
		violations = append(violations, Violation{
			Rule: "too_many_markers",
			Message: fmt.Sprintf("%d %s markers exceed the cap of %d — keep only the questions that genuinely need the human; decide the rest and record them as assumptions",
				markerCount, marker, t.Markers.Max),
		})
	}
	if t.Name == "plan" {
		violations = append(violations, ValidatePlanStatus(content)...)
	}

	return violations
}

// ValidateDocument is the path-aware validation entry point shared by every
// CLI and MCP surface. ValidateTemplate deliberately stays pure; citation
// checks need the document path to resolve repository-relative evidence.
// Keeping the append order stable preserves the existing output contract:
// structural violations first, citation violations second.
func ValidateDocument(content, docPath string, t *Template) []Violation {
	violations := ValidateTemplate(content, t)
	if t.Doc.CheckCitations {
		violations = append(violations, ValidateCitations(content, docPath)...)
	}
	return violations
}

// SectionZone returns the template zone ("human"/"agent") for a document line,
// or "" when no template section covers it.
func SectionZone(content string, t *Template, line int) string {
	structure := markdown.ParseDocument(content)
	for _, ts := range t.Sections {
		if ts.Zone == "" {
			continue
		}
		section := findTemplateSection(structure, ts.Heading)
		if section == nil {
			continue
		}
		if line >= section.StartLine && line <= section.EndLine {
			return ts.Zone
		}
	}
	return ""
}
