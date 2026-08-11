package tui

// / search (vim-adjacent, decided in keybind review): incremental over
// document lines — typing jumps the line-select cursor to the first match at
// or below the origin, Enter accepts, Esc restores the origin. n/N stay on
// NEW-activity navigation (kept deliberately), so repeat-search is vim's own
// other idiom: / then Enter on an empty query jumps to the NEXT match of the
// last accepted query, wrapping.

import (
	"fmt"
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
	return m.findMatchDir(query, from, +1)
}

func (m *Model) findMatchDir(query string, from, dir int) int {
	if strings.TrimSpace(query) == "" || m.doc == nil {
		return 0
	}
	lines := strings.Split(m.doc.Content, "\n")
	q := strings.ToLower(query)
	n := len(lines)
	for i := 0; i < n; i++ {
		idx := ((from-1+dir*i)%n + n) % n
		if strings.Contains(strings.ToLower(lines[idx]), q) {
			return idx + 1
		}
	}
	return 0
}

// matchStats reports which match the cursor sits on and how many exist, so
// the prompt can say "2/5" — a cursor that legitimately does not move (the
// query matches the current line) stops reading as "search is broken".
func (m *Model) matchStats(query string) (cur, total int) {
	if strings.TrimSpace(query) == "" || m.doc == nil {
		return 0, 0
	}
	q := strings.ToLower(query)
	for i, line := range strings.Split(m.doc.Content, "\n") {
		if strings.Contains(strings.ToLower(line), q) {
			total++
			if i+1 <= m.selectedLine {
				cur = total
			}
		}
	}
	return cur, total
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

	case "tab", "ctrl+n", "down":
		// Next match while the prompt is open — the answer to "it didn't
		// move": hop onward without leaving the search
		q := strings.TrimSpace(m.searchInput.Value())
		if q == "" {
			q = m.searchQuery
		}
		if line := m.findMatchDir(q, m.selectedLine+1, +1); line > 0 {
			m.selectedLine = line
			m.refreshCursorView()
		}
		return m, nil

	case "shift+tab", "ctrl+p", "up":
		q := strings.TrimSpace(m.searchInput.Value())
		if q == "" {
			q = m.searchQuery
		}
		if line := m.findMatchDir(q, m.selectedLine-1, -1); line > 0 {
			m.selectedLine = line
			m.refreshCursorView()
		}
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

// activeSearchQuery is the query to highlight while the prompt is open:
// what is being typed, falling back to the last accepted query (cycling an
// empty prompt still shows where the hops go). "" outside search mode.
func (m *Model) activeSearchQuery() string {
	if m.mode != ModeSearch {
		return ""
	}
	if q := strings.TrimSpace(m.searchInput.Value()); q != "" {
		return q
	}
	return m.searchQuery
}

// highlightMatches renders a line raw except its match substrings, which get
// the search-hit background — hlsearch during incsearch. Style-only: the
// ANSI-stripped result is byte-identical to the input.
func (m *Model) highlightMatches(line, query string) string {
	lower := strings.ToLower(line)
	q := strings.ToLower(query)
	var b strings.Builder
	last := 0
	for {
		i := strings.Index(lower[last:], q)
		if i < 0 {
			break
		}
		start := last + i
		b.WriteString(line[last:start])
		b.WriteString(m.styles.searchHit.Render(line[start : start+len(query)]))
		last = start + len(query)
	}
	b.WriteString(line[last:])
	return b.String()
}

// viewSearch renders the live view with a one-line search prompt at the
// bottom, showing the match state.
func (m Model) viewSearch() string {
	state := ""
	if q := strings.TrimSpace(m.searchInput.Value()); q != "" {
		if cur, total := m.matchStats(q); total == 0 {
			state = m.styles.help.Render("  no match")
		} else {
			state = m.styles.help.Render(fmt.Sprintf("  %d/%d · tab: next", cur, total))
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
