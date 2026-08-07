package tui

// Per-dialog frame assertions (Phase 3 of docs/plan-tui-in-context.md):
// every dialog renders as a popup composited over the live view, so the
// document text must be visible in the SAME frame as the dialog chrome.
// Each case drives a real key route into the dialog.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

func TestTitleBarTruncatesLongPathKeepsMode(t *testing.T) {
	m := *testModel(nil)
	m.filename = "/very/deep/" + strings.Repeat("dir/", 20) + "doc.md"
	m = drive(t, m, tea.WindowSizeMsg{Width: 60, Height: 20})

	// JoinVertical pads rows with trailing spaces; measure the bar text itself
	first := strings.TrimRight(strings.SplitN(frame(m), "\n", 2)[0], " ")
	if !strings.Contains(first, "BROWSE") {
		t.Errorf("mode indicator must survive a long path, got %q", first)
	}
	if !strings.Contains(first, "…") {
		t.Errorf("long path should be truncated with an ellipsis, got %q", first)
	}
	if w := lipgloss.Width(first); w > 60 {
		t.Errorf("title bar must fit the terminal width (60), got %d: %q", w, first)
	}

	// Short paths render untouched
	m.filename = "doc.md"
	first = strings.SplitN(frame(m), "\n", 2)[0]
	if !strings.Contains(first, "doc.md - BROWSE") {
		t.Errorf("short path should render whole, got %q", first)
	}
}

func TestDialogsShowDocumentInSameFrame(t *testing.T) {
	cases := []struct {
		name   string
		enter  func(t *testing.T) Model
		mode   ViewMode
		chrome []string
	}{
		{
			name: "add-comment",
			enter: func(t *testing.T) Model {
				return drive(t, panelTestModel(t), keyMsg("c"),
					keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("c"))
			},
			mode:   ModeAddComment,
			chrome: []string{"Add Comment at Line 5", "Ctrl+S: save"},
		},
		{
			name: "choose-target",
			enter: func(t *testing.T) Model {
				// cursor starts on heading line 1: c routes through the choice
				return drive(t, panelTestModel(t), keyMsg("c"), keyMsg("c"))
			},
			mode:   ModeChooseTarget,
			chrome: []string{"Add comment to:"},
		},
		{
			name: "suggestion-type",
			enter: func(t *testing.T) Model {
				return drive(t, panelTestModel(t), keyMsg("c"), keyMsg("s"))
			},
			mode:   ModeSelectSuggestionType,
			chrome: []string{"Create suggestion for:"},
		},
		{
			name: "add-suggestion",
			enter: func(t *testing.T) Model {
				return drive(t, panelTestModel(t), keyMsg("c"),
					keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("j"), keyMsg("s"), keyMsg("enter"))
			},
			mode:   ModeAddSuggestion,
			chrome: []string{"Create Edit Suggestion"},
		},
		{
			name: "reply",
			enter: func(t *testing.T) Model {
				return drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("r"))
			},
			mode: ModeReply,
			// the composer docks INSIDE the panel (no second box, no repeated
			// title): the doc, the thread and its header all stay visible
			chrome: []string{"Ctrl+S: save reply", "Thread at Line 5", "Enter your reply..."},
		},
		{
			name: "resolve",
			enter: func(t *testing.T) Model {
				return drive(t, openThreadAtLine5(t, panelTestModel(t)), keyMsg("x"))
			},
			mode:   ModeResolve,
			chrome: []string{"Resolve this thread?", "Thread at Line 5"},
		},
		{
			name: "verdict",
			enter: func(t *testing.T) Model {
				return drive(t, panelTestModel(t), keyMsg("q"))
			},
			mode:   ModeVerdict,
			chrome: []string{"Submit review for"},
		},
		{
			name: "help",
			enter: func(t *testing.T) Model {
				// the help box is tall: a roomier terminal keeps the doc's
				// first line clear of the centered popup
				return drive(t, panelTestModel(t),
					tea.WindowSizeMsg{Width: 140, Height: 50}, keyMsg("?"))
			},
			mode:   ModeHelp,
			chrome: []string{"Keybindings"},
		},
		{
			name: "toc",
			enter: func(t *testing.T) Model {
				return drive(t, panelTestModel(t), keyMsg("t"))
			},
			mode:   ModeTOC,
			chrome: []string{"Table of Contents"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.enter(t)
			if m.mode != tc.mode {
				t.Fatalf("expected mode %v, got %v", tc.mode, m.mode)
			}
			got := frame(m)
			// The live document is visible behind the dialog ("# Title" is
			// the document's first line, clear of every centered dialog box)
			if !strings.Contains(got, "# Title") {
				t.Errorf("document text missing behind the %s dialog:\n%s", tc.name, got)
			}
			for _, want := range tc.chrome {
				if !strings.Contains(got, want) {
					t.Errorf("%s dialog chrome %q missing from the frame:\n%s", tc.name, want, got)
				}
			}
		})
	}
}
