package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"
	"github.com/rcliao/comments/pkg/comment"
)

// Inline span and line-prefix patterns for in-place markdown styling.
// All delimiters are ASCII, so regexp byte offsets index the line directly.
var (
	codeSpanPattern    = regexp.MustCompile("`[^`]+`")
	boldStarPattern    = regexp.MustCompile(`\*\*[^*]+\*\*`)
	boldUnderPattern   = regexp.MustCompile(`__[^_]+__`)
	italicStarPattern  = regexp.MustCompile(`\*[^*\s][^*]*\*`)
	italicUnderPattern = regexp.MustCompile(`_[^_\s][^_]*_`)

	quotePrefixPattern    = regexp.MustCompile(`^(\s*)(>+)(\s?)`)
	bulletPrefixPattern   = regexp.MustCompile(`^(\s*)([-*+])(\s)`)
	numberedPrefixPattern = regexp.MustCompile(`^(\s*)(\d+[.)])(\s)`)
)

// spanKind classifies an inline markdown span
type spanKind int

const (
	spanCode spanKind = iota
	spanBold
	spanItalic
)

// inlineSpan is one styled region of a line: [start,end) byte offsets into
// the raw line, with markerLen syntax glyphs on each side of the content
type inlineSpan struct {
	start, end, markerLen int
	kind                  spanKind
}

// findInlineSpans locates non-overlapping code/bold/italic spans in priority
// order (code wins over bold wins over italic), returned sorted by start.
// Claimed bytes are masked out before lower-priority patterns scan, so a
// consumed delimiter (e.g. bold's **) can never seed a later match.
func findInlineSpans(line string) []inlineSpan {
	spans := []inlineSpan{}
	taken := make([]bool, len(line))
	masked := []byte(line)
	add := func(re *regexp.Regexp, markerLen int, kind spanKind) {
		for _, loc := range re.FindAllIndex(masked, -1) {
			overlaps := false
			for i := loc[0]; i < loc[1]; i++ {
				if taken[i] {
					overlaps = true
					break
				}
			}
			if overlaps {
				continue
			}
			for i := loc[0]; i < loc[1]; i++ {
				taken[i] = true
				masked[i] = 0
			}
			spans = append(spans, inlineSpan{start: loc[0], end: loc[1], markerLen: markerLen, kind: kind})
		}
	}
	add(codeSpanPattern, 1, spanCode)
	add(boldStarPattern, 2, spanBold)
	add(boldUnderPattern, 2, spanBold)
	add(italicStarPattern, 1, spanItalic)
	add(italicUnderPattern, 1, spanItalic)
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	return spans
}

// styleInlineSpans renders bold/italic/inline-code spans in place: content
// styled, syntax glyphs dimmed but kept, so the ANSI-stripped text is
// byte-identical to the input (no reflow, anchors untouched)
func (st *styleSet) styleInlineSpans(s string) string {
	spans := findInlineSpans(s)
	if len(spans) == 0 {
		return s
	}
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		b.WriteString(s[last:sp.start])
		open := s[sp.start : sp.start+sp.markerLen]
		content := s[sp.start+sp.markerLen : sp.end-sp.markerLen]
		close := s[sp.end-sp.markerLen : sp.end]
		b.WriteString(st.syntaxGlyph.Render(open))
		switch sp.kind {
		case spanCode:
			b.WriteString(st.codeSpan.Render(content))
		case spanBold:
			b.WriteString(st.boldSpan.Render(content))
		case spanItalic:
			b.WriteString(st.italicSpan.Render(content))
		}
		b.WriteString(st.syntaxGlyph.Render(close))
		last = sp.end
	}
	b.WriteString(s[last:])
	return b.String()
}

