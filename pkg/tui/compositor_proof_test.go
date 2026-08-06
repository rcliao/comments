package tui

// Spike proof for Phase 3 (docs/plan-tui-in-context.md): lipgloss v2's
// native compositor is importable from this package and composes two layers
// with real z-order — a dialog drawn OVER document content in one frame,
// both visible. Not wired into the UI yet; this pins the API the dialog
// stack will build on.

import (
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
)

func TestLipglossV2CompositorComposesDialogOverDocument(t *testing.T) {
	doc := strings.TrimRight(strings.Repeat("document line of prose text here\n", 6), "\n")
	dialog := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Render("[ reply dialog ]")

	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(doc).Z(0),
		lipgloss.NewLayer(dialog).ID("dialog").X(4).Y(1).Z(1),
	)
	out := comp.Render()

	if !strings.Contains(out, "[ reply dialog ]") {
		t.Fatalf("dialog layer missing from composed frame:\n%s", out)
	}
	if !strings.Contains(out, "document line of prose") {
		t.Fatalf("document layer not visible around the dialog:\n%s", out)
	}
	// z-order is real: the dialog overwrites the doc cells it covers, so the
	// second row can no longer contain the full uninterrupted doc line
	rows := strings.Split(out, "\n")
	if strings.Contains(rows[1], "document line of prose text here") {
		t.Errorf("dialog should overwrite covered doc cells on row 2, got %q", rows[1])
	}
	// hit testing works: a point inside the dialog box resolves to the
	// dialog layer (note: Hit only reports layers that carry an ID)
	if hit := comp.Hit(6, 1); hit.Empty() || hit.ID() != "dialog" {
		t.Errorf("Hit(6,1) inside the dialog should resolve to the dialog layer, got %q", hit.ID())
	}
}
