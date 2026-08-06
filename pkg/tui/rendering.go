package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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
func styleInlineSpans(s string) string {
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
		b.WriteString(syntaxGlyphStyle.Render(open))
		switch sp.kind {
		case spanCode:
			b.WriteString(codeSpanStyle.Render(content))
		case spanBold:
			b.WriteString(boldSpanStyle.Render(content))
		case spanItalic:
			b.WriteString(italicSpanStyle.Render(content))
		}
		b.WriteString(syntaxGlyphStyle.Render(close))
		last = sp.end
	}
	b.WriteString(s[last:])
	return b.String()
}

// styleLinePrefix colors blockquote > bars and list bullets (-/*/+/numbered)
// without changing any characters. Returns the styled prefix and the rest of
// the line (a blockquote's content may itself carry a list bullet).
func styleLinePrefix(line string) (string, string) {
	prefix := ""
	rest := line
	if m := quotePrefixPattern.FindStringSubmatch(rest); m != nil {
		prefix += m[1] + quoteBarStyle.Render(m[2]) + m[3]
		rest = rest[len(m[0]):]
	}
	if m := bulletPrefixPattern.FindStringSubmatch(rest); m != nil {
		prefix += m[1] + bulletStyle.Render(m[2]) + m[3]
		rest = rest[len(m[0]):]
	} else if m := numberedPrefixPattern.FindStringSubmatch(rest); m != nil {
		prefix += m[1] + bulletStyle.Render(m[2]) + m[3]
		rest = rest[len(m[0]):]
	}
	return prefix, rest
}

// styleMarkdownLine applies in-place syntax styling to a markdown line:
// headers colored whole-line, bold/italic/inline-code spans styled with their
// syntax glyphs dimmed (never removed), list bullets and blockquote bars
// colored. The ANSI-stripped result always equals the input — line widths
// and anchors are untouched.
func styleMarkdownLine(line string) string {
	// Headers - color them for better scannability
	if strings.HasPrefix(line, "# ") {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")). // Purple
			Render(line)
	}
	if strings.HasPrefix(line, "## ") {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33")). // Blue
			Render(line)
	}
	if strings.HasPrefix(line, "### ") {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")). // Cyan
			Render(line)
	}
	if strings.HasPrefix(line, "#### ") || strings.HasPrefix(line, "##### ") || strings.HasPrefix(line, "###### ") {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("45")). // Light cyan
			Render(line)
	}

	prefix, rest := styleLinePrefix(line)
	return prefix + styleInlineSpans(rest)
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
	return !t.Resolved && !(t.IsSuggestion && t.Accepted != nil)
}

// lineMarker builds the gutter marker for a line's threads: unresolved blocking
// stands out, plain unresolved shows a count, fully-resolved lines get a quiet check
func lineMarker(threads []*comment.Comment) string {
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
		return blockingMarkerStyle.Render(fmt.Sprintf("⛔%d", unresolved))
	case unresolved > 0:
		return commentMarkerStyle.Render(fmt.Sprintf("💬%d", unresolved))
	default:
		return resolvedMarkerStyle.Render("✓ ")
	}
}

// lineSummary builds the dimmed virtual-text summary for a commented line:
// first thread's author, thread count, open count (`· @rcliao ×2 1 open`),
// plus a NEW badge when any thread has activity newer than the last signoff
// (`since`). Returns "" for uncommented lines.
func lineSummary(threads []*comment.Comment, since time.Time) string {
	if len(threads) == 0 {
		return ""
	}
	open := 0
	for _, t := range threads {
		if isOpenThread(t) {
			open++
		}
	}
	summary := virtualTextStyle.Render(fmt.Sprintf("· @%s ×%d %d open", threads[0].Author, len(threads), open))
	if anyNewActivity(threads, since) {
		summary += " " + newBadgeStyle.Render("NEW")
	}
	return summary
}

