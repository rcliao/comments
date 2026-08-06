package tui

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// resetTheme restores the default theme after a test mutates package styles.
func resetTheme(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { applyTheme(themes[DefaultThemeName]) })
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
	if !reflect.DeepEqual(activeTheme, themes["dracula"]) {
		t.Fatal("SetTheme(dracula) should make dracula active")
	}
	if SetTheme("no-such-theme") {
		t.Error("SetTheme with unknown name should return false")
	}
	if !reflect.DeepEqual(activeTheme, themes[DefaultThemeName]) {
		t.Error("SetTheme with unknown name should apply the default theme")
	}
}

func TestApplyThemeRecolorsStyles(t *testing.T) {
	resetTheme(t)

	applyTheme(themes["ansi"])
	if got := titleStyle.GetForeground(); got != lipgloss.Color("170") {
		t.Errorf("ansi titleStyle foreground = %v, want 170", got)
	}
	if got := blockingMarkerStyle.GetForeground(); got != lipgloss.Color("196") {
		t.Errorf("ansi blockingMarkerStyle foreground = %v, want 196", got)
	}

	applyTheme(themes["nord"])
	if got := titleStyle.GetForeground(); got != lipgloss.Color("#B48EAD") {
		t.Errorf("nord titleStyle foreground = %v, want #B48EAD", got)
	}
	if got := blockingMarkerStyle.GetForeground(); got != lipgloss.Color("#BF616A") {
		t.Errorf("nord blockingMarkerStyle foreground = %v, want #BF616A", got)
	}
	if got := cursorStyle.GetBackground(); got != lipgloss.Color("#3B4252") {
		t.Errorf("nord cursorStyle background = %v, want #3B4252", got)
	}

	// Non-color attributes survive reassignment
	if w := lineNumberStyle.GetWidth(); w != 4 {
		t.Errorf("lineNumberStyle width = %d, want 4", w)
	}
}

func TestApplyThemeSetsActiveThemeForTypeColors(t *testing.T) {
	resetTheme(t)
	applyTheme(themes["gruvbox"])
	if got := getCommentTypeColor("[B] broken"); got != "#FB4934" {
		t.Errorf("gruvbox [B] color = %q, want #FB4934", got)
	}
	applyTheme(themes["ansi"])
	if got := getCommentTypeColor("[Q] question"); got != "220" {
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
