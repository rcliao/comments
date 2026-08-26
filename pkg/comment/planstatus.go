package comment

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rcliao/comments/pkg/markdown"
)

const (
	// Status history has its own budget so progress never consumes the plan's
	// intent budget. Twenty entries supports multi-week work without becoming
	// an execution transcript; each entry must stay scannable.
	PlanStatusMaxEntries    = 20
	PlanStatusMaxEntryWords = 60
)

var (
	planStatusEntryRE = regexp.MustCompile(`^-\s+(\d{4}-\d{2}-\d{2})\s+[—-]\s+\*\*(pending|active|blocked|done)\*\*\s*$`)
	planStatusFieldRE = regexp.MustCompile(`^\s{2,}-\s+(Summary|Evidence|Next):\s*(.*)$`)
)

type PlanStatusEntry struct {
	Updated  string `json:"updated"`
	State    string `json:"state"`
	Summary  string `json:"summary,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Next     string `json:"next,omitempty"`
	Line     int    `json:"line"`
}

type PlanPhaseStatus struct {
	Title           string            `json:"title"`
	SectionPath     string            `json:"section_path"`
	StartLine       int               `json:"start_line"`
	EndLine         int               `json:"end_line"`
	State           string            `json:"state"`
	HistoryCount    int               `json:"history_count"`
	Latest          *PlanStatusEntry  `json:"latest,omitempty"`
	Entries         []PlanStatusEntry `json:"entries,omitempty"`
	SuccessCriteria string            `json:"success_criteria,omitempty"`
	Warnings        []string          `json:"warnings"`
	Attention       []CommentView     `json:"attention"`
	statusStart     int
	statusEnd       int
}

type ParsedPlanStatus struct {
	Phases   []PlanPhaseStatus `json:"phases"`
	Warnings []string          `json:"warnings"`
}

type PlanApproval struct {
	Freshness           string        `json:"freshness"`
	Decision            string        `json:"decision,omitempty"`
	FullDocumentChanged bool          `json:"full_document_changed,omitempty"`
	Review              *ReviewRecord `json:"review,omitempty"`
}

type PlanImplementationContext struct {
	OverallStatus string            `json:"overall_status"`
	Approval      PlanApproval      `json:"approval"`
	Phases        []PlanPhaseStatus `json:"phases"`
	Warnings      []string          `json:"warnings"`
}

// PlanApprovalState distinguishes stable-intent drift from ordinary status
// edits. Signoffs written before intent hashes were introduced remain useful,
// but their freshness is explicitly unknown.
func PlanApprovalState(doc *DocumentWithComments) PlanApproval {
	if len(doc.Reviews) == 0 {
		return PlanApproval{Freshness: "missing"}
	}
	review := &doc.Reviews[len(doc.Reviews)-1]
	result := PlanApproval{Freshness: "missing", Decision: review.Decision, Review: review}
	if review.Decision != DecisionApproved {
		return result
	}
	if review.IntentHash == "" {
		result.Freshness = "unknown"
		return result
	}
	if review.IntentHash != PlanIntentHash(doc.Content) {
		result.Freshness = "stale"
	} else {
		result.Freshness = "current"
	}
	result.FullDocumentChanged = review.DocumentHash != "" && review.DocumentHash != ComputeDocumentHash(doc.Content)
	return result
}

func BuildPlanImplementationContext(doc *DocumentWithComments) PlanImplementationContext {
	parsed := ParsePlanStatus(doc.Content)
	result := PlanImplementationContext{
		Approval: PlanApprovalState(doc), Phases: parsed.Phases, Warnings: parsed.Warnings,
	}
	attention := len(parsed.Warnings) > 0
	allDone := len(result.Phases) > 0
	for i := range result.Phases {
		phase := &result.Phases[i]
		if len(phase.Warnings) > 0 || phase.State == "blocked" {
			attention = true
		}
		if phase.State != "done" {
			allDone = false
		}
		for _, thread := range doc.Threads {
			if thread.Resolved || (!thread.Blocking && thread.GetPriority() != PriorityHigh) {
				continue
			}
			if thread.Line >= phase.StartLine && thread.Line <= phase.EndLine {
				phase.Attention = append(phase.Attention, NewCommentView(thread))
				attention = true
			}
		}
	}
	switch result.Approval.Freshness {
	case "missing", "unknown":
		result.OverallStatus = "unapproved"
	case "stale":
		result.OverallStatus = "stale"
	default:
		if attention {
			result.OverallStatus = "attention-needed"
		} else if allDone {
			result.OverallStatus = "complete"
		} else {
			result.OverallStatus = "aligned"
		}
	}
	return result
}

// ParsePlanStatus reads the deliberately small plan-ledger convention. A
// missing Status section is a warning, not a validation failure, so historical
// plans remain valid.
func ParsePlanStatus(content string) ParsedPlanStatus {
	structure := markdown.ParseDocument(content)
	implementation := findSectionTitle(structure.Sections, "Implementation Phases")
	result := ParsedPlanStatus{Phases: []PlanPhaseStatus{}, Warnings: []string{}}
	if implementation == nil {
		result.Warnings = append(result.Warnings, "Implementation Phases section is missing")
		return result
	}

	lines := strings.Split(content, "\n")
	for _, section := range implementation.Children {
		if section.Level != 3 {
			continue
		}
		phase := PlanPhaseStatus{
			Title: section.Title, SectionPath: "Implementation Phases > " + section.Title,
			StartLine: section.StartLine, EndLine: section.EndLine, State: "untracked",
			Entries: []PlanStatusEntry{}, Warnings: []string{}, Attention: []CommentView{},
		}
		var statuses []*markdown.Section
		for _, child := range section.Children {
			if child.Level == 4 && strings.EqualFold(strings.TrimSpace(child.Title), "Status") {
				statuses = append(statuses, child)
			}
		}
		if len(statuses) == 0 {
			phase.Warnings = append(phase.Warnings, "Status section is missing")
		} else {
			if len(statuses) > 1 {
				phase.Warnings = append(phase.Warnings, "multiple Status sections found")
			}
			status := statuses[0]
			phase.statusStart = status.StartLine
			phase.statusEnd = statusLogEnd(lines, status)
			phase.Entries, phase.Warnings = parsePlanStatusEntries(lines, phase.statusStart, phase.statusEnd, phase.Warnings)
			phase.HistoryCount = len(phase.Entries)
			if len(phase.Entries) > 0 {
				phase.Latest = &phase.Entries[len(phase.Entries)-1]
				phase.State = phase.Latest.State
			}
		}
		phase.SuccessCriteria = planSuccessCriteria(lines, section, phase.statusStart, phase.statusEnd)
		result.Phases = append(result.Phases, phase)
	}
	if len(result.Phases) == 0 {
		result.Warnings = append(result.Warnings, "Implementation Phases has no H3 phases")
	}
	return result
}

func findSectionTitle(sections []*markdown.Section, title string) *markdown.Section {
	for _, section := range sections {
		if section.Title == title {
			return section
		}
		if found := findSectionTitle(section.Children, title); found != nil {
			return found
		}
	}
	return nil
}

func statusLogEnd(lines []string, status *markdown.Section) int {
	end := min(status.EndLine, len(lines))
	for line := status.StartLine + 1; line <= end; line++ {
		if strings.EqualFold(strings.TrimSpace(lines[line-1]), "**Success Criteria**") {
			return line - 1
		}
	}
	return end
}

func parsePlanStatusEntries(lines []string, start, end int, warnings []string) ([]PlanStatusEntry, []string) {
	entries := []PlanStatusEntry{}
	var current *PlanStatusEntry
	inFence := false
	for line := start + 1; line <= end && line <= len(lines); line++ {
		raw := lines[line-1]
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || trimmed == "" {
			continue
		}
		if match := planStatusEntryRE.FindStringSubmatch(raw); match != nil {
			if _, err := time.Parse("2006-01-02", match[1]); err != nil {
				warnings = append(warnings, fmt.Sprintf("line %d has an invalid status date", line))
			}
			entries = append(entries, PlanStatusEntry{Updated: match[1], State: match[2], Line: line})
			current = &entries[len(entries)-1]
			continue
		}
		if match := planStatusFieldRE.FindStringSubmatch(raw); match != nil && current != nil {
			switch match[1] {
			case "Summary":
				current.Summary = strings.TrimSpace(match[2])
			case "Evidence":
				current.Evidence = strings.TrimSpace(match[2])
			case "Next":
				current.Next = strings.TrimSpace(match[2])
			}
			continue
		}
		warnings = append(warnings, fmt.Sprintf("line %d is not a valid status entry or field", line))
	}
	for _, entry := range entries {
		if entry.Summary == "" || entry.Evidence == "" || entry.Next == "" {
			warnings = append(warnings, fmt.Sprintf("line %d status entry requires Summary, Evidence, and Next", entry.Line))
		}
		if entry.State == "done" && (entry.Evidence == "" || entry.Evidence == "—" || entry.Evidence == "-") {
			warnings = append(warnings, fmt.Sprintf("line %d done status requires evidence", entry.Line))
		}
	}
	return entries, warnings
}

func planSuccessCriteria(lines []string, phase *markdown.Section, statusStart, statusEnd int) string {
	start := 0
	for line := phase.StartLine + 1; line <= phase.EndLine && line <= len(lines); line++ {
		if strings.EqualFold(strings.TrimSpace(lines[line-1]), "**Success Criteria**") {
			start = line + 1
			break
		}
	}
	if start == 0 {
		return ""
	}
	var collected []string
	for line := start; line <= phase.EndLine && line <= len(lines); line++ {
		if statusStart > 0 && line >= statusStart && line <= statusEnd {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(lines[line-1]), "#### ") {
			break
		}
		collected = append(collected, lines[line-1])
	}
	return strings.TrimSpace(strings.Join(collected, "\n"))
}

// MaskPlanStatusLog preserves line numbers while removing only status entries.
// The Status heading and all stable plan intent, including success criteria,
// remain visible to normal validation and intent hashing.
func MaskPlanStatusLog(content string) string {
	lines := strings.Split(content, "\n")
	parsed := ParsePlanStatus(content)
	for _, phase := range parsed.Phases {
		for line := phase.statusStart + 1; phase.statusStart > 0 && line <= phase.statusEnd && line <= len(lines); line++ {
			lines[line-1] = ""
		}
	}
	return strings.Join(lines, "\n")
}

func PlanIntentHash(content string) string {
	return ComputeDocumentHash(MaskPlanStatusLog(content))
}

func ValidatePlanStatus(content string) []Violation {
	parsed := ParsePlanStatus(content)
	var violations []Violation
	for _, phase := range parsed.Phases {
		// Missing status remains warning-only; any present malformed log fails.
		if phase.statusStart > 0 {
			for _, warning := range phase.Warnings {
				violations = append(violations, Violation{Rule: "invalid_status_log", Section: phase.Title, Line: phase.statusStart, Message: warning})
			}
		}
		if len(phase.Entries) > PlanStatusMaxEntries {
			violations = append(violations, Violation{Rule: "status_entry_cap", Section: phase.Title, Line: phase.statusStart,
				Message: fmt.Sprintf("phase has %d status entries (max %d) — consolidate older execution detail", len(phase.Entries), PlanStatusMaxEntries)})
		}
		for _, entry := range phase.Entries {
			words := countWords(strings.Join([]string{entry.Summary, entry.Evidence, entry.Next}, " "))
			if words > PlanStatusMaxEntryWords {
				violations = append(violations, Violation{Rule: "status_entry_over_length", Section: phase.Title, Line: entry.Line,
					Message: fmt.Sprintf("status entry is %d words (max %d)", words, PlanStatusMaxEntryWords)})
			}
		}
	}
	return violations
}