// lineSummarySuffix returns the end-of-line summary (with leading space) for
// a line's threads, honoring the L toggle
func (m *Model) lineSummarySuffix(threads []*comment.Comment) string {
	if !m.showLineSummaries {
		return ""
	}
	summary := lineSummary(threads, lastSignoffTime(m.doc.Reviews))
	if summary == "" {
		return ""
	}
	return " " + summary
}

// renderDocument renders the document with line numbers and comment markers
func (m *Model) renderDocument() string {
	if m.doc == nil {
		return "No document loaded"
	}

	lines := strings.Split(m.doc.Content, "\n")
	var rendered strings.Builder

	// Calculate available width for text: viewport width - line number (4) - marker (3) - spacing (3)
	availableWidth := m.documentViewport.Width - 10
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Group root threads by line (replies share the root's line)
	commentsByLine := rootThreadsByLine(m.doc.Threads)

	for i, line := range lines {
		lineNum := i + 1
		lineNumStr := lineNumberStyle.Render(fmt.Sprintf("%d", lineNum))

		marker := lineMarker(commentsByLine[lineNum])

		// Apply markdown syntax highlighting
		styledLine := styleMarkdownLine(line)

		// Wrap long lines
		wrappedLines := strings.Split(wordwrap.String(styledLine, availableWidth), "\n")
		for j, wrappedLine := range wrappedLines {
			if j == 0 {
				// First line: show line number, marker, and virtual-text summary
				rendered.WriteString(fmt.Sprintf("%s %s %s%s\n", lineNumStr, marker, wrappedLine, m.lineSummarySuffix(commentsByLine[lineNum])))
			} else {
				// Continuation lines: indent with spaces
				rendered.WriteString(fmt.Sprintf("%s %s %s\n", strings.Repeat(" ", 4), "  ", wrappedLine))
			}
		}
	}

	return rendered.String()
}

// renderDocumentWithCursor renders the document with a cursor for line selection
func (m *Model) renderDocumentWithCursor() string {
	if m.doc == nil {
		return "No document loaded"
	}

	lines := strings.Split(m.doc.Content, "\n")
	var rendered strings.Builder

	// Calculate available width for text: viewport width - cursor (2) - line number (4) - marker (3) - spacing (3)
	availableWidth := m.documentViewport.Width - 12
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Group root threads by line
	commentsByLine := rootThreadsByLine(m.doc.Threads)

	for i, line := range lines {
		lineNum := i + 1
		lineNumStr := lineNumberStyle.Render(fmt.Sprintf("%d", lineNum))

		marker := lineMarker(commentsByLine[lineNum])

		// Highlight cursor line
		cursor := "  "
		isSelected := lineNum == m.selectedLine
		inRange := m.rangeActive && lineNum >= m.rangeStartLine && lineNum <= m.rangeEndLine

		// Syntax styling stays on for the cursor line (subtle bg composes with spans poorly,
		// so the cursor line keeps raw text but a gentle background; range keeps raw too)
		styledLine := line
		if !isSelected && !inRange {
			styledLine = styleMarkdownLine(line)
		}
		if isSelected {
			lineNumStr = cursorAccentStyle.Render(fmt.Sprintf("%d", lineNum))
		}

		// Wrap long lines
		wrappedLines := strings.Split(wordwrap.String(styledLine, availableWidth), "\n")
		for j, wrappedLine := range wrappedLines {
			if j == 0 {
				// First line: show cursor, line number and marker
				if isSelected {
					cursor = cursorAccentStyle.Render("▶")
					wrappedLine = cursorStyle.Render(wrappedLine)
				} else if inRange {
					cursor = rangeMarkerStyle.Render("│")
					wrappedLine = selectedLineStyle.Render(wrappedLine)
				}
				rendered.WriteString(fmt.Sprintf("%s %s %s %s%s\n", cursor, lineNumStr, marker, wrappedLine, m.lineSummarySuffix(commentsByLine[lineNum])))
			} else {
				// Continuation lines: indent with spaces
				displayCursor := "  "
				if isSelected {
					displayCursor = cursorAccentStyle.Render("▶ ")
					wrappedLine = cursorStyle.Render(wrappedLine)
				} else if inRange {
					displayCursor = rangeMarkerStyle.Render("│ ")
					wrappedLine = selectedLineStyle.Render(wrappedLine)
				}
				rendered.WriteString(fmt.Sprintf("%s %s %s %s\n", displayCursor, strings.Repeat(" ", 4), "  ", wrappedLine))
			}
		}
	}

	return rendered.String()
}

// getCommentTypeColor returns the color for a comment based on its type prefix
func getCommentTypeColor(text string) string {
	if len(text) < 3 {
		return ""
	}

	prefix := text[:3]
	switch prefix {
	case "[B]": // Blocker
		return "196" // Red
	case "[Q]": // Question
		return "220" // Yellow
	case "[S]": // Suggestion
		return "33"  // Blue
	case "[T]": // Technical
		return "13"  // Magenta
	case "[E]": // Editorial
		return "14"  // Cyan
	default:
		return ""
	}
}

// renderComments renders the comment panel
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
	width := m.commentViewport.Width - 4
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
func renderReplies(b *strings.Builder, replies []*comment.Comment, width, depth int, reviews []comment.ReviewRecord, currentRound *int) {
	indent := strings.Repeat("  ", depth+1)
	for _, r := range replies {
		if round := roundNumber(r.Timestamp, reviews); round != *currentRound {
			b.WriteString(roundSeparatorStyle.Render(fmt.Sprintf("%s── round %d ──", indent, round)))
			b.WriteString("\n")
			*currentRound = round
		}
		b.WriteString(replyMetaStyle.Render(fmt.Sprintf("%s└─ @%s · %s", indent, r.Author, r.Timestamp.Format("01-02 15:04"))))
		b.WriteString("\n")
		b.WriteString(indentWrap(r.Text, width, indent+"   "))
		b.WriteString("\n")
		renderReplies(b, r.Replies, width, depth+1, reviews, currentRound)
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
	rendered.WriteString(fmt.Sprintf("Comments (%d %s)\n\n", len(visible), statusText))

	for i := 0; i < len(visible); {
		line := visible[i].Line
		group := []*comment.Comment{}
		groupStart := i
		for i < len(visible) && visible[i].Line == line {
			group = append(group, visible[i])
			i++
		}

		// Group header: location + count badge
		header := fmt.Sprintf("📍 Line %d", line)
		if sp := group[0].SectionPath; sp != "" {
			parts := strings.Split(sp, " > ")
			header = fmt.Sprintf("📍 %s (Line %d)", parts[len(parts)-1], line)
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
		rendered.WriteString(groupHeaderStyle.Render(header))
		rendered.WriteString("\n")

		for gi, c := range group {
			idx := groupStart + gi
			style := lipgloss.NewStyle()
			if idx == m.selectedComment {
				style = selectedCommentStyle
			} else if typeColor := getCommentTypeColor(c.Text); typeColor != "" {
				style = style.Foreground(lipgloss.Color(typeColor))
			}

			resolvedMark := ""
			if c.Resolved {
				resolvedMark = "✓ "
			}

			// NEW badge: this thread has activity newer than the last signoff
			newBadge := ""
			if threadHasNewActivity(c, since) {
				newBadge = " " + newBadgeStyle.Render("NEW")
			}

			if expanded {
				wrapWidth := m.sidebarWrapWidth()
				text := fmt.Sprintf("  %s@%s%s%s · %s\n%s",
					resolvedMark, c.Author, threadMarkers(c), newBadge,
					c.Timestamp.Format("2006-01-02 15:04"),
					indentWrap(c.Text, wrapWidth, "  "))
				rendered.WriteString(style.Render(text))
				rendered.WriteString("\n")
				rootRound := roundNumber(c.Timestamp, m.doc.Reviews)
				renderReplies(&rendered, c.Replies, wrapWidth, 1, m.doc.Reviews, &rootRound)
				continue
			}

			summary := c.Text
			if len(summary) > 46 {
				summary = summary[:46] + "…"
			}
			text := fmt.Sprintf("  %s@%s%s%s: %s", resolvedMark, c.Author, threadMarkers(c), newBadge, summary)
			rendered.WriteString(style.Render(text))
			rendered.WriteString("\n")
		}
		rendered.WriteString("\n")
	}

	return rendered.String()
}

// renderThread renders an expanded thread view
func (m *Model) renderThread() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	var rendered strings.Builder

	// Thread header with section context
	locationStr := fmt.Sprintf("Line %d", m.selectedThread.Line)
	icon := "💬"
	if m.selectedThread.SectionPath != "" {
		locationStr = fmt.Sprintf("%s (Line %d)", m.selectedThread.SectionPath, m.selectedThread.Line)
		icon = "📍"
	}

	rendered.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%s Thread at %s\n", icon, locationStr)))
	rendered.WriteString("\n")

	// Document context - show lines around the comment
	contextLines := m.getContextLines(m.selectedThread.Line, 2) // 2 lines before/after
	if len(contextLines) > 0 {
		contextStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(m.width - 8)

		// Calculate width for context text wrapping
		contextWidth := m.width - 22 // Account for borders, padding, line numbers, markers
		if contextWidth < 40 {
			contextWidth = 40
		}

		var contextText strings.Builder
		contextText.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Document Context:"))
		contextText.WriteString("\n\n")

		for _, cl := range contextLines {
			marker := " "
			lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			lineStyle := lipgloss.NewStyle()

			// Apply markdown syntax highlighting to context lines
			styledText := styleMarkdownLine(cl.Text)

			// Wrap the line text
			wrappedLines := strings.Split(wordwrap.String(styledText, contextWidth), "\n")

			for i, wrappedLine := range wrappedLines {
				if cl.LineNum == m.selectedThread.Line {
					if i == 0 {
						marker = lipgloss.NewStyle().
							Foreground(lipgloss.Color("170")).
							Bold(true).
							Render("►")
						lineNumStyle = lineNumStyle.Bold(true).Foreground(lipgloss.Color("170"))
						lineStyle = lineStyle.
							Background(lipgloss.Color("235")).
							Foreground(lipgloss.Color("255")).
							Bold(true)
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							marker,
							lineNumStyle.Render(fmt.Sprintf("%4d", cl.LineNum)),
							lineStyle.Render(wrappedLine)))
					} else {
						// Continuation lines for highlighted line
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							" ",
							strings.Repeat(" ", 4),
							lineStyle.Render(wrappedLine)))
					}
				} else {
					if i == 0 {
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							marker,
							lineNumStyle.Render(fmt.Sprintf("%4d", cl.LineNum)),
							wrappedLine))
					} else {
						// Continuation lines for non-highlighted line
						contextText.WriteString(fmt.Sprintf("%s %s │ %s\n",
							" ",
							strings.Repeat(" ", 4),
							wrappedLine))
					}
				}
			}
		}

		rendered.WriteString(contextStyle.Render(contextText.String()))
		rendered.WriteString("\n\n")
	}

	// Root comment
	rootStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1).
		Width(m.width - 8)

	// Wrap root comment text to fit within the box
	rootTextWidth := m.width - 16 // Account for border, padding, and margins
	if rootTextWidth < 40 {
		rootTextWidth = 40
	}
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
			BorderForeground(lipgloss.Color("3")).
			Padding(0, 1).
			Width(m.width - 8)

		suggestionText := fmt.Sprintf("Suggestion Type: multi-line\n")
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

		borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

		// Calculate available width for reply text: width - padding - border characters
		replyWidth := m.width - 12
		if replyWidth < 40 {
			replyWidth = 40
		}

		for _, reply := range m.selectedThread.Replies {
			// Reply header with styled border and author
			rendered.WriteString(borderStyle.Render("│ "))
			rendered.WriteString(authorStyle.Render(fmt.Sprintf("@%s · %s",
				reply.Author,
				reply.Timestamp.Format("2006-01-02 15:04"))))
			rendered.WriteString("\n")

			// Wrap and render reply text
			lines := strings.Split(reply.Text, "\n")
			for _, line := range lines {
				// Wrap each line if it's too long
				wrappedLines := strings.Split(wordwrap.String(line, replyWidth), "\n")
				for _, wrappedLine := range wrappedLines {
					rendered.WriteString(borderStyle.Render("│ "))
					rendered.WriteString(wrappedLine)
					rendered.WriteString("\n")
				}
			}
			rendered.WriteString("\n")
		}
	} else {
		rendered.WriteString(helpStyle.Render("No replies yet\n\nPress 'r' to add a reply"))
	}

	return rendered.String()
}

