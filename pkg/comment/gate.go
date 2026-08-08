package comment

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Gate decisions
const (
	DecisionApproved         = "approved"
	DecisionChangesRequested = "changes_requested"
	// DecisionCommented is a reply-pass: the human answered threads and hands
	// the turn back without judging the doc. Never a gate outcome — the gate
	// still derives from blocking threads; agents treat it as "process my
	// replies and keep iterating".
	DecisionCommented = "commented"
)

// GateExitCode is the process exit code when the gate does not pass.
// Follows the revdiff/Plannotator convention: 0 = clean, 10 = review items remain.
const GateExitCode = 10

// GateResult is the outcome of evaluating a single document's review state
type GateResult struct {
	Decision           string     // "approved" or "changes_requested"
	Blocking           []*Comment // Unresolved threads marked blocking
	NonBlocking        []*Comment // Unresolved threads not marked blocking
	PendingSuggestions []*Comment // Suggestions awaiting accept/reject
	LastReview         *ReviewRecord
}

// EvaluateGate computes the review-gate state for a document.
//
// A document passes the gate when no unresolved blocking threads remain.
// In strict mode, any unresolved thread or pending suggestion fails the gate.
// Resolved threads and decided (accepted/rejected) suggestions never block.
func EvaluateGate(doc *DocumentWithComments, strict bool) GateResult {
	result := GateResult{Decision: DecisionApproved}

	for _, thread := range doc.Threads {
		if thread.IsSuggestion {
			if thread.IsPending() && !thread.Resolved {
				result.PendingSuggestions = append(result.PendingSuggestions, thread)
			}
			continue
		}
		if thread.Resolved {
			continue
		}
		if thread.Blocking {
			result.Blocking = append(result.Blocking, thread)
		} else {
			result.NonBlocking = append(result.NonBlocking, thread)
		}
	}

	if len(result.Blocking) > 0 {
		result.Decision = DecisionChangesRequested
	}
	if strict && (len(result.NonBlocking) > 0 || len(result.PendingSuggestions) > 0) {
		result.Decision = DecisionChangesRequested
	}

	if n := len(doc.Reviews); n > 0 {
		result.LastReview = &doc.Reviews[n-1]
	}

	return result
}

// FindGateTargets resolves a file or directory path to markdown files to gate.
// For a directory, only markdown files that have comment sidecars are included.
// A single file is always included (a missing sidecar means an empty, passing doc).
func FindGateTargets(target string) ([]string, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{target}, nil
	}

	var files []string
	err = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// A directory we may not read holds no documents we could review,
			// so skip it rather than failing the whole scan. Ubuntu's
			// /tmp/snap-private-tmp is the case that caught this.
			if os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if SidecarExists(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// AddReviewRecord appends a signoff to the document's review history.
// If decision is empty, it is derived from the current gate state.
func AddReviewRecord(doc *DocumentWithComments, author, decision, note string, strict bool) ReviewRecord {
	if decision == "" {
		decision = EvaluateGate(doc, strict).Decision
	}
	record := ReviewRecord{
		Author:    author,
		Timestamp: time.Now(),
		Decision:  decision,
		Note:      note,
	}
	doc.Reviews = append(doc.Reviews, record)
	return record
}