// styleLinePrefix colors blockquote > bars and list bullets (-/*/+/numbered)
// without changing any characters. Returns the styled prefix and the rest of
// the line (a blockquote's content may itself carry a list bullet).
func (st *styleSet) styleLinePrefix(line string) (string, string) {
	prefix := ""
	rest := line
	if m := quotePrefixPattern.FindStringSubmatch(rest); m != nil {
		prefix += m[1] + st.quoteBar.Render(m[2]) + m[3]
		rest = rest[len(m[0]):]
	}
	if m := bulletPrefixPattern.FindStringSubmatch(rest); m != nil {
		prefix += m[1] + st.bullet.Render(m[2]) + m[3]
		rest = rest[len(m[0]):]
	} else if m := numberedPrefixPattern.FindStringSubmatch(rest); m != nil {
		prefix += m[1] + st.bullet.Render(m[2]) + m[3]
		rest = rest[len(m[0]):]
	}
	return prefix, rest
}

// styleMarkdownLine applies in-place syntax styling to a markdown line:
// headers colored whole-line, bold/italic/inline-code spans styled with their
// syntax glyphs dimmed (never removed), list bullets and blockquote bars
// colored. The ANSI-stripped result always equals the input — line widths
// and anchors are untouched.
func (st *styleSet) styleMarkdownLine(line string) string {
	// Headers - color them for better scannability
	if strings.HasPrefix(line, "# ") {
		return st.heading1.Render(line)
	}
	if strings.HasPrefix(line, "## ") {
		return st.heading2.Render(line)
	}
	if strings.HasPrefix(line, "### ") {
		return st.heading3.Render(line)
	}
	if strings.HasPrefix(line, "#### ") || strings.HasPrefix(line, "##### ") || strings.HasPrefix(line, "###### ") {
		return st.heading4.Render(line)
	}

	prefix, rest := st.styleLinePrefix(line)
	return prefix + st.styleInlineSpans(rest)
}

// styleDocLine styles one document line: markdown syntax styling plus link
// styling for resolvable file references on the line. Segments between
// references still get inline-span styling; a span that would cross a
// reference boundary is simply left raw (characters are always preserved —
// the ANSI-stripped result equals the input). Headings keep whole-line
// styling and skip reference styling.
func (m *Model) styleDocLine(line string, lineNum int) string {
	refs := m.refsByLine[lineNum]
	styled := make([]resolvedRef, 0, len(refs))
	for _, r := range refs {
		if r.resolved && r.StartCol >= 0 && r.EndCol <= len(line) {
			styled = append(styled, r)
		}
	}
	if len(styled) == 0 || strings.HasPrefix(line, "#") {
		return m.styles.styleMarkdownLine(line)
	}

	var b strings.Builder
	last := 0
	for i, r := range styled {
		if r.StartCol < last {
			continue // overlapping parse; keep the earlier reference
		}
		seg := line[last:r.StartCol]
		if i == 0 {
			prefix, rest := m.styles.styleLinePrefix(seg)
			b.WriteString(prefix + m.styles.styleInlineSpans(rest))
		} else {
			b.WriteString(m.styles.styleInlineSpans(seg))
		}
		b.WriteString(m.styles.refLink.Render(line[r.StartCol:r.EndCol]))
		last = r.EndCol
	}
	b.WriteString(m.styles.styleInlineSpans(line[last:]))
	return b.String()
}

// rootThreadsByLine groups root threads (not replies) by their anchor line
func rootThreadsByLine(threads []*comment.Comment) map[int][]*comment.Comment {
	byLine := map[int][]*comment.Comment{}
	for _, t := range threads {
		byLine[t.Line] = append(byLine[t.Line], t)
	}
	return byLine
}

// isOpenThread reports whether a root thread still needs attention:
// unresolved and, for suggestions, not yet decided
func isOpenThread(t *comment.Comment) bool {
	return !t.Resolved && (!t.IsSuggestion || t.Accepted == nil)
}

// lineMarker builds the gutter marker for a line's threads: unresolved blocking
// stands out, plain unresolved shows a count, fully-resolved lines get a quiet check
func (st *styleSet) lineMarker(threads []*comment.Comment) string {
	if len(threads) == 0 {
		return "  "
	}
	unresolved, blocking := 0, false
	for _, t := range threads {
		if isOpenThread(t) {
			unresolved++
			if t.Blocking {
				blocking = true
			}
		}
	}
	switch {
	case blocking:
		return st.blockingMarker.Render(fmt.Sprintf("⛔%d", unresolved))
	case unresolved > 0:
		return st.commentMarker.Render(fmt.Sprintf("💬%d", unresolved))
	default:
		return st.resolvedMarker.Render("✓ ")
	}
}

