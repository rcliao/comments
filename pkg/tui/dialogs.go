package tui

// Dialog composition over the live view (Phase 3 of
// docs/plan-tui-in-context.md): dialogs render as lipgloss v2 layers
// composited over the screen they interrupt, never as boxes Placed over
// blank whitespace. There is deliberately no dialog-stack machine — our
// dialogs never nest more than one deep over a base view, so each dialog
// mode's view builds its box and hands it to dialogOver with baseView();
// modes stay modes and Esc semantics stay per-mode (Esc pops the one layer
// by returning to the mode underneath).

import lipgloss "charm.land/lipgloss/v2"

// baseView renders the live screen a dialog composites over: the thread
// panel view (document + panel) while a thread is open, otherwise the
// browse/line-select screen. The title bar keeps announcing the dialog's
// mode because viewBrowse renders m.mode.
func (m Model) baseView() string {
	if m.selectedThread != nil {
		return m.viewThreadPanel()
	}
	return m.viewBrowse()
}

// dialogOver composites the dialog box centered over the live base view with
// real z-order. The base renders at full brightness; dimming it later is a
// single style call on the base layer here.
func (m Model) dialogOver(base, dialog string) string {
	if !m.ready || m.width <= 0 {
		// No laid-out screen to composite over (unit tests, startup)
		return dialog
	}
	x := max((m.width-lipgloss.Width(dialog))/2, 0)
	y := max((m.height-lipgloss.Height(dialog))/2, 0)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(base).Z(0),
		lipgloss.NewLayer(dialog).X(x).Y(y).Z(1),
	).Render()
}
