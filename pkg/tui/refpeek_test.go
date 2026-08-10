package tui

// Tests for reference peek (docs/design-reference-jump.md): ref map built at
// load, link styling, f -> peek -> esc flow, cycling, error state, tiny
// windows, and $EDITOR command construction.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rcliao/comments/pkg/comment"
)

// refTestModel builds a model over a real on-disk doc citing a real target,
// so resolution works end to end.
func refTestModel(t *testing.T) (*Model, string) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "research.md")
	targetContent := "# Research\n\n## Findings\n\nline five finding\nline six finding\nline seven finding\n"
	if err := os.WriteFile(target, []byte(targetContent), 0o644); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(dir, "plan.md")
	docContent := "# Plan\n\nPer research.md:5 the finding holds.\nAlso see [notes](./research.md#findings) and missing.md:9 here.\n"
	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatal(err)
	}

	doc := &comment.DocumentWithComments{Content: docContent, Threads: []*comment.Comment{}}
	m := NewModelWithFile(doc, docPath)
	m.width, m.height = 100, 40
	m.handleResize()
	return &m, target
}

func TestRefMapBuiltAtLoad(t *testing.T) {
	m, target := refTestModel(t)

	line3 := m.refsByLine[3]
	if len(line3) != 1 || !line3[0].resolved || line3[0].absPath != target || line3[0].Line != 5 {
		t.Fatalf("line 3 should carry one resolved ref to research.md:5, got %+v", line3)
	}
	line4 := m.refsByLine[4]
	if len(line4) != 2 {
		t.Fatalf("line 4 should carry the md link and the missing ref, got %+v", line4)
	}
	if !line4[0].resolved || line4[0].Heading != "findings" {
		t.Errorf("md link with #findings should resolve, got %+v", line4[0])
	}
	if line4[1].resolved {
		t.Errorf("missing.md must not resolve, got %+v", line4[1])
	}
}

func TestResolvedRefsGetLinkStyling(t *testing.T) {
	m, _ := refTestModel(t)

	styled := m.styleDocLine("Per research.md:5 the finding holds.", 3)
	if !strings.Contains(styled, "\x1b[") {
		t.Error("resolved reference line should carry styling")
	}
	// character preservation: ANSI-stripped output equals input
	if got := stripANSI(styled); got != "Per research.md:5 the finding holds." {
		t.Errorf("styling must not change characters, got %q", got)
	}

	// unresolved-only spans stay unstyled: build a line where the only ref is missing.md:9
	unresolvedOnly := m.styleDocLine("missing.md:9 alone", 99) // line 99 has no refs recorded
	if strings.Contains(unresolvedOnly, "\x1b[4m") {
		t.Error("line without recorded resolved refs must not get link styling")
	}
}

func TestPeekOpenCycleClose(t *testing.T) {
	m, _ := refTestModel(t)
	m.mode = ModeLineSelect
	m.selectedLine = 3

	next, _ := m.handleLineSelectKeys(keyMsg("f"))
	nm := next.(Model)
	if nm.mode != ModeRefPeek {
		t.Fatalf("f on citation line should open peek, mode=%v", nm.mode)
	}
	view := nm.viewRefPeek()
	if !strings.Contains(view, "research.md:5") || !strings.Contains(view, "line five finding") {
		t.Errorf("peek should show target excerpt centered on line 5:\n%s", view)
	}
	if !strings.Contains(view, "►") {
		t.Errorf("cited line should carry the target marker:\n%s", view)
	}

	// esc returns to line-select at the same cursor
	back, _ := nm.handleRefPeekKeys(keyMsg("esc"))
	bm := back.(Model)
	if bm.mode != ModeLineSelect || bm.selectedLine != 3 {
		t.Errorf("esc should return to line-select at cursor 3, got %v line %d", bm.mode, bm.selectedLine)
	}
}

func TestPeekCyclesMultipleRefsAndShowsError(t *testing.T) {
	m, _ := refTestModel(t)
	m.mode = ModeLineSelect
	m.selectedLine = 4 // md link + missing.md:9

	next, _ := m.handleLineSelectKeys(keyMsg("f"))
	nm := next.(Model)
	if !strings.Contains(nm.viewRefPeek(), "(1/2)") {
		t.Errorf("multi-ref peek should show cycle position:\n%s", nm.viewRefPeek())
	}
	// heading ref centers on the Findings heading (line 3 of target)
	if !strings.Contains(nm.viewRefPeek(), "## Findings") {
		t.Errorf("heading ref should show the section:\n%s", nm.viewRefPeek())
	}

	cycled, _ := nm.handleRefPeekKeys(keyMsg("f"))
	cm := cycled.(Model)
	view := cm.viewRefPeek()
	if !strings.Contains(view, "cannot resolve missing.md") {
		t.Errorf("unresolved ref should peek as an error state:\n%s", view)
	}
	// enter on an unresolved ref is a no-op, not a crash
	same, cmd := cm.handleRefPeekKeys(keyMsg("enter"))
	if cmd != nil || same.(Model).mode != ModeRefPeek {
		t.Error("enter on unresolved ref must be a no-op")
	}
}

