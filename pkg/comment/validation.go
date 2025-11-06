package comment

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcliao/comments/pkg/markdown"
)

// ValidationIssue represents a single validation problem
type ValidationIssue struct {
	Severity string // "error", "warning"
	Message  string
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
	allComments := doc.GetAllComments()
	for _, comment := range allComments {
		// Skip already orphaned/completed comments
		if comment.IsOrphaned() || comment.IsCompleted() {
			continue
		}

		orphanReason := ""

		// Check line bounds
		if comment.Line < 1 {
			orphanReason = fmt.Sprintf("Invalid line number: %d", comment.Line)
		} else if comment.Line > lineCount {
			orphanReason = fmt.Sprintf("Line %d out of bounds (document has %d lines)", comment.Line, lineCount)
		}

		// Check section paths (if comment is section-based)
		if orphanReason == "" && comment.SectionPath != "" {
			section := docStructure.FindSection(comment.SectionPath)
			if section == nil {
				orphanReason = fmt.Sprintf("Section '%s' no longer exists", comment.SectionPath)
			} else if section.StartLine != comment.Line {
				// Section exists but moved - update line position
				comment.OriginalLine = comment.Line
				comment.Line = section.StartLine
				issues = append(issues, ValidationIssue{
					Severity:  "info",
					Message:   fmt.Sprintf("Section '%s' moved from line %d to %d", comment.SectionPath, comment.OriginalLine, comment.Line),
					CommentID: comment.ID,
				})
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

	return orphanedCount, issues
}

// ValidateSidecar checks if the sidecar is still valid for the current document
// DEPRECATED: Use ValidateAndUpdateCommentStatus for granular validation
// Returns: isValid, issues, error
func ValidateSidecar(doc *DocumentWithComments) (bool, []ValidationIssue, error) {
	issues := []ValidationIssue{}

	// Check 1: Document hash match
	expectedHash := ComputeDocumentHash(doc.Content)
	if doc.DocumentHash != expectedHash {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Message:  "Document hash mismatch - markdown file has been modified outside tool",
		})
	}

	// Check 2: Line numbers within bounds
	lines := strings.Split(doc.Content, "\n")
	lineCount := len(lines)

	allComments := doc.GetAllComments()
	for _, comment := range allComments {
		if comment.Line > lineCount {
			issues = append(issues, ValidationIssue{
				Severity:  "error",
				Message:   fmt.Sprintf("Comment at line %d out of bounds (document has %d lines)", comment.Line, lineCount),
				CommentID: comment.ID,
			})
		}
		if comment.Line < 1 {
			issues = append(issues, ValidationIssue{
				Severity:  "error",
				Message:   fmt.Sprintf("Comment has invalid line number: %d", comment.Line),
				CommentID: comment.ID,
			})
		}
	}

	// Check 3: Section paths still exist
	docStructure := markdown.ParseDocument(doc.Content)
	for _, comment := range allComments {
		if comment.SectionPath != "" {
			// Check if section still exists
			section := docStructure.FindSection(comment.SectionPath)
			if section == nil {
				issues = append(issues, ValidationIssue{
					Severity:  "warning",
					Message:   fmt.Sprintf("Section path '%s' no longer exists", comment.SectionPath),
					CommentID: comment.ID,
				})
			}
		}
	}

	// Check 4: Suggestion line ranges valid
	for _, comment := range allComments {
		if comment.IsSuggestion {
			if comment.StartLine > lineCount || comment.EndLine > lineCount {
				issues = append(issues, ValidationIssue{
					Severity:  "error",
					Message:   fmt.Sprintf("Suggestion line range %d-%d out of bounds (document has %d lines)", comment.StartLine, comment.EndLine, lineCount),
					CommentID: comment.ID,
				})
			}
		}
	}

	// Determine if valid
	hasErrors := false
	for _, issue := range issues {
		if issue.Severity == "error" {
			hasErrors = true
			break
		}
	}

	return !hasErrors, issues, nil
}

// ValidateAndArchiveIfStale validates the sidecar and archives it if stale
// Returns: isValid, archivedPath (if archived), error
func ValidateAndArchiveIfStale(mdPath string, doc *DocumentWithComments) (bool, string, error) {
	isValid, _, err := ValidateSidecar(doc)
	if err != nil {
		return false, "", fmt.Errorf("validation failed: %w", err)
	}

	if !isValid {
		// Archive the stale sidecar
		if err := ArchiveStaleSidecar(mdPath); err != nil {
			return false, "", fmt.Errorf("failed to archive sidecar: %w", err)
		}

		// Get backup path for reporting
		backupPath := GetSidecarPath(mdPath) + ".backup.*"

		return false, backupPath, nil
	}

	return true, "", nil
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
	result.WriteString(fmt.Sprintf("Found %d validation issue(s):\n\n", len(issues)))

	for i, issue := range issues {
		result.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, strings.ToUpper(issue.Severity), issue.Message))
		if issue.CommentID != "" {
			result.WriteString(fmt.Sprintf(" (Comment: %s)", issue.CommentID))
		}
		result.WriteString("\n")
	}

	return result.String()
}