// lineSummary builds the dimmed virtual-text summary for a commented line:
// first thread's author, thread count, open count (`· @rcliao ×2 1 open`),
// plus a NEW badge when any thread has activity newer than the last signoff
// (`since`). Returns "" for uncommented lines.
func (st *styleSet) lineSummary(threads []*comment.Comment, since time.Time) string {
	if len(threads) == 0 {
		return ""
	}
	open := 0
	for _, t := range threads {
		if isOpenThread(t) {
			open++
		}
	}
	summary := st.virtualText.Render(fmt.Sprintf("· @%s ×%d %d open", threads[0].Author, len(threads), open))
	if anyNewActivity(threads, since) {
		summary += " " + st.newBadge.Render("NEW")
	}
	return summary
}

// threadTypeIcon maps a thread to its sidebar icon by comment type:
// edit suggestions and typed comments read at a glance; untyped fall back
// to the plain bubble.
func threadTypeIcon(c *comment.Comment) string {
	if c.IsSuggestion {
		return "\U0001F4DD" // 📝 edit suggestion
	}
	switch c.Type {
	case "Q":
		return "\u2753" // ❓ question
	case "S":
		return "\U0001F4A1" // 💡 suggestion
	case "B":
		return "\U0001F41B" // 🐛 bug
	case "T":
		return "\U0001F4CB" // 📋 todo
	case "E":
		return "\u2728" // ✨ enhancement
	}
	return "\U0001F4AC" // 💬 untyped
}

// groupIcon is the icon for a sidebar line-group: the first open thread's
// type icon (open threads outrank resolved ones), else the first thread's.
func groupIcon(group []*comment.Comment) string {
	for _, c := range group {
		if isOpenThread(c) {
			return threadTypeIcon(c)
		}
	}
	return threadTypeIcon(group[0])
}

// lineSummarySuffix returns the end-of-line summary (with leading space) for
// a line's threads, honoring the L toggle
func (m *Model) lineSummarySuffix(threads []*comment.Comment) string {
	if !m.showLineSummaries {
		return ""
	}
	summary := m.styles.lineSummary(threads, lastSignoffTime(m.doc.Reviews))
	if summary == "" {
		return ""
	}
	return " " + summary
}

// docWrapWidth is the text wrap width for the document pane, shared by both
// document renders and the scroll math. One constant for both layouts: the
// widest gutter is the cursor view's — cursor (2) + space + line number (4) +
// space + marker (up to 3) + space = 12. Browse mode's gutter is 2 narrower,
// but sharing the cursor width keeps wrap points (and therefore scroll rows)
// identical across modes, so entering/leaving line-select never reflows the
// document under a saved scroll offset.
func (m *Model) docWrapWidth() int {
	return max(m.documentViewport.Width()-m.gutterWidth(), 40)
}

// gutterWidth is the fixed left-gutter width shared by both document renders
// and the scroll math: cursor (2) + space + marker (4) + line number (4) +
// space = 12, shrinking by the number column when line numbers are hidden.
func (m *Model) gutterWidth() int {
	if m.hideLineNumbers {
		return 8
	}
	return 12
}

// renderDocument renders the document pane without a cursor (browse mode)
func (m *Model) renderDocument() string {
	return m.renderDocumentView(false)
}

// renderDocumentWithCursor renders the document pane with the line-select
// cursor and any active range highlight
func (m *Model) renderDocumentWithCursor() string {
	return m.renderDocumentView(true)
}

