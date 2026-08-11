package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

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

// testStyles returns a default-theme styleSet for tests of pure render helpers
func testStyles() *styleSet { return newStyleSet(themes[DefaultThemeName]) }

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
	st := testStyles()
	for _, tc := range cases {
		got := st.lineMarker(tc.threads)
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
	for l := range strings.SplitSeq(out, "\n") {
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
	if !strings.Contains(out, "⛔") {
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
	for line := range strings.SplitSeq(out, "\n") {
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

func lineSelectAt(m *Model, line int) {
	m.mode = ModeLineSelect
	m.selectedLine = line
	m.refreshSidebar()
}

func TestDiveIntoThreadFromLineSelect(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "on five", Author: "rcliao"},
		{ID: "c2", Line: 9, Text: "on nine", Author: "rcliao"},
	})
	lineSelectAt(m, 5)

	next, _ := m.handleLineSelectKeys(keyMsg("r"))
	nm := next.(Model)
	if nm.mode != ModeThreadView || nm.selectedThread == nil || nm.selectedThread.ID != "c1" {
		t.Fatalf("r should open thread view on the cursor line's thread, got mode=%v thread=%v", nm.mode, nm.selectedThread)
	}

	// Esc returns to line-select at the same cursor
	back, _ := nm.handleThreadViewKeys(keyMsg("esc"))
	bm := back.(Model)
	if bm.mode != ModeLineSelect || bm.selectedLine != 5 {
		t.Errorf("esc should return to line-select at line 5, got mode=%v line=%d", bm.mode, bm.selectedLine)
	}
}

func TestDiveIsNoopOnUncommentedLine(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "x", Author: "rcliao"}})
	lineSelectAt(m, 7)

	next, _ := m.handleLineSelectKeys(keyMsg("r"))
	nm := next.(Model)
	if nm.mode != ModeLineSelect {
		t.Errorf("r on uncommented line should stay in line-select, got %v", nm.mode)
	}
}

func TestTabCyclesThreadsOnSameLine(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "first", Author: "rcliao"},
		{ID: "c2", Line: 5, Text: "second", Author: "rcliao"},
		{ID: "c3", Line: 9, Text: "other", Author: "rcliao"},
	})
	lineSelectAt(m, 5)
	if got := m.focusedThreadAtCursor().ID; got != "c1" {
		t.Fatalf("cursor sync should select first thread on line, got %s", got)
	}

	next, _ := m.handleLineSelectKeys(keyMsg("tab"))
	nm := next.(Model)
	if got := nm.focusedThreadAtCursor().ID; got != "c2" {
		t.Errorf("tab should cycle to second thread, got %s", got)
	}
	next2, _ := nm.handleLineSelectKeys(keyMsg("tab"))
	nm2 := next2.(Model)
	if got := nm2.focusedThreadAtCursor().ID; got != "c1" {
		t.Errorf("tab should wrap back to first thread, got %s", got)
	}

	// r opens the Tab-selected thread, not the first one
	dive, _ := nm.handleLineSelectKeys(keyMsg("r"))
	dm := dive.(Model)
	if dm.selectedThread == nil || dm.selectedThread.ID != "c2" {
		t.Errorf("r should open the cycled-to thread c2, got %v", dm.selectedThread)
	}
}

// keyMsg builds a v2 key-press message from a readable key name. Bubbletea v2
// made tea.KeyMsg an interface; presses are tea.KeyPressMsg with Code/Text.
func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	default:
		return tea.KeyPressMsg{Code: []rune(key)[0], Text: key}
	}
}

func TestVerdictDialogFlow(t *testing.T) {
	m := testModel([]*comment.Comment{{ID: "c1", Line: 5, Text: "open", Author: "rcliao"}})
	m.filename = filepath.Join(t.TempDir(), "v.md")
	if err := os.WriteFile(m.filename, []byte(tuiTestDoc), 0644); err != nil {
		t.Fatal(err)
	}

	// q from browse opens the verdict dialog
	next, _ := m.handleBrowseKeys(keyMsg("q"))
	nm := next.(Model)
	if nm.mode != ModeVerdict {
		t.Fatalf("q should open verdict dialog, got %v", nm.mode)
	}
	// esc returns to browse
	back, _ := nm.handleVerdictKeys(keyMsg("esc"))
	if back.(Model).mode != ModeBrowse {
		t.Error("esc should return to previous mode")
	}
	// a approves: records signoff and quits
	done, cmd := nm.handleVerdictKeys(keyMsg("a"))
	dm := done.(Model)
	if dm.VerdictDecision != comment.DecisionApproved || cmd == nil {
		t.Errorf("a should record approval and quit, got %q", dm.VerdictDecision)
	}
	if len(dm.doc.Reviews) != 1 || dm.doc.Reviews[0].Decision != comment.DecisionApproved {
		t.Errorf("signoff not recorded: %v", dm.doc.Reviews)
	}
}

