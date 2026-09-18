package comment

import (
	"fmt"
	"strings"
)

// GateThread is a thread as the gate reports it: the canonical view plus the
// document lines around it, so a reader can act without opening the file.
type GateThread struct {
	CommentView
	Context []string `json:"context,omitempty"`
}

// GateFileReport is the gate's verdict on one document: comment state AND
// template structure. Both halves live here so every surface reports the same
// decision — they were once computed separately per adapter and had drifted.
type GateFileReport struct {
	File               string        `json:"file"`
	Decision           string        `json:"decision"`
	Blocking           []GateThread  `json:"blocking"`
	NonBlocking        []GateThread  `json:"non_blocking"`
	PendingSuggestions []GateThread  `json:"pending_suggestions"`
	Template           string        `json:"template,omitempty"`
	Violations         []Violation   `json:"violations,omitempty"`
	LastReview         *ReviewRecord `json:"last_review,omitempty"`
	// StructureUnchecked marks a commented doc with no template recorded: the
	// gate checked comment state only, so a silent pass is not a structural pass.
	StructureUnchecked bool `json:"structure_unchecked,omitempty"`
}

// GateSummary totals a report across its files.
type GateSummary struct {
	Blocking           int `json:"blocking"`
	NonBlocking        int `json:"non_blocking"`
	PendingSuggestions int `json:"pending_suggestions"`
	Violations         int `json:"violations"`
}

// GateReport is the gate's verdict on a file or a directory of documents.
type GateReport struct {
	Decision string           `json:"decision"`
	Strict   bool             `json:"strict"`
	Files    []GateFileReport `json:"files"`
	Summary  GateSummary      `json:"summary"`
}

// BuildGateReport evaluates the review gate for a file or directory. An
// explicit templateName wins; otherwise each document's template resolves from
// frontmatter, legacy sidecar, then bundle. contextSize is the number of
// document lines shown on each side of a thread (0 disables).
func BuildGateReport(target string, strict bool, templateName string, contextSize int) (*GateReport, error) {
	files, err := FindGateTargets(target)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no markdown files with comment sidecars found under %s", target)
	}

	report := &GateReport{Decision: DecisionApproved, Strict: strict}
	for _, file := range files {
		doc, _, err := LoadDocument(file)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", file, err)
		}
		result := EvaluateGate(doc, strict)
		fr := GateFileReport{
			File:               file,
			Decision:           result.Decision,
			Blocking:           gateThreads(result.Blocking, doc.Content, contextSize),
			NonBlocking:        gateThreads(result.NonBlocking, doc.Content, contextSize),
			PendingSuggestions: gateThreads(result.PendingSuggestions, doc.Content, contextSize),
			LastReview:         result.LastReview,
		}

		t, _, err := ResolveTemplateForDocument(file, doc.Content, templateName, doc.Template)
		if err != nil {
			return nil, err
		}
		if t != nil {
			fr.Template = t.Name
			fr.Violations = ValidateManagedDocument(doc.Content, file, t)
			if len(fr.Violations) > 0 {
				fr.Decision = DecisionChangesRequested
			}
		} else if len(doc.Threads) > 0 {
			// A doc with no discoverable template passes the structural half of the
			// gate by default, which reads identically to passing it on merit.
			// Shipped RPI artifacts have gone out hundreds of words over their
			// caps this way, so say it out loud.
			fr.StructureUnchecked = true
		}

		report.Files = append(report.Files, fr)
		report.Summary.Blocking += len(fr.Blocking)
		report.Summary.NonBlocking += len(fr.NonBlocking)
		report.Summary.PendingSuggestions += len(fr.PendingSuggestions)
		report.Summary.Violations += len(fr.Violations)
		if fr.Decision == DecisionChangesRequested {
			report.Decision = DecisionChangesRequested
		}
	}
	return report, nil
}

func gateThreads(comments []*Comment, docContent string, contextSize int) []GateThread {
	out := []GateThread{}
	lines := strings.Split(docContent, "\n")
	for _, c := range comments {
		item := GateThread{CommentView: NewCommentView(c)}
		if contextSize > 0 {
			start := max(1, c.Line-contextSize)
			end := min(len(lines), c.Line+contextSize)
			for i := start; i <= end; i++ {
				marker := "  "
				if i == c.Line {
					marker = "► "
				}
				item.Context = append(item.Context, fmt.Sprintf("%s%d │ %s", marker, i, lines[i-1]))
			}
		}
		out = append(out, item)
	}
	return out
}