// renderDocumentView renders the document with line numbers, gutter markers,
// and virtual-text summaries. With withCursor it adds the line-select cursor
// column and range highlighting. Both layouts wrap at docWrapWidth.
func (m *Model) renderDocumentView(withCursor bool) string {
	if m.doc == nil {
		return "No document loaded"
	}

	lines := strings.Split(m.doc.Content, "\n")
	var rendered strings.Builder

	availableWidth := m.docWrapWidth()

	// Group root threads by line (replies share the root's line)
	commentsByLine := rootThreadsByLine(m.doc.Threads)

	// Sidebar→doc sync: without a cursor, highlight the line the sidebar (or
	// the open thread panel) is focused on, so navigating threads visibly
	// moves the reader's eye in the document pane
	docFocus := 0
	if !withCursor {
		if m.mode == ModeThreadView && m.selectedThread != nil {
			docFocus = m.selectedThread.Line
		} else {
			docFocus = m.focusLine()
		}
	}

	for i, line := range lines {
		lineNum := i + 1
		lineNumStr := m.styles.lineNumber.Render(fmt.Sprintf("%d", lineNum))

		marker := m.styles.lineMarker(commentsByLine[lineNum])

		isSelected := withCursor && lineNum == m.selectedLine
		isFocused := !withCursor && lineNum == docFocus
		inRange := withCursor && m.rangeActive && lineNum >= m.rangeStartLine && lineNum <= m.rangeEndLine

		// Syntax styling stays off on the cursor/focus line (subtle bg
		// composes with spans poorly, so these lines keep raw text but a
		// gentle background; range keeps raw too)
		styledLine := line
		if !isSelected && !isFocused && !inRange {
			styledLine = m.styleDocLine(line, lineNum)
		}
		if isSelected || isFocused {
			lineNumStr = m.styles.cursorLineNum.Render(fmt.Sprintf("%d", lineNum))
		}

		// Wrap long lines
		wrappedLines := strings.Split(wordwrap.String(styledLine, availableWidth), "\n")
		for j, wrappedLine := range wrappedLines {
			if isSelected || isFocused {
				wrappedLine = m.styles.cursor.Render(wrappedLine)
			} else if inRange {
				wrappedLine = m.styles.selectedLine.Render(wrappedLine)
			}

			// Marker column sits LEFT of the line number (fixed 4 cells)
			// so text alignment never shifts on commented lines. The number
			// column disappears entirely when hidden (# toggles).
			markerCell := marker + strings.Repeat(" ", max(0, 4-lipgloss.Width(marker)))
			numCell := lineNumStr + " "
			if m.hideLineNumbers {
				numCell = ""
			}
			contPad := strings.Repeat(" ", m.gutterWidth()-4)

			if !withCursor {
				if j == 0 {
					// First line: marker, line number, then text + summary
					fmt.Fprintf(&rendered, "%s%s%s%s\n", markerCell, numCell, wrappedLine, m.lineSummarySuffix(commentsByLine[lineNum]))
				} else {
					// Continuation lines: indent with spaces
					fmt.Fprintf(&rendered, "%s%s\n", contPad, wrappedLine)
				}
				continue
			}

			if j == 0 {
				// First line: cursor, marker, line number, text
				cursor := "  "
				if isSelected {
					cursor = m.styles.cursorAccent.Render("▶ ")
				} else if inRange {
					cursor = m.styles.rangeMarker.Render("│")
				}
				fmt.Fprintf(&rendered, "%s %s%s%s%s\n", cursor, markerCell, numCell, wrappedLine, m.lineSummarySuffix(commentsByLine[lineNum]))
			} else {
				// Continuation lines: indent with spaces
				displayCursor := "  "
				if isSelected {
					displayCursor = m.styles.cursorAccent.Render("▶ ")
				} else if inRange {
					displayCursor = m.styles.rangeMarker.Render("│ ")
				}
				fmt.Fprintf(&rendered, "%s %s%s\n", displayCursor, contPad, wrappedLine)
			}
		}
	}

	return rendered.String()
}

// truncate shortens s to at most max runes, cutting on a rune boundary and
// appending ellipsis when it doesn't fit (the ellipsis counts toward max).
// Byte slicing (s[:n]) splits multi-byte runes (CJK, emoji) into mojibake —
// always use this helper for display truncation.
func truncate(s string, maxRunes int, ellipsis string) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	keep := max(maxRunes-len([]rune(ellipsis)), 0)
	return string(runes[:keep]) + ellipsis
}

