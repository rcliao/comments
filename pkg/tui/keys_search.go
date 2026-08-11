package tui

// / search (vim-adjacent, decided in keybind review): incremental over
// document lines — typing jumps the line-select cursor to the first match at
// or below the origin, Enter accepts, Esc restores the origin. n/N stay on
// NEW-activity navigation (kept deliberately), so repeat-search is vim's own
// other idiom: / then Enter on an empty query jumps to the NEXT match of the
// last accepted query, wrapping.

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// openSearch enters the search prompt from browse or line-select.
func (m Model) openSearch() (tea.Model, tea.Cmd) {
	m.searchReturnMode = m.mode
	if m.mode == ModeBrowse {
		// Search moves a cursor; browse has none — matches land in line-select
		m.searchReturnMode = ModeLineSelect
		if m.selectedLine == 0 {
			m.selectedLine = 1
		}
	}
	m.searchOrigin = m.selectedLine
	m.searchInput.Reset()
	m.searchInput.Focus()
	m.mode = ModeSearch
	return m, textarea.Blink
}

// findMatch returns the first document line containing query (case-insensitive)
// at or after `from`, wrapping past the end; 0 when nothing matches.
func (m *Model) findMatch(query string, from int) int {
	if strings.TrimSpace(query) == "" || m.doc == nil {
		return 0
	}
	lines := strings.Split(m.doc.Content, "\n")
	q := strings.ToLower(query)
	for i := 0; i < len(lines); i++ {
		idx := (from - 1 + i) % len(lines)
		if strings.Contains(strings.ToLower(lines[idx]), q) {
			return idx + 1
		}
	}
	return 0
}

func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel: restore where the search started
		m.selectedLine = m.searchOrigin
		m.mode = m.searchReturnMode
		m.searchInput.Blur()
		m.refreshCursorView()
		return m, nil

	case "enter":
		typed := strings.TrimSpace(m.searchInput.Value())
		if typed == "" && m.searchQuery != "" {
			// Empty Enter = next match of the last query (repeat idiom)
			if line := m.findMatch(m.searchQuery, m.selectedLine+1); line > 0 {
				m.selectedLine = line
			}
		} else if typed != "" {
			m.searchQuery = typed
		}
		m.mode = m.searchReturnMode
		m.searchInput.Blur()
		m.refreshCursorView()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	// Incremental: jump to the first match from the origin as the query grows
	if line := m.findMatch(strings.TrimSpace(m.searchInput.Value()), m.searchOrigin); line > 0 {
		m.selectedLine = line
		m.refreshCursorView()
	}
	return m, cmd
}

// viewSearch renders the live view with a one-line search prompt at the
// bottom, showing the match state.
func (m Model) viewSearch() string {
	state := ""
	if q := strings.TrimSpace(m.searchInput.Value()); q != "" {
		if m.findMatch(q, m.searchOrigin) == 0 {
			state = m.styles.help.Render("  no match")
		}
	}
	prompt := m.styles.title.Render("/") + m.searchInput.View() + state
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.theme.Title.Color()).
		Width(min(m.width-4, 60)).
		Render(prompt)
	return m.dialogOver(m.baseView(), box)
}
