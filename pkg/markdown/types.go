package markdown

// Section represents a heading in a markdown document with its hierarchy information
type Section struct {
	ID        string     // Unique identifier (e.g., "s1", "s2")
	Level     int        // Heading level (1 for #, 2 for ##, etc.)
	Title     string     // Heading text without the # markers
	StartLine int        // Line number where this section starts (the heading line)
	EndLine   int        // Line number where this section ends (before next same/higher level heading)
	ParentID  string     // ID of parent section (empty for top-level sections)
	Children  []*Section // Child sections (sub-headings)
}

// DocumentStructure represents the hierarchical structure of a markdown document
type DocumentStructure struct {
	Sections       []*Section       // Top-level sections
	SectionsByID   map[string]*Section // Quick lookup by section ID
	SectionsByLine map[int]*Section    // Quick lookup by line number (maps to closest section above)
}

// GetPath returns the hierarchical path of a section (e.g., "Introduction > Overview > Key Points")
func (s *Section) GetPath() string {
	// This will be implemented to traverse up the parent chain
	return ""
}

// GetFullPath returns the hierarchical path including the current section
func (s *Section) GetFullPath(sections map[string]*Section) string {
	if s.ParentID == "" {
		return s.Title
	}

	parent, exists := sections[s.ParentID]
	if !exists {
		return s.Title
	}

	return parent.GetFullPath(sections) + " > " + s.Title
}

// ContainsLine returns true if the given line number is within this section's range
func (s *Section) ContainsLine(line int) bool {
	return line >= s.StartLine && line <= s.EndLine
}

// GetAllDescendants returns all descendant sections (children, grandchildren, etc.)
func (s *Section) GetAllDescendants() []*Section {
	descendants := []*Section{}

	for _, child := range s.Children {
		descendants = append(descendants, child)
		descendants = append(descendants, child.GetAllDescendants()...)
	}

	return descendants
}