// getCommentTypeColor returns the color for a comment based on its type prefix
func (st *styleSet) getCommentTypeColor(text string) string {
	switch {
	case strings.HasPrefix(text, "[B]"): // Blocker
		return string(st.theme.TypeB)
	case strings.HasPrefix(text, "[Q]"): // Question
		return string(st.theme.TypeQ)
	case strings.HasPrefix(text, "[S]"): // Suggestion
		return string(st.theme.TypeS)
	case strings.HasPrefix(text, "[T]"): // Technical
		return string(st.theme.TypeT)
	case strings.HasPrefix(text, "[E]"): // Editorial
		return string(st.theme.TypeE)
	default:
		return ""
	}
}

// visibleComments returns the sidebar's thread list, ordered by document line
// so the sidebar reads top-to-bottom with the document (focus-follows-cursor, G3)
func (m *Model) visibleComments() []*comment.Comment {
	visible := comment.GetVisibleComments(m.doc.Threads, m.showResolved)
	sort.SliceStable(visible, func(i, j int) bool { return visible[i].Line < visible[j].Line })
	return visible
}

// focusLine is the document line the sidebar should expand around:
// the cursor in line-select mode, otherwise the selected comment's line
func (m *Model) focusLine() int {
	if m.mode == ModeLineSelect {
		return m.selectedLine
	}
	visible := m.visibleComments()
	if len(visible) > 0 && m.selectedComment >= 0 && m.selectedComment < len(visible) {
		return visible[m.selectedComment].Line
	}
	return 0
}

// threadMarkers builds the status/type indicator string for a thread
func threadMarkers(c *comment.Comment) string {
	markers := ""
	if c.Blocking && !c.Resolved {
		markers += " [BLOCKING]"
	}
	if c.IsSuggestion {
		if c.IsPending() {
			markers += " [📝 SUGGESTION]"
		} else if c.Accepted != nil && *c.Accepted {
			markers += " [✓ ACCEPTED]"
		} else if c.Accepted != nil && !*c.Accepted {
			markers += " [✗ REJECTED]"
		}
	}
	switch c.AnchorConfidence {
	case comment.ConfidenceFuzzy:
		markers += " ~fuzzy"
	case comment.ConfidenceSectionLevel:
		markers += " §section"
	}
	return markers
}

// sidebarWrapWidth is the text width for expanded threads in the comment sidebar
func (m *Model) sidebarWrapWidth() int {
	width := m.commentViewport.Width() - 4
	if width < 30 {
		width = 40 // uninitialized viewport (tests) or very narrow terminal
	}
	return width
}

// indentWrap word-wraps text and prefixes every line with indent
func indentWrap(text string, width int, indent string) string {
	wrapped := strings.Split(wordwrap.String(text, width-len(indent)), "\n")
	for i, line := range wrapped {
		wrapped[i] = indent + line
	}
	return strings.Join(wrapped, "\n")
}

// renderReplies renders a thread's replies nested under the root (expanded
// view), inserting a dimmed `── round N ──` separator whenever a reply falls
// in a later review round than everything rendered before it (the thread
// timeline: rounds are partitioned by signoff timestamps, see roundNumber).
// currentRound carries the round of the previously rendered comment through
// the recursion, starting at the root's round.
func (st *styleSet) renderReplies(b *strings.Builder, replies []*comment.Comment, width, depth int, reviews []comment.ReviewRecord, currentRound *int) {
	indent := strings.Repeat("  ", depth+1)
	for _, r := range replies {
		if round := roundNumber(r.Timestamp, reviews); round != *currentRound {
			b.WriteString(st.roundSeparator.Render(fmt.Sprintf("%s── round %d ──", indent, round)))
			b.WriteString("\n")
			*currentRound = round
		}
		b.WriteString(st.replyMeta.Render(fmt.Sprintf("%s└─ @%s · %s", indent, r.Author, r.Timestamp.Format("01-02 15:04"))))
		b.WriteString("\n")
		b.WriteString(indentWrap(r.Text, width, indent+"   "))
		b.WriteString("\n")
		st.renderReplies(b, r.Replies, width, depth+1, reviews, currentRound)
	}
}