func TestPeekCompositesOverLiveDocView(t *testing.T) {
	m, _ := refTestModel(t)
	m.mode = ModeLineSelect
	m.selectedLine = 3
	m.refreshCursorView()

	next, _ := m.handleLineSelectKeys(keyMsg("f"))
	nm := next.(Model)
	got := termEscapes.ReplaceAllString(nm.viewContent(), "")
	// The live document stays visible behind the peek box
	if !strings.Contains(got, "# Plan") {
		t.Errorf("document text missing behind the peek:\n%s", got)
	}
	// Peek chrome and target excerpt in the SAME frame
	if !strings.Contains(got, "research.md:5") || !strings.Contains(got, "line five finding") {
		t.Errorf("peek box should show the target excerpt in the same frame:\n%s", got)
	}
}

func TestLineSelectHintSurfacesMultiRefCycle(t *testing.T) {
	m, _ := refTestModel(t)
	m.mode = ModeLineSelect

	m.selectedLine = 4 // md link + missing.md:9 — two references
	if out := m.viewBrowse(); !strings.Contains(out, "f/Tab: cycle 2 refs") {
		t.Errorf("hint bar should surface the f/Tab cycle on a multi-ref line, got:\n%s", out)
	}

	m.selectedLine = 3 // single reference: plain follow hint, no cycle noise
	out := m.viewBrowse()
	if !strings.Contains(out, "f: follow ref") || strings.Contains(out, "cycle 2 refs") {
		t.Errorf("single-ref line should keep the plain follow hint, got:\n%s", out)
	}
}

func TestPeekTinyWindow(t *testing.T) {
	m, _ := refTestModel(t)
	m.width, m.height = 30, 8
	m.handleResize()
	m.mode = ModeLineSelect
	m.selectedLine = 3

	next, _ := m.handleLineSelectKeys(keyMsg("f"))
	nm := next.(Model)
	if nm.mode != ModeRefPeek {
		t.Fatal("peek must still open on tiny windows (never silently escalates)")
	}
	view := nm.viewRefPeek()
	if !strings.Contains(view, "research.md:5") {
		t.Errorf("tiny peek still shows the target header:\n%s", view)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Setenv("EDITOR", "nvim")
	cmd := editorCommand("/tmp/research.md", 42)
	if got := strings.Join(cmd.Args, " "); !strings.HasSuffix(got, "nvim +42 /tmp/research.md") {
		t.Errorf("editor command should be 'nvim +42 <path>', got %q", got)
	}
	cmd = editorCommand("/tmp/research.md", 0)
	if got := strings.Join(cmd.Args, " "); strings.Contains(got, "+") {
		t.Errorf("no cited line -> no +N arg, got %q", got)
	}
	t.Setenv("EDITOR", "")
	if cmd := editorCommand("/x.md", 1); filepath.Base(cmd.Args[0]) != "vi" {
		t.Errorf("EDITOR unset should fall back to vi, got %q", cmd.Args[0])
	}
}

// keyMsg and stripANSI helpers are shared from rendering_test.go

// f on a line with no citations must answer with the empty-state box, not
// silently do nothing (silent no-op reads as "peek is broken").
func TestRefPeekEmptyStateOnCitationlessLine(t *testing.T) {
	m := testModel(nil)
	m.width, m.height = 100, 40
	m.handleResize()
	lineSelectAt(m, 5) // plain body line, no refs
	next, _ := m.handleLineSelectKeys(keyMsg("f"))
	nm := next.(Model)
	if nm.mode != ModeRefPeek {
		t.Fatalf("f should open the peek empty state, got %v", nm.mode)
	}
	if got := frame(nm); !strings.Contains(got, "no citations on line 5") {
		t.Errorf("empty state should explain itself:\n%s", got)
	}
	back, _ := nm.handleRefPeekKeys(keyMsg("esc"))
	if back.(Model).mode != ModeLineSelect {
		t.Errorf("esc should return to line-select")
	}
	// enter on the empty state must not panic
	nm.handleRefPeekKeys(keyMsg("enter"))
}
