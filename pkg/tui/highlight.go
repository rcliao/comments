package tui

// Fence-aware rendering (docs/design-markdown-render.md): inside a fenced
// code block, prose styling is suppressed and chroma highlights the
// code — whole block at a time (multi-line constructs color correctly), split
// back to lines, verified count-preserving. Each fence line splits at its
// comment marker: code goes to chroma, the trail keeps the existing citation
// styling, so peekable file:line evidence in DBML trails stays peekable with
// no ANSI/byte-offset collision.

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// fenceLine is the precomputed render state for one source line of a fence.
type fenceLine struct {
	delimiter bool   // the ``` line itself
	code      string // chroma-highlighted code part ("" for delimiter lines)
	trail     string // raw comment trail (styled at render for citations)
}

// buildFenceCache scans the document once per content change and returns the
// per-line fence state. Lines absent from the map are prose.
func (m *Model) buildFenceCache() map[int]fenceLine {
	cache := map[int]fenceLine{}
	if m.doc == nil {
		return cache
	}
	lines := strings.Split(m.doc.Content, "\n")

	type block struct {
		start, end int // 1-based, exclusive of delimiters
		lang       string
		codes      []string
		trails     []string
	}
	var blocks []*block
	var cur *block
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			if cur == nil {
				cur = &block{start: i + 2, lang: strings.TrimSpace(strings.TrimPrefix(t, "```"))}
				cache[i+1] = fenceLine{delimiter: true}
			} else {
				cur.end = i + 1
				cache[i+1] = fenceLine{delimiter: true}
				blocks = append(blocks, cur)
				cur = nil
			}
			continue
		}
		if cur != nil {
			code, trail := splitCommentTrail(line)
			cur.codes = append(cur.codes, code)
			cur.trails = append(cur.trails, trail)
		}
	}

	for _, b := range blocks {
		highlighted := highlightBlock(strings.Join(b.codes, "\n"), b.lang, m.styles.theme.ChromaStyle())
		hl := strings.Split(highlighted, "\n")
		for j := 0; j < b.end-b.start; j++ {
			code := b.codes[j]
			if j < len(hl) && len(hl) == len(b.codes) {
				code = hl[j]
			}
			cache[b.start+j] = fenceLine{code: code, trail: b.trails[j]}
		}
	}
	return cache
}

// splitCommentTrail cuts a fence line at its comment marker (// or #), so the
// trail can keep citation styling. The marker stays with the trail.
func splitCommentTrail(line string) (string, string) {
	idx := -1
	for _, marker := range []string{"//", "#"} {
		if i := strings.Index(line, marker); i >= 0 && (idx == -1 || i < idx) {
			idx = i
		}
	}
	if idx < 0 {
		return line, ""
	}
	return line[:idx], line[idx:]
}

// highlightBlock runs chroma over a whole block. Any failure returns the
// source unchanged — a mis-highlighted doc must never lose content.
func highlightBlock(code, lang, styleName string) string {
	if strings.TrimSpace(code) == "" {
		return code
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return code
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}
	var b strings.Builder
	if err := formatter.Format(&b, style, it); err != nil {
		return code
	}
	out := strings.TrimSuffix(b.String(), "\n")
	if strings.Count(out, "\n") != strings.Count(code, "\n") {
		return code // line-preservation is the contract; bail if broken
	}
	return out
}