// renderComments renders the sidebar grouped by line: the focused line's group
// auto-expands for glanceable review; other groups collapse to one line per thread
func (m *Model) renderComments() string {
	if m.doc == nil {
		return "No comments"
	}
	if m.sidebarDensity == densityHidden {
		return ""
	}

	visible := m.visibleComments()
	if len(visible) == 0 {
		if m.showResolved {
			return "No comments"
		}
		return "No unresolved comments\n\nPress R to show resolved"
	}

	statusText := "unresolved"
	if m.showResolved {
		statusText = "all"
	}
	focus := m.focusLine()
	since := lastSignoffTime(m.doc.Reviews)

	var rendered strings.Builder
	fmt.Fprintf(&rendered, "Comments (%d %s)\n\n", len(visible), statusText)

	for i := 0; i < len(visible); {
		line := visible[i].Line
		group := []*comment.Comment{}
		groupStart := i
		for i < len(visible) && visible[i].Line == line {
			group = append(group, visible[i])
			i++
		}

		// Group header: location + count badge, icon keyed to the group's
		// first open thread's comment type
		header := fmt.Sprintf("%s Line %d", groupIcon(group), line)
		if sp := group[0].SectionPath; sp != "" {
			parts := strings.Split(sp, " > ")
			header = fmt.Sprintf("%s %s (Line %d)", groupIcon(group), parts[len(parts)-1], line)
		}
		if len(group) > 1 {
			header += fmt.Sprintf(" · %d threads", len(group))
		}
		// Condensed density forces every group to one line per thread
		expanded := line == focus && m.sidebarDensity == densityFull
		if expanded {
			header = "▼ " + header
		} else {
			header = "▸ " + header
		}
		rendered.WriteString(m.styles.groupHeader.Render(header))
		rendered.WriteString("\n")

		for gi, c := range group {
			idx := groupStart + gi
			style := lipgloss.NewStyle()
			if idx == m.selectedComment {
				style = m.styles.selectedComment
			} else if typeColor := m.styles.getCommentTypeColor(c.Text); typeColor != "" {
				style = style.Foreground(lipgloss.Color(typeColor))
			}

			resolvedMark := ""
			if c.Resolved {
				resolvedMark = "✓ "
			}

			// NEW badge: this thread has activity newer than the last signoff
			newBadge := ""
			if threadHasNewActivity(c, since) {
				newBadge = " " + m.styles.newBadge.Render("NEW")
			}

			if expanded {
				wrapWidth := m.sidebarWrapWidth()
				text := fmt.Sprintf("  %s@%s%s%s · %s · %s\n%s",
					resolvedMark, c.Author, threadMarkers(c), newBadge,
					c.Timestamp.Format("2006-01-02 15:04"), c.ID,
					indentWrap(c.Text, wrapWidth, "  "))
				rendered.WriteString(style.Render(text))
				rendered.WriteString("\n")
				rootRound := roundNumber(c.Timestamp, m.doc.Reviews)
				m.styles.renderReplies(&rendered, c.Replies, wrapWidth, 1, m.doc.Reviews, &rootRound)
				continue
			}

			// ID first: a scannable column for matching agent-referenced
			// thread IDs ("answer c7q39") against the sidebar
			summary := truncate(c.Text, 41, "…")
			text := fmt.Sprintf("  %s%s @%s%s%s: %s", resolvedMark, c.ID, c.Author, threadMarkers(c), newBadge, summary)
			rendered.WriteString(style.Render(text))
			rendered.WriteString("\n")
		}
		rendered.WriteString("\n")
	}

	return rendered.String()
}

// renderThread renders the expanded thread at the full terminal width (the
// legacy full-screen thread layout, still used by unlaid-out unit tests).
func (m *Model) renderThread() string { return m.renderThreadWidth(m.width) }

