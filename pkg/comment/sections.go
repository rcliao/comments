package comment

import (
	"fmt"

	"github.com/rcliao/comments/pkg/markdown"
)

// ComputeSectionsForComments updates section metadata for all comments in a document
// This should be called when:
// - Loading a v1.0 sidecar file (to compute sections retroactively)
// - Document structure has changed (headings added/removed/renamed)
// - New comments are added without section information
func ComputeSectionsForComments(doc *DocumentWithComments) {
	if doc == nil || len(doc.Threads) == 0 {
		return
	}

	// Parse document structure
	docStructure := markdown.ParseDocument(doc.Content)

	// Update each comment with section information (roots and replies)
	allComments := doc.GetAllComments()
	for _, comment := range allComments {
		if comment.Line <= 0 {
			continue // Skip comments without valid line numbers
		}

		// Find the section for this comment's line
		section, exists := docStructure.SectionsByLine[comment.Line]
		if exists {
			comment.SectionID = section.ID
			comment.SectionPath = section.GetFullPath(docStructure.SectionsByID)
		} else {
			// Line is not within any section (e.g., before first heading)
			comment.SectionID = ""
			comment.SectionPath = ""
		}
	}
}

// ResolveSectionToLines resolves a section path to a line range
// Returns (startLine, endLine, error)
// If includeChildren is true, returns the range including all nested sub-sections
func ResolveSectionToLines(docContent string, sectionPath string, includeChildren bool) (int, int, error) {
	docStructure := markdown.ParseDocument(docContent)

	if includeChildren {
		// Get section and all its descendants
		sections := docStructure.GetAllSectionsInPath(sectionPath)
		if len(sections) == 0 {
			return 0, 0, fmt.Errorf("section not found: %s", sectionPath)
		}

		// Find the overall range (min start line to max end line)
		minStart := sections[0].StartLine
		maxEnd := sections[0].EndLine

		for _, s := range sections {
			if s.StartLine < minStart {
				minStart = s.StartLine
			}
			if s.EndLine > maxEnd {
				maxEnd = s.EndLine
			}
		}

		return minStart, maxEnd, nil
	}

	// Get just this section's range
	return docStructure.GetSectionRange(sectionPath)
}

// GetCommentsInSection returns all comments within a specific section (including nested sections)
func GetCommentsInSection(doc *DocumentWithComments, sectionPath string) []*Comment {
	if doc == nil {
		return []*Comment{}
	}

	// Parse document to get section info
	docStructure := markdown.ParseDocument(doc.Content)
	section := docStructure.FindSection(sectionPath)
	if section == nil {
		return []*Comment{}
	}

	// Get all sections in this path (including descendants)
	sections := docStructure.GetAllSectionsInPath(sectionPath)
	sectionIDs := make(map[string]bool)
	for _, s := range sections {
		sectionIDs[s.ID] = true
	}

	// Filter comments that belong to these sections
	result := []*Comment{}
	allComments := doc.GetAllComments()
	for _, comment := range allComments {
		if sectionIDs[comment.SectionID] {
			result = append(result, comment)
		}
	}

	return result
}

// ListAvailableSections returns all section paths in the document
func ListAvailableSections(docContent string) []string {
	docStructure := markdown.ParseDocument(docContent)
	return docStructure.ListAllPaths()
}

// ValidateSectionPath checks if a section path exists in the document
func ValidateSectionPath(docContent string, sectionPath string) error {
	docStructure := markdown.ParseDocument(docContent)
	section := docStructure.FindSection(sectionPath)
	if section == nil {
		// Provide helpful error message with available sections
		available := docStructure.ListAllPaths()
		if len(available) == 0 {
			return fmt.Errorf("section '%s' not found: document has no headings", sectionPath)
		}
		return fmt.Errorf("section '%s' not found\nAvailable sections:\n  - %s",
			sectionPath, joinPaths(available))
	}
	return nil
}

// joinPaths joins section paths with newline and indent
func joinPaths(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	result := paths[0]
	for i := 1; i < len(paths); i++ {
		result += "\n  - " + paths[i]
	}
	return result
}

// GetSectionForLine returns the section path for a specific line number
func GetSectionForLine(docContent string, line int) string {
	docStructure := markdown.ParseDocument(docContent)
	return docStructure.GetSectionPath(line)
}

// UpdateCommentSection updates section metadata for a single comment
func UpdateCommentSection(comment *Comment, docContent string) {
	if comment == nil || comment.Line <= 0 {
		return
	}

	docStructure := markdown.ParseDocument(docContent)
	section, exists := docStructure.SectionsByLine[comment.Line]
	if exists {
		comment.SectionID = section.ID
		comment.SectionPath = section.GetFullPath(docStructure.SectionsByID)
	} else {
		comment.SectionID = ""
		comment.SectionPath = ""
	}
}
