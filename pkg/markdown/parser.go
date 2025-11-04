package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Matches ATX-style headings: # Title, ## Title, etc.
	headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
)

// ParseDocument parses markdown content and extracts the document structure
func ParseDocument(content string) *DocumentStructure {
	lines := strings.Split(content, "\n")

	// First pass: identify all headings
	headings := []headingInfo{}
	for i, line := range lines {
		if matches := headingRegex.FindStringSubmatch(line); matches != nil {
			level := len(matches[1]) // Count the # characters
			title := strings.TrimSpace(matches[2])

			headings = append(headings, headingInfo{
				level:     level,
				title:     title,
				lineNumber: i + 1, // Line numbers are 1-indexed
			})
		}
	}

	// Second pass: build sections with hierarchy
	sections := buildSectionHierarchy(headings, len(lines))

	// Third pass: build lookup maps
	sectionsByID := make(map[string]*Section)
	sectionsByLine := make(map[int]*Section)

	populateLookupMaps(sections, sectionsByID, sectionsByLine)

	return &DocumentStructure{
		Sections:       sections,
		SectionsByID:   sectionsByID,
		SectionsByLine: sectionsByLine,
	}
}

// headingInfo is an intermediate structure used during parsing
type headingInfo struct {
	level      int
	title      string
	lineNumber int
}

// buildSectionHierarchy converts a flat list of headings into a hierarchical structure
func buildSectionHierarchy(headings []headingInfo, totalLines int) []*Section {
	if len(headings) == 0 {
		return []*Section{}
	}

	// Convert headings to sections
	sections := make([]*Section, len(headings))
	for i, h := range headings {
		sections[i] = &Section{
			ID:        fmt.Sprintf("s%d", i+1),
			Level:     h.level,
			Title:     h.title,
			StartLine: h.lineNumber,
			EndLine:   totalLines, // Will be adjusted
			Children:  []*Section{},
		}
	}

	// Calculate end lines: each section ends at the line before the next same-or-higher level heading
	for i := 0; i < len(sections); i++ {
		for j := i + 1; j < len(sections); j++ {
			if sections[j].Level <= sections[i].Level {
				sections[i].EndLine = sections[j].StartLine - 1
				break
			}
		}
	}

	// Build hierarchy: attach children to parents
	topLevel := []*Section{}
	parentStack := []*Section{} // Stack to track current parent at each level

	for _, section := range sections {
		// Pop stack until we find the appropriate parent
		for len(parentStack) > 0 && parentStack[len(parentStack)-1].Level >= section.Level {
			parentStack = parentStack[:len(parentStack)-1]
		}

		if len(parentStack) == 0 {
			// This is a top-level section
			topLevel = append(topLevel, section)
		} else {
			// This is a child of the last item in the stack
			parent := parentStack[len(parentStack)-1]
			parent.Children = append(parent.Children, section)
			section.ParentID = parent.ID
		}

		// Push this section onto the stack
		parentStack = append(parentStack, section)
	}

	return topLevel
}

// populateLookupMaps recursively populates the lookup maps
func populateLookupMaps(sections []*Section, byID map[string]*Section, byLine map[int]*Section) {
	for _, section := range sections {
		byID[section.ID] = section

		// Recursively process children FIRST (they take precedence for line mapping)
		populateLookupMaps(section.Children, byID, byLine)

		// Then map parent section lines (only for lines not already mapped by children)
		for line := section.StartLine; line <= section.EndLine; line++ {
			// Only map if not already mapped (child sections take precedence)
			if _, exists := byLine[line]; !exists {
				byLine[line] = section
			}
		}
	}
}

// FindSection finds a section by its hierarchical path (e.g., "Introduction > Overview")
func (d *DocumentStructure) FindSection(path string) *Section {
	parts := strings.Split(path, " > ")
	if len(parts) == 0 {
		return nil
	}

	// Trim whitespace from each part
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Start searching from top-level sections
	return d.findSectionRecursive(parts, d.Sections)
}

// findSectionRecursive recursively searches for a section by path parts
func (d *DocumentStructure) findSectionRecursive(parts []string, sections []*Section) *Section {
	if len(parts) == 0 {
		return nil
	}

	targetTitle := parts[0]

	// Find matching section at this level
	for _, section := range sections {
		if section.Title == targetTitle {
			if len(parts) == 1 {
				// This is the final part, we found it
				return section
			}
			// Continue searching in children
			return d.findSectionRecursive(parts[1:], section.Children)
		}
	}

	return nil
}

// GetSectionPath returns the hierarchical path for a given line number
func (d *DocumentStructure) GetSectionPath(line int) string {
	section, exists := d.SectionsByLine[line]
	if !exists {
		return ""
	}

	return section.GetFullPath(d.SectionsByID)
}

// GetSectionRange returns the start and end line numbers for a section path
func (d *DocumentStructure) GetSectionRange(path string) (startLine, endLine int, err error) {
	section := d.FindSection(path)
	if section == nil {
		return 0, 0, fmt.Errorf("section not found: %s", path)
	}

	return section.StartLine, section.EndLine, nil
}

// GetAllSectionsInPath returns the section and all its descendants for a given path
func (d *DocumentStructure) GetAllSectionsInPath(path string) []*Section {
	section := d.FindSection(path)
	if section == nil {
		return []*Section{}
	}

	result := []*Section{section}
	result = append(result, section.GetAllDescendants()...)
	return result
}

// ListAllPaths returns all possible section paths in the document
func (d *DocumentStructure) ListAllPaths() []string {
	paths := []string{}
	d.collectPaths(d.Sections, &paths)
	return paths
}

// collectPaths recursively collects all section paths
func (d *DocumentStructure) collectPaths(sections []*Section, paths *[]string) {
	for _, section := range sections {
		*paths = append(*paths, section.GetFullPath(d.SectionsByID))
		d.collectPaths(section.Children, paths)
	}
}