// renderThreadWidth renders the expanded thread as if the screen were `width`
// columns wide — the same content at any width, so the thread panel
// (keys_threadpanel.go) can reuse this renderer inside a narrower pane. The
// widest inner box renders at width-8 content + 2 border columns, i.e.
// width-6 total (see threadRenderPad). The thread's location header lives in
// the panel chrome (renderThreadPanelBox), and the live document beside the
// panel is the context — neither is repeated in here.
func (m *Model) renderThreadWidth(width int) string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	theme := m.styles.theme
	var rendered strings.Builder

	// Root comment
	rootStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.GroupHeader.Color()).
		Padding(1).
		Width(width - 8)

	// Wrap root comment text to fit within the box
	// Account for border, padding, and margins
	rootTextWidth := max(width-16, 20)
	wrappedRootText := wordwrap.String(m.selectedThread.Text, rootTextWidth)

	rootText := fmt.Sprintf("@%s · %s\n\n%s",
		m.selectedThread.Author,
		m.selectedThread.Timestamp.Format("2006-01-02 15:04"),
		wrappedRootText,
	)

	if m.selectedThread.Resolved {
		rootText = "✓ RESOLVED\n\n" + rootText
	}

	// Add suggestion indicator if root comment is a suggestion
	if m.selectedThread.IsSuggestion {
		var stateText string
		if m.selectedThread.IsPending() {
			stateText = "📝 PENDING SUGGESTION"
			if queued := m.queuedLabel(m.selectedThread.ID); queued != "" {
				stateText = "⏳ " + queued + " — applies at verdict (q)"
			}
		} else if m.selectedThread.Accepted != nil && *m.selectedThread.Accepted {
			stateText = "✓ ACCEPTED SUGGESTION"
		} else if m.selectedThread.Accepted != nil && !*m.selectedThread.Accepted {
			stateText = "✗ REJECTED SUGGESTION"
		}
		rootText = stateText + "\n\n" + rootText
	}

	rendered.WriteString(rootStyle.Render(rootText))
	rendered.WriteString("\n\n")

	// Show suggestion details if this is a suggestion (v2.0 simplified)
	if m.selectedThread.IsSuggestion {
		suggestionStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Warning.Color()).
			Padding(0, 1).
			Width(width - 8)

		suggestionText := "Suggestion Type: multi-line\n"
		suggestionText += fmt.Sprintf("Lines: %d-%d\n", m.selectedThread.StartLine, m.selectedThread.EndLine)

		if m.selectedThread.OriginalText != "" {
			suggestionText += fmt.Sprintf("\nOriginal:\n  %s\n", m.selectedThread.OriginalText)
		}
		if m.selectedThread.ProposedText != "" {
			suggestionText += fmt.Sprintf("\nProposed:\n  %s\n", m.selectedThread.ProposedText)
		}

		suggestionText += "\n'a' queues accept · 'x' queues reject — all queued decisions apply at the verdict (q)"

		rendered.WriteString(suggestionStyle.Render(suggestionText))
		rendered.WriteString("\n\n")
	}

	// Replies
	if len(m.selectedThread.Replies) > 0 {
		rendered.WriteString(lipgloss.NewStyle().Bold(true).Render(
			fmt.Sprintf("Replies (%d):", len(m.selectedThread.Replies))))
		rendered.WriteString("\n\n")

		borderStyle := lipgloss.NewStyle().Foreground(theme.Border.Color())
		authorStyle := lipgloss.NewStyle().Foreground(theme.MetaText.Color())

		// Calculate available width for reply text: width - padding - border characters
		replyWidth := max(width-12, 20)

		for _, reply := range m.selectedThread.Replies {
			// Reply header with styled border and author
			rendered.WriteString(borderStyle.Render("│ "))
			rendered.WriteString(authorStyle.Render(fmt.Sprintf("@%s · %s",
				reply.Author,
				reply.Timestamp.Format("2006-01-02 15:04"))))
			rendered.WriteString("\n")

			// Wrap and render reply text
			for line := range strings.SplitSeq(reply.Text, "\n") {
				// Wrap each line if it's too long
				for wrappedLine := range strings.SplitSeq(wordwrap.String(line, replyWidth), "\n") {
					rendered.WriteString(borderStyle.Render("│ "))
					rendered.WriteString(wrappedLine)
					rendered.WriteString("\n")
				}
			}
			rendered.WriteString("\n")
		}
	} else {
		rendered.WriteString(m.styles.help.Render("No replies yet\n\nPress 'r' to add a reply"))
	}

	return rendered.String()
}