// ContextLine represents a line with its line number for context display
type ContextLine struct {
	LineNum int
	Text    string
}

// getContextLines extracts lines around a specific line number for context
func (m *Model) getContextLines(lineNum int, contextSize int) []ContextLine {
	if m.doc == nil {
		return nil
	}

	lines := strings.Split(m.doc.Content, "\n")
	var result []ContextLine

	// Calculate range
	start := lineNum - contextSize - 1 // -1 for 0-based indexing
	if start < 0 {
		start = 0
	}

	end := lineNum + contextSize // inclusive
	if end > len(lines) {
		end = len(lines)
	}

	// Extract lines
	for i := start; i < end; i++ {
		result = append(result, ContextLine{
			LineNum: i + 1, // 1-based for display
			Text:    lines[i],
		})
	}

	return result
}

// getSectionContext returns section-aware context for a line
func (m *Model) getSectionContext(lineNum int) string {
	if m.doc == nil || m.documentSections == nil {
		return ""
	}

	var contextText strings.Builder
	lines := strings.Split(m.doc.Content, "\n")

	// Get section for this line
	section := m.getSectionAtLine(lineNum)
	if section == nil {
		// No section found, fall back to simple line context
		return ""
	}

	// Styles
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)
	
	headingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)
	
	lineNumStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255")).
		Bold(true)
	
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242"))

	// Show section path
	sectionPath := m.getSectionPath(section)
	contextText.WriteString(sectionStyle.Render("📍 Section: "))
	contextText.WriteString(sectionPath)
	contextText.WriteString("\n\n")

	// Show the heading line
	if section.StartLine > 0 && section.StartLine <= len(lines) {
		headingLine := lines[section.StartLine-1]
		contextText.WriteString(headingStyle.Render(headingLine))
		contextText.WriteString("\n")
		contextText.WriteString(dimStyle.Render(fmt.Sprintf("(lines %d-%d)", section.StartLine, section.EndLine)))
		contextText.WriteString("\n\n")
	}

	// Show context around target line within section
	// Show a few lines before and after the target, but stay within section bounds
	contextSize := 2
	start := lineNum - contextSize
	if start < section.StartLine {
		start = section.StartLine
	}
	end := lineNum + contextSize
	if end > section.EndLine {
		end = section.EndLine
	}

	// Skip heading line if we're showing section content
	if start == section.StartLine && lineNum != section.StartLine {
		start++
	}

	// Show ellipsis if we're skipping lines after heading
	if start > section.StartLine+1 {
		contextText.WriteString(dimStyle.Render("    ...\n"))
	}

	for i := start; i <= end && i <= len(lines); i++ {
		lineText := ""
		if i > 0 && i <= len(lines) {
			lineText = lines[i-1]
		}
		
		linePrefix := fmt.Sprintf("%4d │ ", i)
		if i == lineNum {
			// Highlight the target line
			contextText.WriteString(lineNumStyle.Bold(true).Render(linePrefix))
			contextText.WriteString(highlightStyle.Render(lineText))
		} else {
			contextText.WriteString(lineNumStyle.Render(linePrefix))
			contextText.WriteString(lineText)
		}
		contextText.WriteString("\n")
	}

	// Show ellipsis if there's more content in the section
	if end < section.EndLine {
		contextText.WriteString(dimStyle.Render("    ...\n"))
	}

	return contextText.String()
}
