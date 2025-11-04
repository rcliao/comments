package markdown

import (
	"testing"
)

func TestParseDocumentSimple(t *testing.T) {
	content := `# Introduction

This is the introduction.

# Overview

This is the overview.`

	doc := ParseDocument(content)

	if len(doc.Sections) != 2 {
		t.Fatalf("Expected 2 top-level sections, got %d", len(doc.Sections))
	}

	// Check first section
	if doc.Sections[0].Title != "Introduction" {
		t.Errorf("Expected first section title 'Introduction', got '%s'", doc.Sections[0].Title)
	}
	if doc.Sections[0].Level != 1 {
		t.Errorf("Expected first section level 1, got %d", doc.Sections[0].Level)
	}
	if doc.Sections[0].StartLine != 1 {
		t.Errorf("Expected first section start line 1, got %d", doc.Sections[0].StartLine)
	}
	if doc.Sections[0].EndLine != 4 {
		t.Errorf("Expected first section end line 4, got %d", doc.Sections[0].EndLine)
	}

	// Check second section
	if doc.Sections[1].Title != "Overview" {
		t.Errorf("Expected second section title 'Overview', got '%s'", doc.Sections[1].Title)
	}
	if doc.Sections[1].StartLine != 5 {
		t.Errorf("Expected second section start line 5, got %d", doc.Sections[1].StartLine)
	}
}

func TestParseDocumentNested(t *testing.T) {
	content := `# Introduction

Introduction text.

## Background

Background text.

### History

History text.

## Motivation

Motivation text.

# Conclusion

Conclusion text.`

	doc := ParseDocument(content)

	// Check top-level sections
	if len(doc.Sections) != 2 {
		t.Fatalf("Expected 2 top-level sections, got %d", len(doc.Sections))
	}

	intro := doc.Sections[0]
	if intro.Title != "Introduction" {
		t.Fatalf("Expected first section 'Introduction', got '%s'", intro.Title)
	}

	// Check Introduction has 2 children (Background, Motivation)
	if len(intro.Children) != 2 {
		t.Fatalf("Expected Introduction to have 2 children, got %d", len(intro.Children))
	}

	background := intro.Children[0]
	if background.Title != "Background" {
		t.Errorf("Expected first child 'Background', got '%s'", background.Title)
	}
	if background.ParentID != intro.ID {
		t.Errorf("Expected Background parent to be Introduction")
	}

	// Check Background has 1 child (History)
	if len(background.Children) != 1 {
		t.Fatalf("Expected Background to have 1 child, got %d", len(background.Children))
	}

	history := background.Children[0]
	if history.Title != "History" {
		t.Errorf("Expected nested child 'History', got '%s'", history.Title)
	}
	if history.ParentID != background.ID {
		t.Errorf("Expected History parent to be Background")
	}

	motivation := intro.Children[1]
	if motivation.Title != "Motivation" {
		t.Errorf("Expected second child 'Motivation', got '%s'", motivation.Title)
	}
}

func TestFindSection(t *testing.T) {
	content := `# Introduction

Text.

## Overview

More text.

### Key Points

Details.

# Conclusion`

	doc := ParseDocument(content)

	tests := []struct {
		path          string
		expectedTitle string
		shouldFind    bool
	}{
		{"Introduction", "Introduction", true},
		{"Introduction > Overview", "Overview", true},
		{"Introduction > Overview > Key Points", "Key Points", true},
		{"Conclusion", "Conclusion", true},
		{"NonExistent", "", false},
		{"Introduction > NonExistent", "", false},
	}

	for _, tt := range tests {
		section := doc.FindSection(tt.path)
		if tt.shouldFind {
			if section == nil {
				t.Errorf("FindSection(%q) returned nil, expected section", tt.path)
			} else if section.Title != tt.expectedTitle {
				t.Errorf("FindSection(%q) returned title '%s', expected '%s'", tt.path, section.Title, tt.expectedTitle)
			}
		} else {
			if section != nil {
				t.Errorf("FindSection(%q) returned section, expected nil", tt.path)
			}
		}
	}
}

func TestGetSectionPath(t *testing.T) {
	content := `# Introduction

Line 3 text.

## Overview

Line 7 text.

### Key Points

Line 11 text.

# Conclusion

Line 15 text.`

	doc := ParseDocument(content)

	tests := []struct {
		line         int
		expectedPath string
	}{
		{1, "Introduction"},
		{3, "Introduction"},
		{5, "Introduction > Overview"},
		{7, "Introduction > Overview"},
		{9, "Introduction > Overview > Key Points"},
		{11, "Introduction > Overview > Key Points"},
		{13, "Conclusion"},
		{15, "Conclusion"},
	}

	for _, tt := range tests {
		path := doc.GetSectionPath(tt.line)
		if path != tt.expectedPath {
			t.Errorf("GetSectionPath(%d) = %q, expected %q", tt.line, path, tt.expectedPath)
		}
	}
}

