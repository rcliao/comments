package comment

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcliao/comments/pkg/markdown"
)

// ValidationIssue represents a single validation problem
type ValidationIssue struct {
	Severity  string // "error", "warning"
	Message   string
	CommentID string // Optional: which comment has the issue
}

// ValidateAndUpdateCommentStatus validates comments and marks orphaned ones (granular validation)
// This is the new approach that preserves comments across document modifications
// Returns: number of comments orphaned, validation issues
func ValidateAndUpdateCommentStatus(doc *DocumentWithComments) (int, []ValidationIssue) {
	issues := []ValidationIssue{}
	orphanedCount := 0

	// Check 1: Document hash - if mismatch, validate individual comments
	expectedHash := ComputeDocumentHash(doc.Content)
	hashMismatch := doc.DocumentHash != expectedHash

	if hashMismatch {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			Message:  "Document has been modified - validating individual comments",
		})
	}

	lines := strings.Split(doc.Content, "\n")
	lineCount := len(lines)

	// Parse document structure for section validation
	docStructure := markdown.ParseDocument(doc.Content)

	// Validate each comment individually
	movedCount := 0
	allComments := doc.GetAllComments()
	for _, comment := range allComments {
		// Skip already orphaned/completed comments
		if comment.IsOrphaned() || comment.IsCompleted() {
			continue
		}

		orphanReason := ""

		// Check line bounds for anchor-less comments (anchored ones may re-locate)
		if comment.Line < 1 {
			orphanReason = fmt.Sprintf("Invalid line number: %d", comment.Line)
		} else if comment.Line > lineCount && comment.Anchor == nil {
			orphanReason = fmt.Sprintf("Line %d out of bounds (document has %d lines)", comment.Line, lineCount)
		}

		if orphanReason == "" {
			if hashMismatch {
				// Re-anchor cascade: exact position -> exact text -> normalized text
				// -> section fallback -> orphan (v2.1, replaces snap-to-heading)
				moved, reason := ReanchorComment(comment, lines, docStructure)
				orphanReason = reason
				if moved {
					movedCount++
				}
			} else if comment.Anchor == nil && comment.Line <= lineCount {
				// Backfill anchors lazily: document unchanged, so current content
				// is exactly what the comment was created against
				comment.Anchor = CaptureAnchor(doc.Content, comment.Line)
				if comment.AnchorConfidence == "" {
					comment.AnchorConfidence = ConfidenceExact
				}
			}
		}

		// Check suggestion line ranges
		if orphanReason == "" && comment.IsSuggestion {
			if comment.StartLine > lineCount || comment.EndLine > lineCount {
				orphanReason = fmt.Sprintf("Suggestion line range %d-%d out of bounds (document has %d lines)", comment.StartLine, comment.EndLine, lineCount)
			}
		}

		// Mark comment as orphaned if validation failed
		if orphanReason != "" {
			now := time.Now()
			comment.Status = "orphaned"
			comment.OrphanedReason = orphanReason
			comment.OrphanedAt = &now
			if comment.OriginalLine == 0 {
				comment.OriginalLine = comment.Line
			}

			orphanedCount++
			issues = append(issues, ValidationIssue{
				Severity:  "warning",
				Message:   fmt.Sprintf("Comment orphaned: %s", orphanReason),
				CommentID: comment.ID,
			})
		}
	}

	if movedCount > 0 {
		// Moved comments' section metadata may be stale; recompute from new positions.
		// One summary line instead of per-comment spam (dogfood feedback).
		RecomputeAllSections(doc)
		issues = append(issues, ValidationIssue{
			Severity: "info",
			Message:  fmt.Sprintf("%d comment(s) re-anchored after document changes", movedCount),
		})
	}

	return orphanedCount, issues
}

// RecomputeAllSections recomputes section metadata for all comments in the document
// This should be called when the markdown structure changes
func RecomputeAllSections(doc *DocumentWithComments) {
	if doc == nil || len(doc.Threads) == 0 {
		return
	}

	// Parse document structure
	docStructure := markdown.ParseDocument(doc.Content)

	// Update all comments (roots and replies)
	allComments := doc.GetAllComments()
	for _, comment := range allComments {
		if comment.Line <= 0 {
			continue
		}

		// Find section for this line
		section, exists := docStructure.SectionsByLine[comment.Line]
		if exists {
			comment.SectionID = section.ID
			comment.SectionPath = section.GetFullPath(docStructure.SectionsByID)
		} else {
			// Line is not within any section
			comment.SectionID = ""
			comment.SectionPath = ""
		}
	}
}

// FormatValidationIssues formats validation issues as a human-readable string
func FormatValidationIssues(issues []ValidationIssue) string {
	if len(issues) == 0 {
		return "No issues found"
	}

	var result strings.Builder
	fmt.Fprintf(&result, "Found %d validation issue(s):\n\n", len(issues))

	for i, issue := range issues {
		fmt.Fprintf(&result, "%d. [%s] %s", i+1, strings.ToUpper(issue.Severity), issue.Message)
		if issue.CommentID != "" {
			fmt.Fprintf(&result, " (Comment: %s)", issue.CommentID)
		}
		result.WriteString("\n")
	}

	return result.String()
}
