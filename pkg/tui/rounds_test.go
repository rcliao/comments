package tui

// Tests for Phase A thread tracking (docs/design-markdown-render.md):
// round partitioning as pure functions, NEW badges in sidebar and line
// summaries, round separators in the expanded thread timeline, and the
//  n/N  new-activity motions.

import (
	"strings"
	"testing"
	"time"

	"github.com/rcliao/comments/pkg/comment"
)

// rt builds a deterministic timestamp for round tests (minute-granular)
func rt(min int) time.Time {
	return time.Date(2026, 8, 5, 10, min, 0, 0, time.UTC)
}

func signoffs(mins ...int) []comment.ReviewRecord {
	reviews := []comment.ReviewRecord{}
	for _, m := range mins {
		reviews = append(reviews, comment.ReviewRecord{
			Author: "rcliao", Timestamp: rt(m), Decision: comment.DecisionApproved,
		})
	}
	return reviews
}

// --- Pure round partitioning -------------------------------------------------

func TestLastSignoffTime(t *testing.T) {
	if got := lastSignoffTime(nil); !got.IsZero() {
		t.Errorf("no reviews should fall back to zero time, got %v", got)
	}
	// Newest wins regardless of order
	if got := lastSignoffTime(signoffs(10, 30, 20)); !got.Equal(rt(30)) {
		t.Errorf("expected newest signoff rt(30), got %v", got)
	}
}

func TestThreadHasNewActivity(t *testing.T) {
	nestedNew := &comment.Comment{ID: "c1", Timestamp: rt(5), Replies: []*comment.Comment{
		{ID: "r1", Timestamp: rt(6), Replies: []*comment.Comment{
			{ID: "r2", Timestamp: rt(15)}, // nested two deep, after signoff
		}},
	}}
	allOld := &comment.Comment{ID: "c2", Timestamp: rt(5), Replies: []*comment.Comment{
		{ID: "r3", Timestamp: rt(8)},
	}}
	newRoot := &comment.Comment{ID: "c3", Timestamp: rt(15)}

	since := rt(10)
	if !threadHasNewActivity(nestedNew, since) {
		t.Error("nested reply after signoff should count as new activity")
	}
	if threadHasNewActivity(allOld, since) {
		t.Error("thread fully before signoff must not count as new")
	}
	if !threadHasNewActivity(newRoot, since) {
		t.Error("root created after signoff should count as new activity")
	}
	// No reviews -> zero time fallback: everything (with real timestamps) is new
	if !threadHasNewActivity(allOld, lastSignoffTime(nil)) {
		t.Error("with no signoffs everything should be new")
	}
}

func TestRoundNumberPartitions(t *testing.T) {
	reviews := signoffs(10, 20)
	cases := []struct {
		ts   time.Time
		want int
	}{
		{rt(5), 1},  // before first signoff
		{rt(15), 2}, // between signoffs
		{rt(25), 3}, // after both
	}
	for _, tc := range cases {
		if got := roundNumber(tc.ts, reviews); got != tc.want {
			t.Errorf("roundNumber(%v) = %d, want %d", tc.ts, got, tc.want)
		}
	}
	if got := roundNumber(rt(5), nil); got != 1 {
		t.Errorf("no signoffs should always be round 1, got %d", got)
	}
}

// --- NEW badges --------------------------------------------------------------

func TestSidebarNewBadgeOnExpandedAndCollapsedRows(t *testing.T) {
	m := testModel([]*comment.Comment{
		// Line 5 (expanded, selectedComment=0): old root, reply after signoff
		{ID: "c1", Line: 5, Text: "first", Author: "rcliao", Timestamp: rt(5),
			Replies: []*comment.Comment{{ID: "r1", Author: "claude", Line: 5, Text: "answer", Timestamp: rt(15)}}},
		// Line 9 (collapsed): one stale thread, one created after signoff
		{ID: "c2", Line: 9, Text: "stale", Author: "rcliao", Timestamp: rt(5)},
		{ID: "c3", Line: 9, Text: "fresh", Author: "claude", Timestamp: rt(25)},
	})
	m.doc.Reviews = signoffs(10)

	out := m.renderComments()
	if got := strings.Count(out, "NEW"); got != 2 {
		t.Errorf("expected NEW on expanded c1 and collapsed c3 (2 badges), got %d in:\n%s", got, out)
	}
}

func TestSidebarNoNewBadgeWhenAllSeen(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "old", Author: "rcliao", Timestamp: rt(5)},
	})
	m.doc.Reviews = signoffs(10)
	if out := m.renderComments(); strings.Contains(out, "NEW") {
		t.Errorf("no thread is newer than the signoff, badge must not render:\n%s", out)
	}
}

func TestLineSummaryNewBadge(t *testing.T) {
	threads := []*comment.Comment{
		{Author: "rcliao", Text: "note", Timestamp: rt(5),
			Replies: []*comment.Comment{{Author: "claude", Text: "late reply", Timestamp: rt(15)}}},
	}
	if got := lineSummary(threads, rt(10)); !strings.Contains(got, "NEW") {
		t.Errorf("summary should carry NEW badge for post-signoff reply, got %q", got)
	}
	if got := lineSummary(threads, rt(20)); strings.Contains(got, "NEW") {
		t.Errorf("summary must not carry NEW when everything is seen, got %q", got)
	}
}

func TestDocumentRenderShowsNewBadgeInSummary(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "note", Author: "rcliao", Timestamp: rt(15)},
	})
	m.doc.Reviews = signoffs(10)
	if out := m.renderDocument(); !strings.Contains(out, "NEW") {
		t.Errorf("document virtual-text summary should show NEW badge, got:\n%s", out)
	}
	lineSelectAt(m, 5)
	if out := m.renderDocumentWithCursor(); !strings.Contains(out, "NEW") {
		t.Errorf("cursor view summary should show NEW badge too, got:\n%s", out)
	}
}

