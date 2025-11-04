package comment

import (
	"testing"
)

func TestRecalculateCommentLines(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", Line: 5, Replies: []*Comment{}},
		{ID: "c2", Line: 10, Replies: []*Comment{}},
		{ID: "c3", Line: 15, Replies: []*Comment{}},
		{ID: "c4", Line: 20, Replies: []*Comment{}},
	}

	// Edit lines 10-12 (delete 3 lines, add 2 lines) - net -1 line
	RecalculateCommentLines(comments, 10, 12, 2)

	// c1 should be unchanged (before edit)
	if comments[0].Line != 5 {
		t.Errorf("c1 line = %d, want 5", comments[0].Line)
	}

	// c2 should be moved to start of edit (within edited range)
	if comments[1].Line != 10 {
		t.Errorf("c2 line = %d, want 10", comments[1].Line)
	}

	// c3 should be shifted by delta (-1) since it's after the edit range
	if comments[2].Line != 14 {
		t.Errorf("c3 line = %d, want 14", comments[2].Line)
	}

	// c4 should be shifted by delta (-1)
	if comments[3].Line != 19 {
		t.Errorf("c4 line = %d, want 19", comments[3].Line)
	}
}

func TestRecalculateCommentLinesInsertion(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", Line: 5, Replies: []*Comment{}},
		{ID: "c2", Line: 10, Replies: []*Comment{}},
	}

	// Insert 3 lines at line 7 (replace 1 line with 3 lines)
	RecalculateCommentLines(comments, 7, 7, 3)

	// c1 before insertion - unchanged
	if comments[0].Line != 5 {
		t.Errorf("c1 line = %d, want 5", comments[0].Line)
	}

	// c2 after insertion - should shift by +2 (net gain of 2 lines)
	if comments[1].Line != 12 {
		t.Errorf("c2 line = %d, want 12", comments[1].Line)
	}
}

func TestRecalculateCommentLinesDeletion(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", Line: 5, Replies: []*Comment{}},
		{ID: "c2", Line: 10, Replies: []*Comment{}},
		{ID: "c3", Line: 15, Replies: []*Comment{}},
	}

	// Delete lines 8-12 (5 lines deleted, 0 added)
	RecalculateCommentLines(comments, 8, 12, 0)

	// c1 before deletion - unchanged
	if comments[0].Line != 5 {
		t.Errorf("c1 line = %d, want 5", comments[0].Line)
	}

	// c2 within deleted range - moved to start
	if comments[1].Line != 8 {
		t.Errorf("c2 line = %d, want 8", comments[1].Line)
	}

	// c3 after deletion - shifted by -5
	if comments[2].Line != 10 {
		t.Errorf("c3 line = %d, want 10", comments[2].Line)
	}
}

func TestRecalculateCommentLinesNestedReplies(t *testing.T) {
	comments := []*Comment{
		{
			ID:   "c1",
			Line: 5,
			Replies: []*Comment{
				{ID: "c2", Line: 5, Replies: []*Comment{}},
			},
		},
		{ID: "c3", Line: 15, Replies: []*Comment{}},
	}

	// Edit lines 10-12
	RecalculateCommentLines(comments, 10, 12, 2)

	// c1 and its reply should be unchanged (before edit)
	if comments[0].Line != 5 {
		t.Errorf("c1 line = %d, want 5", comments[0].Line)
	}
	if comments[0].Replies[0].Line != 5 {
		t.Errorf("c2 line = %d, want 5", comments[0].Replies[0].Line)
	}

	// c3 should be shifted by delta (-1) since it's after the edit range
	if comments[1].Line != 14 {
		t.Errorf("c3 line = %d, want 14", comments[1].Line)
	}
}

func TestSortSuggestionsByLine(t *testing.T) {
	suggestions := []*Comment{
		{ID: "s1", StartLine: 10, EndLine: 10},
		{ID: "s2", StartLine: 5, EndLine: 5},
		{ID: "s3", StartLine: 20, EndLine: 20},
	}

	SortSuggestionsByLine(suggestions)

	// Should be sorted descending (20, 10, 5)
	if suggestions[0].ID != "s3" {
		t.Errorf("First should be s3, got %s", suggestions[0].ID)
	}
	if suggestions[1].ID != "s1" {
		t.Errorf("Second should be s1, got %s", suggestions[1].ID)
	}
	if suggestions[2].ID != "s2" {
		t.Errorf("Third should be s2, got %s", suggestions[2].ID)
	}
}

