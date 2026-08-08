package comment

import (
	"errors"
	"strings"
	"testing"
)

const anchorTextDoc = `# Title

Alpha body text.
Beta body text.
repeated line
middle text
repeated line
  Indented   Gamma line
`

func TestResolveAnchorTextUniqueMatch(t *testing.T) {
	for anchor, want := range map[string]int{
		"Alpha body text.": 3, // whole line
		"Beta body":        4, // substring
		"# Title":          1, // heading
		"indented gamma":   8, // normalized (case + whitespace)
	} {
		got, err := ResolveAnchorText(anchorTextDoc, anchor)
		if err != nil || got != want {
			t.Errorf("ResolveAnchorText(%q) = %d, %v; want %d", anchor, got, err, want)
		}
	}
}

func TestResolveAnchorTextAmbiguousListsCandidates(t *testing.T) {
	_, err := ResolveAnchorText(anchorTextDoc, "repeated line")
	var amb *AmbiguousAnchorError
	if !errors.As(err, &amb) {
		t.Fatalf("expected AmbiguousAnchorError, got %v", err)
	}
	if len(amb.Candidates) != 2 || amb.Candidates[0] != 5 || amb.Candidates[1] != 7 {
		t.Errorf("candidates = %v, want [5 7]", amb.Candidates)
	}
	if !strings.Contains(err.Error(), "5, 7") {
		t.Errorf("error should list candidate lines: %v", err)
	}
}

func TestResolveAnchorTextMissing(t *testing.T) {
	if _, err := ResolveAnchorText(anchorTextDoc, "no such text anywhere"); err == nil {
		t.Error("missing anchor should error")
	}
	if _, err := ResolveAnchorText(anchorTextDoc, "   "); err == nil {
		t.Error("blank anchor should error")
	}
}