// --- Round separators in the expanded thread timeline -----------------------

func TestExpandedThreadShowsRoundSeparators(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "root question", Author: "rcliao", Timestamp: rt(5),
			Replies: []*comment.Comment{
				{ID: "r1", Author: "claude", Line: 5, Text: "round two answer", Timestamp: rt(15)},
				{ID: "r2", Author: "rcliao", Line: 5, Text: "round three follow-up", Timestamp: rt(25)},
			}},
	})
	m.doc.Reviews = signoffs(10, 20)
	lineSelectAt(m, 5)

	out := m.renderComments()
	if !strings.Contains(out, "── round 2 ──") || !strings.Contains(out, "── round 3 ──") {
		t.Errorf("expected round 2 and round 3 separators, got:\n%s", out)
	}
	if got := strings.Count(out, "── round"); got != 2 {
		t.Errorf("expected exactly 2 separators, got %d in:\n%s", got, out)
	}
}

func TestNoSeparatorsWithinOneRound(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "root", Author: "rcliao", Timestamp: rt(5),
			Replies: []*comment.Comment{
				{ID: "r1", Author: "claude", Line: 5, Text: "same round reply", Timestamp: rt(6)},
			}},
	})
	m.doc.Reviews = signoffs(10)
	lineSelectAt(m, 5)

	if out := m.renderComments(); strings.Contains(out, "── round") {
		t.Errorf("replies in the root's round must not get a separator:\n%s", out)
	}
}

func TestRoundSeparatorOnNestedReply(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "root", Author: "rcliao", Timestamp: rt(5),
			Replies: []*comment.Comment{
				{ID: "r1", Author: "claude", Line: 5, Text: "old reply", Timestamp: rt(6),
					Replies: []*comment.Comment{
						{ID: "r2", Author: "rcliao", Line: 5, Text: "nested after signoff", Timestamp: rt(15)},
					}},
			}},
	})
	m.doc.Reviews = signoffs(10)
	lineSelectAt(m, 5)

	out := m.renderComments()
	if got := strings.Count(out, "── round 2 ──"); got != 1 {
		t.Errorf("nested reply straddling the signoff should get one separator, got %d in:\n%s", got, out)
	}
}

// --- ]r / [r new-activity motions -------------------------------------------

func TestNewActivityMotions(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 3, Text: "seen", Author: "rcliao", Timestamp: rt(5)},
		{ID: "c2", Line: 5, Text: "has late reply", Author: "rcliao", Timestamp: rt(5),
			Replies: []*comment.Comment{{ID: "r1", Author: "claude", Line: 5, Text: "late", Timestamp: rt(15)}}},
		{ID: "c3", Line: 9, Text: "fresh thread", Author: "claude", Timestamp: rt(25)},
	})
	m.doc.Reviews = signoffs(10)
	lineSelectAt(m, 1)

	// ]r skips line 3 (seen) and lands on line 5 (new reply)
	next, _ := m.handleLineSelectKeys(keyMsg("n"))
	nm := next.(Model)
	if nm.selectedLine != 5 {
		t.Fatalf("]r should skip seen line 3 and jump to 5, got %d", nm.selectedLine)
	}
	// ]r again lands on line 9 (fresh root)
	next2, _ := nm.handleLineSelectKeys(keyMsg("n"))
	nm2 := next2.(Model)
	if nm2.selectedLine != 9 {
		t.Fatalf("]r should jump to fresh thread at 9, got %d", nm2.selectedLine)
	}
	// No further new activity: cursor stays put
	next3, _ := nm2.handleLineSelectKeys(keyMsg("n"))
	if got := next3.(Model).selectedLine; got != 9 {
		t.Errorf("]r with nothing ahead should stay at 9, got %d", got)
	}
	// [r goes back to 5, then stays (line 3 has no new activity)
	prev, _ := nm2.handleLineSelectKeys(keyMsg("N"))
	pm := prev.(Model)
	if pm.selectedLine != 5 {
		t.Fatalf("[r should jump back to 5, got %d", pm.selectedLine)
	}
	prev2, _ := pm.handleLineSelectKeys(keyMsg("N"))
	if got := prev2.(Model).selectedLine; got != 5 {
		t.Errorf("[r past seen threads should stay at 5, got %d", got)
	}
}

func TestNewActivityMotionsZeroSignoffFallback(t *testing.T) {
	// No reviews: everything with a real timestamp is new
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 3, Text: "a", Author: "rcliao", Timestamp: rt(5)},
		{ID: "c2", Line: 9, Text: "b", Author: "rcliao", Timestamp: rt(6)},
	})
	lineSelectAt(m, 1)

	next, _ := m.handleLineSelectKeys(keyMsg("n"))
	if got := next.(Model).selectedLine; got != 3 {
		t.Errorf("]r with no signoffs should treat every thread as new, got line %d", got)
	}
}

func TestHelpAndHintBarMentionNewMotions(t *testing.T) {
	if out := renderHelpOverlay(); !strings.Contains(out, "n / N") {
		t.Error("help overlay should document the  n/N  motions")
	}
	m := testModel(nil)
	m.width, m.height = 120, 40
	m.handleResize()
	m.mode = ModeLineSelect
	if out := m.viewBrowse(); !strings.Contains(out, "n/N:") {
		t.Error("line-select hint bar should mention  n/N ")
	}
}