// --- In-place markdown span styling (Phase B step 4) -------------------------

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string { return ansiPattern.ReplaceAllString(s, "") }

// Lipgloss v2 removed the global color profile: Style.Render always emits
// ANSI escape sequences and downsampling happens in the output writer, so
// tests no longer need to force a profile (v1's withANSIProfile is gone).

func TestStyleMarkdownLinePreservesWidth(t *testing.T) {
	lines := []string{
		"plain prose with nothing special",
		"Some **bold** and *italic* and `code` here.",
		"__under bold__ and _under italic_ mixed",
		"- bullet item with **bold** inside",
		"* star bullet stays a bullet, *this* is italic",
		"+ plus bullet",
		"1. numbered item",
		"42) paren numbered item",
		"  - indented bullet",
		"> quoted text with `code`",
		">> nested quote",
		"> - bullet inside a quote",
		"# H1 header",
		"## H2 header with **bold**",
		"`code with **not bold** inside`",
		"broken **unclosed bold and `unclosed tick",
		"a * b times * c spaced stars",
		"",
	}
	st := testStyles()
	for _, line := range lines {
		styled := st.styleMarkdownLine(line)
		if got := stripANSI(styled); got != line {
			t.Errorf("ANSI-stripped output must equal input.\n in: %q\nout: %q", line, got)
		}
	}
}

func TestStyleMarkdownLineStylesSpansWithDimmedGlyphs(t *testing.T) {
	// Pin the legacy palette: this test asserts exact 256-color codes
	st := newStyleSet(themes["ansi"])
	out := st.styleMarkdownLine("mix of **bold**, *ital*, and `code` spans")

	if !strings.Contains(out, "\x1b[") {
		t.Fatal("expected ANSI styling in output")
	}
	// Syntax glyphs kept but dimmed (color 240), not removed
	if !strings.Contains(out, "\x1b[38;5;240m**") {
		t.Errorf("bold glyphs should render dimmed, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;5;240m`") {
		t.Errorf("backtick glyphs should render dimmed, got %q", out)
	}
	// Content styled: bold weight on bold text, code color on code text
	if !strings.Contains(out, "\x1b[1mbold") {
		t.Errorf("bold content should render bold, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;5;173mcode") {
		t.Errorf("code content should render in the code color, got %q", out)
	}
	if !strings.Contains(out, "\x1b[3mital") {
		t.Errorf("italic content should render italic, got %q", out)
	}
}

func TestStyleMarkdownLineColorsBulletsAndQuoteBars(t *testing.T) {
	st := testStyles()
	for _, line := range []string{"- item", "* item", "+ item", "3. item", "> quote"} {
		out := st.styleMarkdownLine(line)
		if out == line {
			t.Errorf("prefix of %q should be styled, got unstyled output", line)
		}
		if got := stripANSI(out); got != line {
			t.Errorf("prefix styling must not change characters: in %q out %q", line, got)
		}
	}
	// The prose after the bullet is NOT swallowed by the bullet style:
	// only the bullet glyph itself is wrapped. (Lipgloss v2 resets styles
	// with the shorter \x1b[m SGR; v1 emitted \x1b[0m.)
	out := st.styleMarkdownLine("- item")
	if !strings.Contains(out, "m-\x1b[m item") {
		t.Errorf("only the bullet glyph should be styled, got %q", out)
	}
}

func TestOpenCommentMotions(t *testing.T) {
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 3, Text: "a", Author: "r"},
		{ID: "c2", Line: 5, Text: "b", Author: "r", Resolved: true},
		{ID: "c3", Line: 9, Text: "c", Author: "r"},
	})
	lineSelectAt(m, 3)

	next, _ := m.handleLineSelectKeys(keyMsg("]"))
	nm := next.(Model)
	if nm.selectedLine != 9 { // skips resolved c2 at line 5
		t.Errorf("] should jump to next OPEN comment line 9, got %d", nm.selectedLine)
	}
	prev, _ := nm.handleLineSelectKeys(keyMsg("["))
	if prev.(Model).selectedLine != 3 {
		t.Errorf("[ should jump back to line 3, got %d", prev.(Model).selectedLine)
	}
}

