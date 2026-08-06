package tui

// Line-select mode and its offshoots: cursor movement over document lines,
// choose-target (section vs line), suggestion-type choice, and range
// selection for multi-line suggestions. Also home of the cursor scroll math.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/markdown"
)

// handleLineSelectKeys handles keys in line select mode
func (m Model) handleLineSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	lines := strings.Split(m.doc.Content, "\n")
	totalLines := len(lines)

	switch msg.String() {
	case "esc":
		// Cancel line selection and reset viewport
		m.mode = ModeBrowse

		// Reset the viewport to fix any scroll offset issues
		m.documentViewport = viewport.New(m.docPaneWidth(), m.height-2)
		m.documentViewport.YOffset = 0
		m.documentViewport.SetContent(m.renderDocument())
		m.documentViewport.YOffset = 0
		return m, nil

	case "?":
		m.helpReturnMode = m.mode
		m.mode = ModeHelp
		return m, nil

	case "S":
		m.cycleSidebarDensity()
		return m, nil

	case "L":
		m.toggleLineSummaries()
		return m, nil

	case "t":
		m.openTOC()
		return m, nil

	case "j", "down":
		// Move cursor down
		if m.selectedLine < totalLines {
			m.selectedLine++
			m.refreshCursorView()
		}
		return m, nil

	case "k", "up":
		// Move cursor up
		if m.selectedLine > 1 {
			m.selectedLine--
			m.refreshCursorView()
		}
		return m, nil

	case "ctrl+d":
		// Page down (half page)
		pageSize := m.documentViewport.Height / 2
		m.selectedLine = min(m.selectedLine+pageSize, totalLines)
		m.refreshCursorView()
		return m, nil

	case "ctrl+u":
		// Page up (half page)
		pageSize := m.documentViewport.Height / 2
		m.selectedLine = max(m.selectedLine-pageSize, 1)
		m.refreshCursorView()
		return m, nil

	case "g":
		// Go to first line
		m.selectedLine = 1
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.documentViewport.GotoTop()
		return m, nil

	case "G":
		// Go to last line
		m.selectedLine = totalLines
		m.refreshCursorView()
		return m, nil

	case "c", "enter":
		// Check if on a heading line
		if m.isHeadingLine(m.selectedLine) {
			// On a heading - let user choose section vs line
			m.mode = ModeChooseTarget
			return m, nil
		}
		// Regular line - go directly to add comment
		m.targetIsSection = false
		m.mode = ModeAddComment
		m.commentInput.Reset()
		m.commentInput.Focus()
		return m, textarea.Blink

	case "q":
		m.verdictReturnMode = ModeLineSelect
		m.mode = ModeVerdict
		return m, nil

	case "]", "]c":
		// Jump to next line with an open thread
		for _, c := range m.visibleComments() {
			if !c.Resolved && c.Line > m.selectedLine {
				m.selectedLine = c.Line
				break
			}
		}
		m.refreshCursorView()
		return m, nil

	case "[":
		// Jump to previous line with an open thread
		for _, c := range slices.Backward(m.visibleComments()) {
			if !c.Resolved && c.Line < m.selectedLine {
				m.selectedLine = c.Line
				break
			}
		}
		m.refreshCursorView()
		return m, nil

	case "n":
		// Jump to next line whose threads have NEW activity since the last
		// signoff (the inbox motion, mirroring ]/[ above)
		since := lastSignoffTime(m.doc.Reviews)
		for _, c := range m.visibleComments() {
			if c.Line > m.selectedLine && threadHasNewActivity(c, since) {
				m.selectedLine = c.Line
				break
			}
		}
		m.refreshCursorView()
		return m, nil

	case "N":
		// Jump to previous line whose threads have NEW activity
		since := lastSignoffTime(m.doc.Reviews)
		for _, c := range slices.Backward(m.visibleComments()) {
			if c.Line < m.selectedLine && threadHasNewActivity(c, since) {
				m.selectedLine = c.Line
				break
			}
		}
		m.refreshCursorView()
		return m, nil

	case "r":
		// Dive into the focused thread on this line (reply from there)
		if thread := m.focusedThreadAtCursor(); thread != nil {
			m.selectedThread = thread
			m.returnToLineSelect = true
			m.mode = ModeThreadView
			m.threadViewport.SetContent(m.renderThread())
		}
		return m, nil

	case "tab":
		// Cycle between threads stacked on this line
		indices := m.threadIndicesAtLine(m.selectedLine)
		if len(indices) > 1 {
			next := indices[0]
			for i, idx := range indices {
				if idx == m.selectedComment {
					next = indices[(i+1)%len(indices)]
					break
				}
			}
			m.selectedComment = next
			m.commentViewport.SetContent(m.renderComments())
		}
		return m, nil

	case "s":
		// Check if on a heading line
		if m.isHeadingLine(m.selectedLine) {
			// On heading - choose range vs section
			m.mode = ModeSelectSuggestionType
			return m, nil
		}
		// Regular line - start range selection
		m.rangeStartLine = m.selectedLine
		m.rangeEndLine = m.selectedLine
		m.rangeActive = true
		m.suggestionIsSection = false
		m.mode = ModeSelectRange
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		return m, nil
	}

	return m, nil
}

