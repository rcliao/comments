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

func TestRecalculateSuggestionRangeShiftOnGrow(t *testing.T) {
	comments := []*Comment{
		{ID: "s1", Line: 3, IsSuggestion: true, StartLine: 3, EndLine: 4, Replies: []*Comment{}},
		{ID: "s2", Line: 10, IsSuggestion: true, StartLine: 10, EndLine: 12, Replies: []*Comment{}},
	}

	// Accepting s1 replaced lines 3-4 with 4 lines (delta +2)
	RecalculateCommentLines(comments, 3, 4, 4)

	// s1 (the just-applied range) is untouched
	if comments[0].StartLine != 3 || comments[0].EndLine != 4 {
		t.Errorf("s1 range = %d-%d, want 3-4", comments[0].StartLine, comments[0].EndLine)
	}

	// s2 is entirely below the edit - both bounds shift by +2
	if comments[1].StartLine != 12 || comments[1].EndLine != 14 {
		t.Errorf("s2 range = %d-%d, want 12-14", comments[1].StartLine, comments[1].EndLine)
	}
	if comments[1].Line != 12 {
		t.Errorf("s2 line = %d, want 12", comments[1].Line)
	}
}

func TestRecalculateSuggestionRangeShiftOnShrink(t *testing.T) {
	comments := []*Comment{
		{ID: "s1", Line: 10, IsSuggestion: true, StartLine: 10, EndLine: 11, Replies: []*Comment{}},
	}

	// An edit above replaced lines 5-8 with 1 line (delta -3)
	RecalculateCommentLines(comments, 5, 8, 1)

	if comments[0].StartLine != 7 || comments[0].EndLine != 8 {
		t.Errorf("s1 range = %d-%d, want 7-8", comments[0].StartLine, comments[0].EndLine)
	}
	if comments[0].Line != 7 {
		t.Errorf("s1 line = %d, want 7", comments[0].Line)
	}
}

func TestRecalculateSuggestionRangeOverlapUntouched(t *testing.T) {
	comments := []*Comment{
		// Overlaps the edited range 5-8 (starts inside it)
		{ID: "s1", Line: 7, IsSuggestion: true, StartLine: 7, EndLine: 10, Replies: []*Comment{}},
		// Starts exactly at the edit's last line - still overlapping
		{ID: "s2", Line: 8, IsSuggestion: true, StartLine: 8, EndLine: 9, Replies: []*Comment{}},
		// Entirely above the edit
		{ID: "s3", Line: 2, IsSuggestion: true, StartLine: 2, EndLine: 3, Replies: []*Comment{}},
	}

	RecalculateCommentLines(comments, 5, 8, 2)

	// Overlapping ranges are left stale on purpose; OriginalText
	// verification in ApplySuggestion guards against misapplication.
	if comments[0].StartLine != 7 || comments[0].EndLine != 10 {
		t.Errorf("s1 range = %d-%d, want 7-10 (untouched)", comments[0].StartLine, comments[0].EndLine)
	}
	if comments[1].StartLine != 8 || comments[1].EndLine != 9 {
		t.Errorf("s2 range = %d-%d, want 8-9 (untouched)", comments[1].StartLine, comments[1].EndLine)
	}
	// Above the edit: untouched
	if comments[2].StartLine != 2 || comments[2].EndLine != 3 {
		t.Errorf("s3 range = %d-%d, want 2-3 (untouched)", comments[2].StartLine, comments[2].EndLine)
	}
}

func TestRecalculateCommentExactlyAtEditEndLine(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", Line: 8, Replies: []*Comment{}},
	}

	// Comment sits exactly on the last edited line - snaps to edit start
	RecalculateCommentLines(comments, 5, 8, 2)

	if comments[0].Line != 5 {
		t.Errorf("c1 line = %d, want 5 (snapped to edit start)", comments[0].Line)
	}
}

func TestRecalculateZeroDelta(t *testing.T) {
	comments := []*Comment{
		{ID: "c1", Line: 8, Replies: []*Comment{}},
		{ID: "s1", Line: 9, IsSuggestion: true, StartLine: 9, EndLine: 10, Replies: []*Comment{}},
	}

	// Replace lines 5-6 with 2 lines: delta is zero, nothing below moves
	RecalculateCommentLines(comments, 5, 6, 2)

	if comments[0].Line != 8 {
		t.Errorf("c1 line = %d, want 8", comments[0].Line)
	}
	if comments[1].StartLine != 9 || comments[1].EndLine != 10 {
		t.Errorf("s1 range = %d-%d, want 9-10", comments[1].StartLine, comments[1].EndLine)
	}
}

func TestRecalculateNestedRepliesBelowEditShift(t *testing.T) {
	comments := []*Comment{
		{
			ID:   "c1",
			Line: 10,
			Replies: []*Comment{
				{
					ID:   "c2",
					Line: 10,
					Replies: []*Comment{
						{ID: "c3", Line: 10, Replies: []*Comment{}},
					},
				},
			},
		},
	}

	// Edit above (lines 2-3 replaced with 5 lines, delta +3)
	RecalculateCommentLines(comments, 2, 3, 5)

	if comments[0].Line != 13 {
		t.Errorf("c1 line = %d, want 13", comments[0].Line)
	}
	if comments[0].Replies[0].Line != 13 {
		t.Errorf("c2 line = %d, want 13", comments[0].Replies[0].Line)
	}
	if comments[0].Replies[0].Replies[0].Line != 13 {
		t.Errorf("c3 line = %d, want 13", comments[0].Replies[0].Replies[0].Line)
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
