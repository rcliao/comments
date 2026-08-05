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