// handleChooseTargetKeys handles keys in choose target mode
func (m Model) handleChooseTargetKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		// Section mode
		m.targetIsSection = true
		m.mode = ModeAddComment
		m.commentInput.Reset()
		m.commentInput.Focus()
		return m, textarea.Blink

	case "l":
		// Line only mode
		m.targetIsSection = false
		m.mode = ModeAddComment
		m.commentInput.Reset()
		m.commentInput.Focus()
		return m, textarea.Blink

	case "esc", "q":
		m.mode = ModeLineSelect
		return m, nil
	}

	return m, nil
}

// handleSelectSuggestionTypeKeys handles keys in select suggestion type mode
func (m Model) handleSelectSuggestionTypeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		// Range selection
		section := m.getSectionAtLine(m.selectedLine)
		if section != nil {
			m.rangeStartLine = section.StartLine
			m.rangeEndLine = section.EndLine
		} else {
			m.rangeStartLine = m.selectedLine
			m.rangeEndLine = m.selectedLine
		}
		m.rangeActive = true
		m.suggestionIsSection = false
		m.mode = ModeSelectRange
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		return m, nil

	case "s":
		// Section-based suggestion
		section := m.getSectionAtLine(m.selectedLine)
		if section != nil {
			m.rangeStartLine = section.StartLine
			m.rangeEndLine = section.EndLine
			m.suggestionIsSection = true

			// Capture original text from range
			lines := strings.Split(m.doc.Content, "\n")
			if m.rangeStartLine > 0 && m.rangeEndLine <= len(lines) {
				originalLines := lines[m.rangeStartLine-1 : m.rangeEndLine]
				m.suggestionOriginalText = strings.Join(originalLines, "\n")
			}

			m.mode = ModeAddSuggestion
			m.proposedTextInput.Reset()
			m.proposedTextInput.SetValue(m.suggestionOriginalText)
			m.proposedTextInput.Focus()
			return m, textarea.Blink
		}
		return m, nil

	case "esc", "q":
		m.mode = ModeLineSelect
		return m, nil
	}

	return m, nil
}

