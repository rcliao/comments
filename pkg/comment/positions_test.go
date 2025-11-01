package comment

import (
	"testing"
)

func TestCalculateByteOffset(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	tests := []struct {
		line     int
		column   int
		expected int
	}{
		{1, 0, 0},                   // Start of line 1
		{1, 4, 4},                   // Middle of line 1
		{2, 0, 7},                   // Start of line 2 (6 chars + 1 newline)
		{2, 4, 11},                  // Middle of line 2
		{3, 0, 14},                  // Start of line 3
		{3, 6, 20},                  // End of line 3
		{0, 0, 0},                   // Before start
		{10, 0, len(content)},       // After end
	}

	for _, tt := range tests {
		offset := CalculateByteOffset(content, tt.line, tt.column)
		if offset != tt.expected {
			t.Errorf("CalculateByteOffset(%d, %d) = %d, want %d", tt.line, tt.column, offset, tt.expected)
		}
	}
}

func TestRecalculatePositionsAfterEdit(t *testing.T) {
	positions := map[string]Position{
		"c1": {Line: 1, Column: 0},
		"c2": {Line: 5, Column: 0},
		"c3": {Line: 10, Column: 0},
		"c4": {Line: 15, Column: 0},
	}

	// Edit lines 10-12 (delete 3 lines, add 2 lines) - net -1 line
	RecalculatePositionsAfterEdit(10, 12, 2, positions)

	// c1 should be unchanged (before edit)
	if positions["c1"].Line != 1 {
		t.Errorf("c1 line = %d, want 1", positions["c1"].Line)
	}

	// c2 should be unchanged (before edit)
	if positions["c2"].Line != 5 {
		t.Errorf("c2 line = %d, want 5", positions["c2"].Line)
	}

	// c3 should be moved to start of edit (within edited range)
	if positions["c3"].Line != 10 {
		t.Errorf("c3 line = %d, want 10", positions["c3"].Line)
	}

	// c4 should be shifted by delta (-1)
	if positions["c4"].Line != 14 {
		t.Errorf("c4 line = %d, want 14", positions["c4"].Line)
	}
}

func TestRecalculatePositionsAfterInsertion(t *testing.T) {
	positions := map[string]Position{
		"c1": {Line: 5, Column: 0},
		"c2": {Line: 10, Column: 0},
	}

	// Insert 3 lines at line 7 (replace line 7 with 3 lines)
	RecalculatePositionsAfterEdit(7, 7, 3, positions)

	// c1 before insertion - unchanged
	if positions["c1"].Line != 5 {
		t.Errorf("c1 line = %d, want 5", positions["c1"].Line)
	}

	// c2 after insertion - should shift by +2 (net gain of 2 lines)
	if positions["c2"].Line != 12 {
		t.Errorf("c2 line = %d, want 12 (got shift of %d)", positions["c2"].Line, positions["c2"].Line-10)
	}
}

func TestRecalculatePositionsAfterDeletion(t *testing.T) {
	positions := map[string]Position{
		"c1": {Line: 5, Column: 0},
		"c2": {Line: 10, Column: 0},
		"c3": {Line: 15, Column: 0},
	}

	// Delete lines 8-12 (5 lines deleted, 0 added)
	RecalculatePositionsAfterEdit(8, 12, 0, positions)

	// c1 before deletion - unchanged
	if positions["c1"].Line != 5 {
		t.Errorf("c1 line = %d, want 5", positions["c1"].Line)
	}

	// c2 within deleted range - moved to start
	if positions["c2"].Line != 8 {
		t.Errorf("c2 line = %d, want 8", positions["c2"].Line)
	}

	// c3 after deletion - shifted by -5
	if positions["c3"].Line != 10 {
		t.Errorf("c3 line = %d, want 10", positions["c3"].Line)
	}
}

func TestRecalculatePositions(t *testing.T) {
	oldContent := `Line 1
Line 2
Line 3
Line 4
Line 5`

	newContent := `Line 1
Line 2 Modified
Line 3
New Line 4
Line 5`

	positions := map[string]Position{
		"c1": {Line: 1, Column: 0, ByteOffset: 0},
		"c2": {Line: 2, Column: 0, ByteOffset: 7},
		"c3": {Line: 5, Column: 0, ByteOffset: 28},
	}

	RecalculatePositions(oldContent, newContent, positions)

	// c1 on line 1 (unchanged) - should remain at line 1
	if positions["c1"].Line != 1 {
		t.Errorf("c1 line = %d, want 1", positions["c1"].Line)
	}

	// c2 on line 2 (modified) - should still be at line 2
	if positions["c2"].Line != 2 {
		t.Errorf("c2 line = %d, want 2", positions["c2"].Line)
	}

	// c3 on line 5 (unchanged) - should remain at line 5
	if positions["c3"].Line != 5 {
		t.Errorf("c3 line = %d, want 5", positions["c3"].Line)
	}

	// Byte offsets should be recalculated
	if positions["c1"].ByteOffset != 0 {
		t.Errorf("c1 byte offset = %d, want 0", positions["c1"].ByteOffset)
	}
}

