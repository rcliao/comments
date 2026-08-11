package tui

// Aligned table rendering (live review: ragged pipes made tables unreviewable
// — the alignment-vs-copy-fidelity trade resolved in favor of review).
// Display-only: each row stays exactly one source line, but cells pad to the
// block's column widths, so the bytes a terminal copy yields differ from
// source. This is the renderer's ONE deliberate byte divergence; suggest
// --original against a copied table row must use the raw source instead.

import (
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// buildTableCache maps each table row's line number to its aligned raw text.
// Rebuilt wherever the document content changes (same sites as fenceCache).
func (m *Model) buildTableCache() map[int]string {
	cache := map[int]string{}
	if m.doc == nil {
		return cache
	}
	lines := strings.Split(m.doc.Content, "\n")

	var block []int // line numbers of the current run of table rows
	flush := func() {
		if len(block) >= 2 {
			alignBlock(lines, block, cache)
		}
		block = nil
	}
	for i, line := range lines {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "|") {
			block = append(block, i+1)
			continue
		}
		flush()
	}
	flush()
	return cache
}

// alignBlock computes per-column widths across the block and writes each
// row's aligned form into the cache.
func alignBlock(lines []string, rows []int, cache map[int]string) {
	type parsed struct {
		lineNum int
		cells   []string
		sep     bool
	}
	var ps []parsed
	var widths []int
	for _, ln := range rows {
		raw := strings.TrimSpace(lines[ln-1])
		raw = strings.TrimPrefix(raw, "|")
		raw = strings.TrimSuffix(raw, "|")
		cells := strings.Split(raw, "|")
		sep := tableSepPattern.MatchString(lines[ln-1])
		for i, c := range cells {
			c = strings.TrimSpace(c)
			cells[i] = c
			w := lipgloss.Width(c)
			if sep {
				w = 3 // separators never widen a column
			}
			if i >= len(widths) {
				widths = append(widths, w)
			} else if w > widths[i] {
				widths[i] = w
			}
		}
		ps = append(ps, parsed{lineNum: ln, cells: cells, sep: sep})
	}
	for _, p := range ps {
		var b strings.Builder
		b.WriteString("|")
		for i, w := range widths {
			cell := ""
			if i < len(p.cells) {
				cell = p.cells[i]
			}
			if p.sep {
				b.WriteString(strings.Repeat("-", w+2))
			} else {
				pad := w - lipgloss.Width(cell)
				fmt.Fprintf(&b, " %s%s ", cell, strings.Repeat(" ", max(pad, 0)))
			}
			b.WriteString("|")
		}
		cache[p.lineNum] = b.String()
	}
}