func TestTruncateRuneSafe(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		max      int
		ellipsis string
		want     string
	}{
		{"short ascii unchanged", "hello", 10, "...", "hello"},
		{"exact fit unchanged", strings.Repeat("a", 60), 60, "...", strings.Repeat("a", 60)},
		{"long ascii cut", strings.Repeat("a", 61), 60, "...", strings.Repeat("a", 57) + "..."},
		{"cjk cut on rune boundary", strings.Repeat("日", 50), 10, "…", strings.Repeat("日", 9) + "…"},
		{"emoji cut on rune boundary", strings.Repeat("🎉", 20), 5, "...", "🎉🎉..."},
		{"mixed ascii cjk", "ab" + strings.Repeat("界", 10), 6, "…", "ab界界界…"},
		{"combining chars stay valid", strings.Repeat("é", 40), 10, "…", ""},
	}
	for _, tc := range cases {
		got := truncate(tc.in, tc.max, tc.ellipsis)
		if !utf8.ValidString(got) {
			t.Errorf("%s: truncate produced invalid UTF-8: %q", tc.name, got)
		}
		if n := len([]rune(got)); n > tc.max {
			t.Errorf("%s: result is %d runes, exceeds max %d: %q", tc.name, n, tc.max, got)
		}
		if tc.want != "" && got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}

	// The old byte-slicing bug: s[:57] inside a multi-byte rune. The helper
	// must never produce the replacement-character mojibake that displayed.
	s := strings.Repeat("汉", 30) // 90 bytes, 30 runes
	got := truncate(s, 60, "...")
	if got != s {
		t.Errorf("30 runes fit in max 60, should be unchanged, got %q", got)
	}
	if bad := truncate(s, 10, "..."); strings.ContainsRune(bad, utf8.RuneError) {
		t.Errorf("truncate split a rune: %q", bad)
	}
}

func TestSidebarSummaryTruncationIsRuneSafe(t *testing.T) {
	// A collapsed (non-focused) sidebar entry with a long CJK text must not
	// render mojibake. Put the long thread on a different line from the
	// focused one so it renders collapsed.
	long := strings.Repeat("測試文字", 30) // 120 runes
	m := testModel([]*comment.Comment{
		{ID: "c1", Line: 5, Text: "focused", Author: "a"},
		{ID: "c2", Line: 9, Text: long, Author: "b"},
	})
	m.selectedComment = 0
	out := m.renderComments()
	if !utf8.ValidString(out) {
		t.Fatal("sidebar output contains invalid UTF-8")
	}
	if strings.ContainsRune(out, utf8.RuneError) {
		t.Error("sidebar output contains U+FFFD mojibake from byte truncation")
	}
	if !strings.Contains(out, "…") {
		t.Error("long collapsed summary should be ellipsized")
	}
}

// Phase 1 of docs/plan-markdown-render.md: fence interiors carry no prose
// styling, chroma highlights code line-preservingly, and comment-trail
// citations inside fences stay link-styled (peekable).
func TestFenceRenderingHighlightsAndPreservesCitations(t *testing.T) {
	content := "# T\n\n```dbml\nTable eval {\n  doc string [pk] // pkg/comment/types.go:140 key\n}\n```\n- a **bold** bullet\n"
	doc := &comment.DocumentWithComments{Content: content, Threads: []*comment.Comment{}}
	m := NewModelWithFile(doc, "/tmp/nonexistent-doc.md")
	m.width, m.height = 100, 40
	m.handleResize()

	// Fence cache: delimiters at 3 and 7, code lines 4-6
	if fl, ok := m.fenceCache[3]; !ok || !fl.delimiter {
		t.Fatalf("line 3 should be a fence delimiter: %+v", m.fenceCache)
	}
	fl := m.fenceCache[5]
	if fl.trail == "" || !strings.Contains(fl.trail, "types.go:140") {
		t.Fatalf("line 5 should split its citation trail, got %+v", fl)
	}
	// Highlighted code carries ANSI but identical raw text
	if stripANSI(fl.code)+stripANSI(fl.trail) != "  doc string [pk] // pkg/comment/types.go:140 key" {
		t.Errorf("fence line content must be byte-identical ignoring ANSI: %q + %q", stripANSI(fl.code), stripANSI(fl.trail))
	}
	// Rendered doc: bullet line still styled, fence line not bullet-styled,
	// and total line count unchanged
	out := m.renderDocument()
	if strings.Count(out, "\n") < strings.Count(content, "\n") {
		t.Errorf("rendering must not drop lines")
	}
	styled := m.styleDocLine("  doc string [pk] // pkg/comment/types.go:140 key", 5)
	if stripANSI(styled) != "  doc string [pk] // pkg/comment/types.go:140 key" {
		t.Errorf("fence line styling changed characters: %q", stripANSI(styled))
	}
}

// Phases 2-3 of docs/plan-markdown-render.md: strikethrough + links styled
// with dimmed markers; heading hashes dim; hrule dims whole-line — all
// byte-identical ignoring ANSI.
func TestTyporaSpansAndGlyphs(t *testing.T) {
	st := testStyles()
	for _, line := range []string{
		"~~vetoed option~~ stays visible",
		"see [the plan](docs/plan.md) for detail",
		"# Heading One",
		"---",
	} {
		var styled string
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "---" {
			styled = st.styleMarkdownLine(line)
		} else {
			styled = st.styleInlineSpans(line)
		}
		if got := stripANSI(styled); got != line {
			t.Errorf("styling changed characters: %q -> %q", line, got)
		}
	}
	if out := stripANSI(st.styleInlineSpans("~~x~~")); out != "~~x~~" {
		t.Errorf("strike markers must remain: %q", out)
	}
}
