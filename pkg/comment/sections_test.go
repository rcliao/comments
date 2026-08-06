package comment

import (
	"strings"
	"testing"
)

const sectionsDoc = `Preamble line.

# Intro

Intro body.

## Background

Background body.

# Impl

Impl body.
`

func TestUpdateCommentSection(t *testing.T) {
	c := &Comment{ID: "c1", Line: 9} // "Background body."
	UpdateCommentSection(c, sectionsDoc)

	if c.SectionPath != "Intro > Background" {
		t.Errorf("SectionPath = %q, want %q", c.SectionPath, "Intro > Background")
	}
	if c.SectionID == "" {
		t.Error("SectionID not set")
	}
	if c.Anchor == nil {
		t.Fatal("anchor not captured")
	}
	if c.Anchor.SelectedText != "Background body." {
		t.Errorf("anchor SelectedText = %q, want %q", c.Anchor.SelectedText, "Background body.")
	}
	if c.AnchorConfidence != ConfidenceExact {
		t.Errorf("AnchorConfidence = %q, want %q", c.AnchorConfidence, ConfidenceExact)
	}
}

func TestUpdateCommentSectionBeforeFirstHeading(t *testing.T) {
	c := &Comment{ID: "c1", Line: 1, SectionID: "stale", SectionPath: "Stale > Path"}
	UpdateCommentSection(c, sectionsDoc)

	if c.SectionID != "" || c.SectionPath != "" {
		t.Errorf("section metadata not cleared: id=%q path=%q", c.SectionID, c.SectionPath)
	}
}

func TestUpdateCommentSectionInvalidLineNoOp(t *testing.T) {
	c := &Comment{ID: "c1", Line: 0, SectionPath: "unchanged"}
	UpdateCommentSection(c, sectionsDoc)
	if c.SectionPath != "unchanged" {
		t.Errorf("comment with line 0 was modified: %q", c.SectionPath)
	}

	UpdateCommentSection(nil, sectionsDoc) // must not panic
}

func TestUpdateCommentSectionKeepsExistingAnchor(t *testing.T) {
	existing := &Anchor{SelectedText: "already captured"}
	c := &Comment{ID: "c1", Line: 5, Anchor: existing}
	UpdateCommentSection(c, sectionsDoc)
	if c.Anchor != existing {
		t.Error("existing anchor was overwritten")
	}
}

func TestComputeSectionsForComments(t *testing.T) {
	doc := &DocumentWithComments{
		Content: sectionsDoc,
		Threads: []*Comment{
			{ID: "c1", Line: 5, Replies: []*Comment{
				{ID: "c2", Line: 13}, // "Impl body." — replies get sections too
			}},
			{ID: "c3", Line: 1, SectionPath: "Stale"}, // before first heading
			{ID: "c4", Line: 0, SectionPath: "Kept"},  // invalid line skipped
		},
	}

	ComputeSectionsForComments(doc)

	if got := doc.Threads[0].SectionPath; got != "Intro" {
		t.Errorf("root SectionPath = %q, want %q", got, "Intro")
	}
	if got := doc.Threads[0].Replies[0].SectionPath; got != "Impl" {
		t.Errorf("reply SectionPath = %q, want %q", got, "Impl")
	}
	if got := doc.Threads[1].SectionPath; got != "" {
		t.Errorf("pre-heading SectionPath = %q, want cleared", got)
	}
	if got := doc.Threads[2].SectionPath; got != "Kept" {
		t.Errorf("invalid-line comment modified: %q", got)
	}
}

func TestGetCommentsInSection(t *testing.T) {
	doc := &DocumentWithComments{
		Content: sectionsDoc,
		Threads: []*Comment{
			{ID: "c1", Line: 5},  // Intro
			{ID: "c2", Line: 9},  // Intro > Background
			{ID: "c3", Line: 13}, // Impl
		},
	}
	ComputeSectionsForComments(doc)

	got := GetCommentsInSection(doc, "Intro")
	if len(got) != 2 {
		t.Fatalf("GetCommentsInSection(Intro) = %d comments, want 2 (own + nested)", len(got))
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids["c1"] || !ids["c2"] {
		t.Errorf("wrong comments in Intro: %v", ids)
	}

	if got := GetCommentsInSection(doc, "Impl"); len(got) != 1 || got[0].ID != "c3" {
		t.Errorf("GetCommentsInSection(Impl) wrong: %v", got)
	}
	if got := GetCommentsInSection(doc, "Nope"); len(got) != 0 {
		t.Errorf("unknown section returned %d comments, want 0", len(got))
	}
}

func TestResolveSectionToLines(t *testing.T) {
	// "Intro > Background" without children: heading at line 7, ends at line 10
	start, end, err := ResolveSectionToLines(sectionsDoc, "Intro > Background", false)
	if err != nil {
		t.Fatalf("ResolveSectionToLines failed: %v", err)
	}
	if start != 7 || end != 10 {
		t.Errorf("Background range = %d-%d, want 7-10", start, end)
	}

	// "Intro" with children spans through Background
	start, end, err = ResolveSectionToLines(sectionsDoc, "Intro", true)
	if err != nil {
		t.Fatalf("ResolveSectionToLines failed: %v", err)
	}
	if start != 3 || end != 10 {
		t.Errorf("Intro range = %d-%d, want 3-10", start, end)
	}

	if _, _, err := ResolveSectionToLines(sectionsDoc, "Nope", true); err == nil {
		t.Error("expected error for unknown section")
	}
}

func TestValidateSectionPath(t *testing.T) {
	if err := ValidateSectionPath(sectionsDoc, "Intro > Background"); err != nil {
		t.Errorf("valid path rejected: %v", err)
	}

	err := ValidateSectionPath(sectionsDoc, "Overvieww")
	if err == nil {
		t.Fatal("expected error for unknown section")
	}
	if !strings.Contains(err.Error(), "Available sections") ||
		!strings.Contains(err.Error(), "Intro > Background") {
		t.Errorf("error should list available sections, got: %v", err)
	}

	err = ValidateSectionPath("no headings here\n", "Intro")
	if err == nil || !strings.Contains(err.Error(), "no headings") {
		t.Errorf("headingless doc error wrong: %v", err)
	}
}
