package tui

import (
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

const tuiTestDoc = `# Title

## Alpha

Alpha body text.

## Beta

Beta body text.
`

func testModel(threads []*comment.Comment) *Model {
	doc := &comment.DocumentWithComments{Content: tuiTestDoc, Threads: threads}
	m := NewModelWithFile(doc, "test.md")
	return &m
}

func TestLineMarkerVariants(t *testing.T) {
	cases := []struct {
		name    string
		threads []*comment.Comment
		want    string
	}{
		{"no threads", nil, "  "},
		{"unresolved", []*comment.Comment{{Text: "a"}}, "💬1"},
		{"two unresolved", []*comment.Comment{{Text: "a"}, {Text: "b"}}, "💬2"},
		{"blocking wins", []*comment.Comment{{Text: "a"}, {Text: "b", Blocking: true}}, "⛔2"},
		{"resolved blocking ignored", []*comment.Comment{{Text: "a", Blocking: true, Resolved: true}}, "✓"},
		{"all resolved", []*comment.Comment{{Text: "a", Resolved: true}}, "✓"},
	}
	for _, tc := range cases {
		got := lineMarker(tc.threads)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: marker %q does not contain %q", tc.name, got, tc.want)
		}
	}
}

func TestRenderDocumentGutterMarkers(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "note", Blocking: true},
		{ID: "c2", Line: 9, Text: "done", Resolved: true},
	})
	out := m.renderDocument()
	if !strings.Contains(out, "⛔1") {
		t.Error("blocking gutter marker missing from document render")
	}
	if !strings.Contains(out, "✓") {
		t.Error("resolved gutter marker missing from document render")
	}
}

func TestSidebarGroupsAndBadges(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "first", SectionPath: "Title > Alpha"},
		{ID: "c2", Line: 5, Text: "second", SectionPath: "Title > Alpha"},
		{ID: "c3", Line: 9, Text: "other", SectionPath: "Title > Beta"},
	})
	out := m.renderComments()

	if !strings.Contains(out, "2 threads") {
		t.Errorf("stacked line should show count badge, got:\n%s", out)
	}
	if !strings.Contains(out, "Alpha (Line 5)") || !strings.Contains(out, "Beta (Line 9)") {
		t.Errorf("group headers should show section short name + line, got:\n%s", out)
	}
	// selectedComment=0 -> line 5 group is focused/expanded, line 9 collapsed
	if !strings.Contains(out, "▼") || !strings.Contains(out, "▸") {
		t.Errorf("expected one expanded (▼) and one collapsed (▸) group, got:\n%s", out)
	}
}

func TestSidebarFocusFollowsCursor(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "alpha note"},
		{ID: "c2", Line: 9, Text: "beta note"},
	})
	m.mode = ModeLineSelect
	m.selectedLine = 9

	out := m.renderComments()
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		if strings.Contains(l, "▼") && !strings.Contains(l, "Line 9") {
			t.Errorf("expanded group should be the cursor line (9), got: %s", l)
		}
	}
	if m.focusLine() != 9 {
		t.Errorf("focusLine should follow cursor in line-select mode, got %d", m.focusLine())
	}
}

func TestSidebarShowsBlockingAndConfidenceMarkers(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "must fix", Blocking: true, AnchorConfidence: comment.ConfidenceFuzzy},
	})
	out := m.renderComments()
	if !strings.Contains(out, "[BLOCKING]") {
		t.Error("blocking thread marker missing from sidebar")
	}
	if !strings.Contains(out, "~fuzzy") {
		t.Error("fuzzy anchor-confidence marker missing from sidebar")
	}
}

func TestSidebarOrderedByLine(t *testing.T) {
	// Threads stored out of order must render in document order
	m := testModel([]*comment.Comment{
		{ID: "c9", Line: 9, Text: "later"},
		{ID: "c5", Line: 5, Text: "earlier"},
	})
	out := m.renderComments()
	if strings.Index(out, "Line 5") > strings.Index(out, "Line 9") {
		t.Error("sidebar groups not ordered by document line")
	}
}

func TestExpandedThreadShowsRepliesInline(t *testing.T) {
	reply := &comment.Comment{ID: "r1", Author: "claude", Line: 5,
		Text: "Adopted as the core of G3 with count badges"}
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "make the sidebar follow the cursor", Author: "rcliao",
			Replies: []*comment.Comment{reply}},
		{ID: "c2", Line: 9, Text: "other thread", Author: "rcliao",
			Replies: []*comment.Comment{{ID: "r2", Author: "claude", Line: 9, Text: "hidden reply"}}},
	})
	m.mode = ModeLineSelect
	m.selectedLine = 5

	out := m.renderComments()
	if !strings.Contains(out, "Adopted as the core of G3") {
		t.Errorf("expanded thread should show reply text inline, got:\n%s", out)
	}
	if !strings.Contains(out, "└─ @claude") {
		t.Errorf("reply meta line missing, got:\n%s", out)
	}
	if strings.Contains(out, "hidden reply") {
		t.Errorf("collapsed group must not show reply bodies, got:\n%s", out)
	}
}

func TestExpandedThreadWrapsLongText(t *testing.T) {
	long := strings.Repeat("wrapme ", 30) // ~210 chars, must wrap at sidebar width
	m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: long, Author: "rcliao"}})
	m.mode = ModeLineSelect
	m.selectedLine = 5

	out := m.renderComments()
	for _, line := range strings.Split(out, "\n") {
		if len(line) > 120 { // generous bound: styled + indented but wrapped
			t.Errorf("expanded thread line not wrapped (%d chars): %.60s…", len(line), line)
		}
	}
}

func TestNestedRepliesRenderRecursively(t *testing.T) {
	nested := &comment.Comment{ID: "r2", Author: "rcliao", Line: 5, Text: "nested answer"}
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "root", Author: "rcliao", Replies: []*comment.Comment{
			{ID: "r1", Author: "claude", Line: 5, Text: "first reply", Replies: []*comment.Comment{nested}},
		}},
	})
	m.mode = ModeLineSelect
	m.selectedLine = 5

	out := m.renderComments()
	if !strings.Contains(out, "first reply") || !strings.Contains(out, "nested answer") {
		t.Errorf("nested replies should render recursively, got:\n%s", out)
	}
}