func TestGetAffectedComments(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", Line: 5, Replies: []*Comment{}},
		{ID: "c2", Line: 10, Replies: []*Comment{}},
		{ID: "c3", Line: 15, Replies: []*Comment{}},
		{ID: "c4", Line: 20, Replies: []*Comment{}},
	}

	// Edit lines 10-15
	affected := GetAffectedComments(comments, 10, 15)

	if len(affected) != 2 {
		t.Errorf("Expected 2 affected comments, got %d", len(affected))
	}

	// Should contain c2 and c3
	hasC2, hasC3 := false, false
	for _, c := range affected {
		if c.ID == "c2" {
			hasC2 = true
		}
		if c.ID == "c3" {
			hasC3 = true
		}
	}

	if !hasC2 {
		t.Error("Expected c2 to be affected")
	}
	if !hasC3 {
		t.Error("Expected c3 to be affected")
	}
}

func TestGetAffectedCommentsWithReplies(t *testing.T) {
	comments := []*Comment{
		{
			ID:   "c1",
			Line: 5,
			Replies: []*Comment{
				{ID: "c2", Line: 10, Replies: []*Comment{}},
			},
		},
		{ID: "c3", Line: 15, Replies: []*Comment{}},
	}

	// Edit lines 10-15
	affected := GetAffectedComments(comments, 10, 15)

	// Should find c2 (nested reply) and c3
	if len(affected) != 2 {
		t.Errorf("Expected 2 affected comments, got %d", len(affected))
	}

	// Should contain c2 and c3
	hasC2, hasC3 := false, false
	for _, c := range affected {
		if c.ID == "c2" {
			hasC2 = true
		}
		if c.ID == "c3" {
			hasC3 = true
		}
	}

	if !hasC2 {
		t.Error("Expected c2 (nested reply) to be affected")
	}
	if !hasC3 {
		t.Error("Expected c3 to be affected")
	}
}

func TestDetectConflictsOverlap(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil, // Pending
			StartLine:    5,
			EndLine:      10,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil, // Pending
			StartLine:    8,
			EndLine:      12,
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	if conflicts[0].Type != ConflictOverlap {
		t.Errorf("Expected ConflictOverlap, got %s", conflicts[0].Type)
	}
}

func TestDetectConflictsNested(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    5,
			EndLine:      15,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    8,
			EndLine:      10,
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	if conflicts[0].Type != ConflictNested {
		t.Errorf("Expected ConflictNested, got %s", conflicts[0].Type)
	}
}

func TestDetectConflictsAdjacent(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    10,
			EndLine:      10,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    11,
			EndLine:      11,
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	if conflicts[0].Type != ConflictAdjacent {
		t.Errorf("Expected ConflictAdjacent, got %s", conflicts[0].Type)
	}
}

func TestDetectConflictsNoConflict(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    5,
			EndLine:      5,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    20,
			EndLine:      20,
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(conflicts))
	}
}

func TestDetectConflictsSkipsNonPending(t *testing.T) {
	accepted := true

	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     &accepted, // Not pending
			StartLine:    10,
			EndLine:      10,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil, // Pending
			StartLine:    10,
			EndLine:      10,
		},
	}

	conflicts := DetectConflicts(suggestions)

	// Should not detect conflict because s1 is not pending
	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts (s1 not pending), got %d", len(conflicts))
	}
}

func TestHasConflicts(t *testing.T) {
	conflicts := []Conflict{
		{Type: ConflictAdjacent}, // Not a serious conflict
		{Type: ConflictOverlap},  // Serious conflict
	}

	if !HasConflicts(conflicts) {
		t.Error("HasConflicts should return true when overlaps exist")
	}

	noConflicts := []Conflict{
		{Type: ConflictAdjacent},
		{Type: ConflictNone},
	}

	if HasConflicts(noConflicts) {
		t.Error("HasConflicts should return false for only adjacent conflicts")
	}
}

func TestFilterNonConflicting(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    10,
			EndLine:      15,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    12,
			EndLine:      14,
		},
		{
			ID:           "s3",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    20,
			EndLine:      20,
		},
	}

	filtered := FilterNonConflicting(suggestions)

	// Should keep s1 and s3, remove s2 (conflicts with s1)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 non-conflicting suggestions, got %d", len(filtered))
	}

	// Check that s1 and s3 are kept
	hasS1, hasS3 := false, false
	for _, s := range filtered {
		if s.ID == "s1" {
			hasS1 = true
		}
		if s.ID == "s3" {
			hasS3 = true
		}
		if s.ID == "s2" {
			t.Error("s2 should have been filtered out")
		}
	}

	if !hasS1 {
		t.Error("s1 should be in filtered list")
	}
	if !hasS3 {
		t.Error("s3 should be in filtered list")
	}
}

func TestFilterNonConflictingNoConflicts(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:           "s1",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    5,
			EndLine:      5,
		},
		{
			ID:           "s2",
			IsSuggestion: true,
			Accepted:     nil,
			StartLine:    10,
			EndLine:      10,
		},
	}

	filtered := FilterNonConflicting(suggestions)

	// Should keep all suggestions
	if len(filtered) != 2 {
		t.Errorf("Expected 2 suggestions (no conflicts), got %d", len(filtered))
	}
}