func TestRecalculatePositionsWithLineInsertion(t *testing.T) {
	oldContent := `Line 1
Line 3
Line 4`

	newContent := `Line 1
Line 2
Line 3
Line 4`

	positions := map[string]Position{
		"c1": {Line: 1, Column: 0},
		"c3": {Line: 2, Column: 0}, // Was on "Line 3" at line 2
		"c4": {Line: 3, Column: 0}, // Was on "Line 4" at line 3
	}

	RecalculatePositions(oldContent, newContent, positions)

	// c1 unchanged
	if positions["c1"].Line != 1 {
		t.Errorf("c1 line = %d, want 1", positions["c1"].Line)
	}

	// c3 should move to line 3 (after inserted line)
	if positions["c3"].Line != 3 {
		t.Errorf("c3 line = %d, want 3", positions["c3"].Line)
	}

	// c4 should move to line 4
	if positions["c4"].Line != 4 {
		t.Errorf("c4 line = %d, want 4", positions["c4"].Line)
	}
}

func TestRecalculatePositionsWithLineDeletion(t *testing.T) {
	oldContent := `Line 1
Line 2
Line 3
Line 4`

	newContent := `Line 1
Line 3
Line 4`

	positions := map[string]Position{
		"c1": {Line: 1, Column: 0},
		"c2": {Line: 2, Column: 0}, // This line will be deleted
		"c3": {Line: 3, Column: 0},
		"c4": {Line: 4, Column: 0},
	}

	RecalculatePositions(oldContent, newContent, positions)

	// c1 unchanged
	if positions["c1"].Line != 1 {
		t.Errorf("c1 line = %d, want 1", positions["c1"].Line)
	}

	// c2 on deleted line - should map to nearest line
	// Should map to line 2 (the line after line 1, which is now "Line 3")
	if positions["c2"].Line < 1 || positions["c2"].Line > 3 {
		t.Errorf("c2 line = %d, want between 1-3", positions["c2"].Line)
	}

	// c3 should move to line 2 (shifted up)
	if positions["c3"].Line != 2 {
		t.Errorf("c3 line = %d, want 2", positions["c3"].Line)
	}

	// c4 should move to line 3
	if positions["c4"].Line != 3 {
		t.Errorf("c4 line = %d, want 3", positions["c4"].Line)
	}
}

func TestUpdatePositionsByteOffsets(t *testing.T) {
	content := `Line 1
Line 2
Line 3`

	positions := map[string]Position{
		"c1": {Line: 1, Column: 0, ByteOffset: 999}, // Wrong offset
		"c2": {Line: 2, Column: 4, ByteOffset: 999}, // Wrong offset
		"c3": {Line: 3, Column: 0, ByteOffset: 999}, // Wrong offset
	}

	UpdatePositionsByteOffsets(content, positions)

	// Check that byte offsets were recalculated correctly
	if positions["c1"].ByteOffset != 0 {
		t.Errorf("c1 byte offset = %d, want 0", positions["c1"].ByteOffset)
	}

	expectedC2 := 7 + 4 // Line 2 starts at 7, plus column 4
	if positions["c2"].ByteOffset != expectedC2 {
		t.Errorf("c2 byte offset = %d, want %d", positions["c2"].ByteOffset, expectedC2)
	}

	expectedC3 := 14 // Line 3 starts at 14
	if positions["c3"].ByteOffset != expectedC3 {
		t.Errorf("c3 byte offset = %d, want %d", positions["c3"].ByteOffset, expectedC3)
	}
}