// handleSelectRangeKeys handles keys in select range mode
func (m Model) handleSelectRangeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	lines := strings.Split(m.doc.Content, "\n")
	totalLines := len(lines)

	switch msg.String() {
	case "j", "down":
		// Extend range down
		if m.rangeEndLine < totalLines {
			m.rangeEndLine++
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			m.scrollToLine(m.rangeEndLine)
		}
		return m, nil

	case "k", "up":
		// Shrink range up
		if m.rangeEndLine > m.rangeStartLine {
			m.rangeEndLine--
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			m.scrollToLine(m.rangeEndLine)
		}
		return m, nil

	case "enter":
		// Confirm range - capture original text
		if m.rangeStartLine > 0 && m.rangeEndLine <= totalLines {
			originalLines := lines[m.rangeStartLine-1 : m.rangeEndLine]
			m.suggestionOriginalText = strings.Join(originalLines, "\n")
		}
		m.mode = ModeAddSuggestion
		m.proposedTextInput.Reset()
		m.proposedTextInput.SetValue(m.suggestionOriginalText)
		m.proposedTextInput.Focus()
		return m, textarea.Blink

	case "esc", "q":
		// Cancel range selection
		m.rangeActive = false
		m.mode = ModeLineSelect
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		return m, nil
	}

	return m, nil
}

// calculateDisplayRow calculates the actual display row for a line number,
// accounting for wrapped lines (shared wrap math: docWrapWidth)
func (m *Model) calculateDisplayRow(targetLineNum int) int {
	if m.doc == nil {
		return 0
	}

	lines := strings.Split(m.doc.Content, "\n")
	availableWidth := m.docWrapWidth()

	displayRow := 0
	for i := 0; i < len(lines) && i < targetLineNum; i++ {
		line := lines[i]
		// Count how many rows this line takes when wrapped
		wrappedLines := strings.Split(wordwrap.String(line, availableWidth), "\n")
		displayRow += len(wrappedLines)
	}

	return displayRow
}

// scrollToLine adjusts viewport to keep the specified line visible
func (m *Model) scrollToLine(lineNum int) {
	// Special case: if going to line 1, use GotoTop
	if lineNum == 1 {
		m.documentViewport.GotoTop()
		return
	}

	// Calculate the actual display row for this line (accounting for wrapped lines)
	displayRow := m.calculateDisplayRow(lineNum - 1) // -1 because we want the start of this line

	// Calculate visible range
	topRow := m.documentViewport.YOffset
	bottomRow := topRow + m.documentViewport.Height - 1

	// Scroll if line is out of view
	if displayRow < topRow {
		// Line is above visible area - scroll up
		m.documentViewport.YOffset = displayRow
	} else if displayRow > bottomRow {
		// Line is below visible area - scroll down
		// Position it near the bottom of the viewport
		m.documentViewport.YOffset = displayRow - m.documentViewport.Height + 5
	}

	// Ensure we don't scroll past the end
	m.documentViewport.YOffset = max(m.documentViewport.YOffset, 0)
}

// threadIndicesAtLine returns indices into visibleComments() of threads on a line
func (m *Model) threadIndicesAtLine(line int) []int {
	indices := []int{}
	for i, c := range m.visibleComments() {
		if c.Line == line {
			indices = append(indices, i)
		}
	}
	return indices
}

// focusedThreadAtCursor returns the selected thread on the cursor line, if any.
// Prefers the current sidebar selection when it sits on this line (Tab cycling).
func (m *Model) focusedThreadAtCursor() *comment.Comment {
	visible := m.visibleComments()
	if m.selectedComment >= 0 && m.selectedComment < len(visible) && visible[m.selectedComment].Line == m.selectedLine {
		return visible[m.selectedComment]
	}
	if indices := m.threadIndicesAtLine(m.selectedLine); len(indices) > 0 {
		return visible[indices[0]]
	}
	return nil
}

// refreshSidebar re-renders the comment sidebar around the current focus line
// and scrolls the focused group into view (focus-follows-cursor, G3)
func (m *Model) refreshSidebar() {
	// Keep the sidebar selection in step with the cursor: moving onto a
	// commented line selects its first thread (Tab cycles among them)
	if m.mode == ModeLineSelect {
		if indices := m.threadIndicesAtLine(m.selectedLine); len(indices) > 0 && !slices.Contains(indices, m.selectedComment) {
			m.selectedComment = indices[0]
		}
	}

	content := m.renderComments()
	m.commentViewport.SetContent(content)

	// Scroll the expanded ("▼") group into view
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "▼ ") {
			offset := max(i-2, 0)
			maxOffset := max(len(lines)-m.commentViewport.Height, 0)
			m.commentViewport.YOffset = min(offset, maxOffset)
			return
		}
	}
}

