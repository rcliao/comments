package tui

import (
	"reflect"
	"sync"
	"testing"

	lipgloss "charm.land/lipgloss/v2"

	"github.com/rcliao/comments/pkg/comment"
)

// resetTheme restores the default startup theme after a test calls SetTheme.
func resetTheme(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
}

func TestThemeByNameLookup(t *testing.T) {
	for _, name := range []string{"nord", "dracula", "gruvbox", "ansi"} {
		th, ok := ThemeByName(name)
		if !ok {
			t.Errorf("ThemeByName(%q) should succeed", name)
		}
		if th != themes[name] {
			t.Errorf("ThemeByName(%q) returned wrong theme", name)
		}
	}
}

func TestThemeByNameUnknownFallsBackToDefault(t *testing.T) {
	th, ok := ThemeByName("solarized")
	if ok {
		t.Error("unknown theme name should return ok=false")
	}
	if th != themes[DefaultThemeName] {
		t.Errorf("unknown theme name should fall back to %s", DefaultThemeName)
	}
}

func TestSetThemeUnknownAppliesDefaultAndReportsFalse(t *testing.T) {
	resetTheme(t)
	SetTheme("dracula")
	if !reflect.DeepEqual(currentStartupTheme(), themes["dracula"]) {
		t.Fatal("SetTheme(dracula) should make dracula the startup theme")
	}
	if SetTheme("no-such-theme") {
		t.Error("SetTheme with unknown name should return false")
	}
	if !reflect.DeepEqual(currentStartupTheme(), themes[DefaultThemeName]) {
		t.Error("SetTheme with unknown name should apply the default theme")
	}
}

func TestSetThemeStylesNewModels(t *testing.T) {
	resetTheme(t)
	SetTheme("gruvbox")
	m := NewModelWithFile(&comment.DocumentWithComments{Content: "# x\n"}, "x.md")
	if !reflect.DeepEqual(m.styles.theme, themes["gruvbox"]) {
		t.Error("models constructed after SetTheme should carry that theme")
	}
}

func TestNewStyleSetRecolorsStyles(t *testing.T) {
	ansi := newStyleSet(themes["ansi"])
	if got := ansi.title.GetForeground(); got != lipgloss.Color("170") {
		t.Errorf("ansi title foreground = %v, want 170", got)
	}
	if got := ansi.blockingMarker.GetForeground(); got != lipgloss.Color("196") {
		t.Errorf("ansi blockingMarker foreground = %v, want 196", got)
	}

	nord := newStyleSet(themes["nord"])
	if got := nord.title.GetForeground(); got != lipgloss.Color("#B48EAD") {
		t.Errorf("nord title foreground = %v, want #B48EAD", got)
	}
	if got := nord.blockingMarker.GetForeground(); got != lipgloss.Color("#BF616A") {
		t.Errorf("nord blockingMarker foreground = %v, want #BF616A", got)
	}
	if got := nord.cursor.GetBackground(); got != lipgloss.Color("#3B4252") {
		t.Errorf("nord cursor background = %v, want #3B4252", got)
	}

	// Non-color attributes carry over into every set
	if w := nord.lineNumber.GetWidth(); w != 4 {
		t.Errorf("lineNumber width = %d, want 4", w)
	}
}

func TestStyleSetThemeDrivesTypeColors(t *testing.T) {
	if got := newStyleSet(themes["gruvbox"]).getCommentTypeColor("[B] broken"); got != "#FB4934" {
		t.Errorf("gruvbox [B] color = %q, want #FB4934", got)
	}
	if got := newStyleSet(themes["ansi"]).getCommentTypeColor("[Q] question"); got != "220" {
		t.Errorf("ansi [Q] color = %q, want 220", got)
	}
}

func TestEveryThemeDefinesEveryRole(t *testing.T) {
	for name, th := range themes {
		v := reflect.ValueOf(th)
		tt := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).String() == "" {
				t.Errorf("theme %q: role %s has zero value", name, tt.Field(i).Name)
			}
		}
	}
}

func TestDefaultThemeIsNord(t *testing.T) {
	if _, ok := themes[DefaultThemeName]; !ok {
		t.Fatalf("default theme %q not registered", DefaultThemeName)
	}
	if DefaultThemeName != "nord" {
		t.Errorf("default theme = %q, want nord", DefaultThemeName)
	}
}

// TestSetThemeConcurrentWithRendering pins the t.Parallel readiness contract:
// SetTheme only touches the mutex-guarded startup theme, and models render
// from their own immutable styleSet, so theme switching in one goroutine can
// never race construction or rendering in another (verified under -race).
func TestSetThemeConcurrentWithRendering(t *testing.T) {
	resetTheme(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			SetTheme("dracula")
		}()
		go func() {
			defer wg.Done()
			m := NewModelWithFile(&comment.DocumentWithComments{
				Content: "# x\n\nbody\n",
				Threads: []*comment.Comment{{ID: "c1", Line: 3, Text: "note", Author: "a"}},
			}, "x.md")
			_ = m.renderDocument()
			_ = m.renderComments()
			_ = m.styles.renderHelpOverlay()
		}()
	}
	wg.Wait()
}

// The gutter comment-count marker must never share a color with the cursor
// accent — identical values made counts blend into the highlighted line
// number (live review).
func TestMarkerDiffersFromCursorAccent(t *testing.T) {
	for name, th := range themes {
		if th.Marker == th.CursorAccent {
			t.Errorf("theme %q: Marker == CursorAccent (%s) — gutter counts must have their own color", name, th.Marker)
		}
	}
}

// Every theme must define the changed-line accent, and it must differ from the
// plain line-number color or the mark is invisible.
func TestEveryThemeDefinesDistinctChangedColor(t *testing.T) {
	for name, th := range themes {
		if th.Changed == "" {
			t.Errorf("theme %s: Changed color unset", name)
		}
		if th.Changed == th.LineNumber {
			t.Errorf("theme %s: Changed must differ from LineNumber", name)
		}
	}
	if got := newStyleSet(themes["nord"]).changedLineNum.GetForeground(); got != lipgloss.Color("#A3BE8C") {
		t.Errorf("nord changedLineNum = %v", got)
	}
}