func TestGetAffectedComments(t *testing.T) {
	positions := map[string]Position{
		"c1": {Line: 5},
		"c2": {Line: 10},
		"c3": {Line: 15},
		"c4": {Line: 20},
	}

	// Edit lines 10-15
	affected := GetAffectedComments(positions, 10, 15)

	if len(affected) != 2 {
		t.Errorf("Expected 2 affected comments, got %d", len(affected))
	}

	// Should contain c2 and c3
	hasC2, hasC3 := false, false
	for _, id := range affected {
		if id == "c2" {
			hasC2 = true
		}
		if id == "c3" {
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

func TestSortSuggestionsByPosition(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:   "s1",
			Line: 10,
			Selection: &Selection{
				StartLine: 10,
			},
		},
		{
			ID:   "s2",
			Line: 5,
			Selection: &Selection{
				StartLine: 5,
			},
		},
		{
			ID:   "s3",
			Line: 20,
			Selection: &Selection{
				StartLine: 20,
			},
		},
	}

	SortSuggestionsByPosition(suggestions)

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

func TestSortSuggestionsByPositionSameLine(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:   "s1",
			Line: 10,
			Selection: &Selection{
				StartLine:  10,
				ByteOffset: 20,
			},
		},
		{
			ID:   "s2",
			Line: 10,
			Selection: &Selection{
				StartLine:  10,
				ByteOffset: 5,
			},
		},
		{
			ID:   "s3",
			Line: 10,
			Selection: &Selection{
				StartLine:  10,
				ByteOffset: 50,
			},
		},
	}

	SortSuggestionsByPosition(suggestions)

	// Same line - should be sorted by byte offset descending (50, 20, 5)
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

func TestBuildLineMapping(t *testing.T) {
	oldLines := []string{"Line 1", "Line 2", "Line 3", "Line 4"}
	newLines := []string{"Line 1", "Line 2 Modified", "Line 3", "Line 4"}

	mapping := buildLineMapping(oldLines, newLines)

	// Lines 1, 3, 4 should map to themselves
	if mapping[1] != 1 {
		t.Errorf("Line 1 mapping = %d, want 1", mapping[1])
	}
	if mapping[3] != 3 {
		t.Errorf("Line 3 mapping = %d, want 3", mapping[3])
	}
	if mapping[4] != 4 {
		t.Errorf("Line 4 mapping = %d, want 4", mapping[4])
	}

	// Line 2 was modified, might not have a mapping
	// (depends on exact algorithm behavior)
}

func TestFindNearestLine(t *testing.T) {
	mapping := map[int]int{
		1: 1,
		2: 2,
		4: 4,
		5: 5,
	}

	// Line 3 was deleted - should map to nearest line
	nearest := findNearestLine(3, mapping)

	// Should map to line 3 (one after line 2's mapping)
	if nearest != 3 {
		t.Errorf("Nearest line for deleted line 3 = %d, want 3", nearest)
	}
}

func TestDetectConflictsOverlap(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 5,
				EndLine:   10,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 8,
				EndLine:   12,
			},
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
			ID:              "s1",
			SuggestionType:  SuggestionMultiLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 5,
				EndLine:   15,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 8,
				EndLine:   10,
			},
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

func TestDetectConflictsSameLine(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 10,
				EndLine:   10,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 10,
				EndLine:   10,
			},
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	if conflicts[0].Type != ConflictSameLine {
		t.Errorf("Expected ConflictSameLine, got %s", conflicts[0].Type)
	}
}

func TestDetectConflictsAdjacent(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 10,
				EndLine:   10,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 11,
				EndLine:   11,
			},
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

func TestDetectConflictsCharRange(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionCharRange,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				ByteOffset: 10,
				Length:     20,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionCharRange,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				ByteOffset: 25,
				Length:     10,
			},
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 1 {
		t.Fatalf("Expected 1 conflict, got %d", len(conflicts))
	}

	if conflicts[0].Type != ConflictOverlap {
		t.Errorf("Expected ConflictOverlap for char ranges, got %s", conflicts[0].Type)
	}
}

func TestDetectConflictsNoConflict(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 5,
				EndLine:   5,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 20,
				EndLine:   20,
			},
		},
	}

	conflicts := DetectConflicts(suggestions)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(conflicts))
	}
}

func TestDetectConflictsSkipsNonPending(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptanceAccepted, // Not pending
			Selection: &Selection{
				StartLine: 10,
				EndLine:   10,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 10,
				EndLine:   10,
			},
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
		{Type: ConflictAdjacent}, // Not a real conflict
		{Type: ConflictOverlap},  // Real conflict
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

func TestGetConflictingSuggestions(t *testing.T) {
	s1 := &Comment{ID: "s1"}
	s2 := &Comment{ID: "s2"}
	s3 := &Comment{ID: "s3"}

	conflicts := []Conflict{
		{
			Type:        ConflictOverlap,
			Suggestion1: s1,
			Suggestion2: s2,
		},
		{
			Type:        ConflictAdjacent, // Not counted
			Suggestion1: s2,
			Suggestion2: s3,
		},
	}

	conflicting := GetConflictingSuggestions(conflicts)

	if len(conflicting) != 2 {
		t.Errorf("Expected 2 conflicting suggestions, got %d", len(conflicting))
	}

	if !conflicting["s1"] {
		t.Error("s1 should be conflicting")
	}
	if !conflicting["s2"] {
		t.Error("s2 should be conflicting")
	}
	if conflicting["s3"] {
		t.Error("s3 should not be conflicting (adjacent only)")
	}
}

func TestFilterNonConflicting(t *testing.T) {
	suggestions := []*Comment{
		{
			ID:              "s1",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 10,
				EndLine:   15,
			},
		},
		{
			ID:              "s2",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 12,
				EndLine:   14,
			},
		},
		{
			ID:              "s3",
			SuggestionType:  SuggestionLine,
			AcceptanceState: AcceptancePending,
			Selection: &Selection{
				StartLine: 20,
				EndLine:   20,
			},
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