func TestGetSectionRange(t *testing.T) {
	content := `# Introduction

Text.

## Overview

More text.

# Conclusion

Final text.`

	doc := ParseDocument(content)

	tests := []struct {
		path      string
		startLine int
		endLine   int
		shouldErr bool
	}{
		{"Introduction", 1, 8, false},
		{"Introduction > Overview", 5, 8, false},
		{"Conclusion", 9, 11, false},
		{"NonExistent", 0, 0, true},
	}

	for _, tt := range tests {
		start, end, err := doc.GetSectionRange(tt.path)
		if tt.shouldErr {
			if err == nil {
				t.Errorf("GetSectionRange(%q) expected error, got nil", tt.path)
			}
		} else {
			if err != nil {
				t.Errorf("GetSectionRange(%q) unexpected error: %v", tt.path, err)
			}
			if start != tt.startLine {
				t.Errorf("GetSectionRange(%q) start = %d, expected %d", tt.path, start, tt.startLine)
			}
			if end != tt.endLine {
				t.Errorf("GetSectionRange(%q) end = %d, expected %d", tt.path, end, tt.endLine)
			}
		}
	}
}

func TestGetAllSectionsInPath(t *testing.T) {
	content := `# Introduction

Text.

## Overview

More text.

### Details

Detail text.

## Summary

Summary text.

# Conclusion`

	doc := ParseDocument(content)

	// Get all sections in "Introduction" path (should include Overview, Details, Summary)
	sections := doc.GetAllSectionsInPath("Introduction")

	expectedTitles := []string{"Introduction", "Overview", "Details", "Summary"}
	if len(sections) != len(expectedTitles) {
		t.Fatalf("Expected %d sections in 'Introduction' path, got %d", len(expectedTitles), len(sections))
	}

	for i, expected := range expectedTitles {
		if sections[i].Title != expected {
			t.Errorf("Section %d: expected title '%s', got '%s'", i, expected, sections[i].Title)
		}
	}
}

func TestListAllPaths(t *testing.T) {
	content := `# Introduction

## Overview

### Background

# Conclusion`

	doc := ParseDocument(content)

	paths := doc.ListAllPaths()

	expectedPaths := []string{
		"Introduction",
		"Introduction > Overview",
		"Introduction > Overview > Background",
		"Conclusion",
	}

	if len(paths) != len(expectedPaths) {
		t.Fatalf("Expected %d paths, got %d", len(expectedPaths), len(paths))
	}

	for i, expected := range expectedPaths {
		if paths[i] != expected {
			t.Errorf("Path %d: expected '%s', got '%s'", i, expected, paths[i])
		}
	}
}

func TestEmptyDocument(t *testing.T) {
	content := "Just text with no headings."

	doc := ParseDocument(content)

	if len(doc.Sections) != 0 {
		t.Errorf("Expected 0 sections for document without headings, got %d", len(doc.Sections))
	}
}

func TestMultipleLevels(t *testing.T) {
	content := `# Level 1
## Level 2
### Level 3
#### Level 4
##### Level 5
###### Level 6`

	doc := ParseDocument(content)

	if len(doc.Sections) != 1 {
		t.Fatalf("Expected 1 top-level section, got %d", len(doc.Sections))
	}

	// Traverse down the hierarchy
	current := doc.Sections[0]
	for level := 1; level <= 6; level++ {
		if current.Level != level {
			t.Errorf("Expected level %d, got %d", level, current.Level)
		}
		if level < 6 {
			if len(current.Children) != 1 {
				t.Fatalf("Expected 1 child at level %d, got %d", level, len(current.Children))
			}
			current = current.Children[0]
		}
	}
}

func TestSectionEndLines(t *testing.T) {
	content := `Line 1
# Section 1
Line 3
Line 4
## Section 1.1
Line 6
# Section 2
Line 8`

	doc := ParseDocument(content)

	if len(doc.Sections) != 2 {
		t.Fatalf("Expected 2 top-level sections, got %d", len(doc.Sections))
	}

	// Section 1 should end at line 6 (before Section 2 starts at line 7)
	if doc.Sections[0].EndLine != 6 {
		t.Errorf("Section 1 end line: expected 6, got %d", doc.Sections[0].EndLine)
	}

	// Section 1.1 should end at line 6 (before Section 2 starts at line 7)
	if len(doc.Sections[0].Children) != 1 {
		t.Fatalf("Expected Section 1 to have 1 child, got %d", len(doc.Sections[0].Children))
	}
	if doc.Sections[0].Children[0].EndLine != 6 {
		t.Errorf("Section 1.1 end line: expected 6, got %d", doc.Sections[0].Children[0].EndLine)
	}

	// Section 2 should end at line 8 (last line of document)
	if doc.Sections[1].EndLine != 8 {
		t.Errorf("Section 2 end line: expected 8, got %d", doc.Sections[1].EndLine)
	}
}