// getSectionAtLine returns the section containing the given line, or nil
func (m *Model) getSectionAtLine(lineNum int) *markdown.Section {
	if m.documentSections == nil {
		return nil
	}
	if section, exists := m.documentSections.SectionsByLine[lineNum]; exists {
		return section
	}
	return nil
}

// isHeadingLine returns true if the line is a markdown heading
func (m *Model) isHeadingLine(lineNum int) bool {
	section := m.getSectionAtLine(lineNum)
	if section == nil {
		return false
	}
	return section.StartLine == lineNum
}

// getSectionPath returns the full hierarchical path for a section
func (m *Model) getSectionPath(section *markdown.Section) string {
	if section == nil {
		return ""
	}
	return section.GetFullPath(m.documentSections.SectionsByID)
}

// viewChooseTarget renders the section vs line choice modal
func (m Model) viewChooseTarget() string {
	if !m.ready {
		return "Loading..."
	}

	// Base layout with document
	modeStr := "Choose Target"
	title := m.styles.title.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		m.styles.commentPanel.Render(m.commentViewport.View()),
	)

	// Get section info
	section := m.getSectionAtLine(m.selectedLine)
	sectionPath := m.getSectionPath(section)

	// Build choice modal
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Title).
		Render("Add comment to:")

	var choices strings.Builder
	fmt.Fprintf(&choices, "  [s] 📍 Section: %s\n", sectionPath)
	fmt.Fprintf(&choices, "      (covers lines %d-%d)\n\n", section.StartLine, section.EndLine)
	fmt.Fprintf(&choices, "  [l] 💬 Line %d only (heading line)\n\n", m.selectedLine)

	modalHelp := m.styles.help.Render("s: section • l: line • Esc: cancel")

	modal := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			choices.String(),
			modalHelp,
		),
	)

	// Position modal over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.Place(
			m.width,
			m.height-2,
			lipgloss.Left,
			lipgloss.Top,
			content,
		),
		positioned,
	)
}

// viewSelectSuggestionType renders the suggestion type choice modal
func (m Model) viewSelectSuggestionType() string {
	if !m.ready {
		return "Loading..."
	}

	// Base layout with document
	modeStr := "Choose Suggestion Type"
	title := m.styles.title.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		m.styles.commentPanel.Render(m.commentViewport.View()),
	)

	// Get section info
	section := m.getSectionAtLine(m.selectedLine)
	sectionPath := m.getSectionPath(section)

	// Build choice modal
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.theme.Title).
		Render("Create suggestion for:")

	var choices strings.Builder
	choices.WriteString("  [r] Line range (manual selection)\n\n")
	fmt.Fprintf(&choices, "  [s] 📍 Section: %s\n", sectionPath)
	fmt.Fprintf(&choices, "      (lines %d-%d)\n\n", section.StartLine, section.EndLine)

	modalHelp := m.styles.help.Render("r: range • s: section • Esc: cancel")

	modal := m.styles.modalOverlay.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			choices.String(),
			modalHelp,
		),
	)

	// Position modal over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		lipgloss.Place(
			m.width,
			m.height-2,
			lipgloss.Left,
			lipgloss.Top,
			content,
		),
		positioned,
	)
}

// viewSelectRange renders the range selection view
func (m Model) viewSelectRange() string {
	if !m.ready {
		return "Loading..."
	}

	// Base layout with document (showing range highlighting)
	modeStr := fmt.Sprintf("Range Selection: Lines %d-%d", m.rangeStartLine, m.rangeEndLine)
	title := m.styles.title.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		m.styles.commentPanel.Render(m.commentViewport.View()),
	)

	helpText := m.styles.help.Render("j/k: adjust end line • Enter: confirm • Esc: cancel")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		helpText,
	)
}
