package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/markdown"
)

// Model represents the enhanced TUI application state
type Model struct {
	// View mode
	mode ViewMode

	// File picker
	filePicker       filepicker.Model
	startedWithFile  bool // Track if file was provided directly vs picked

	// Document state
	doc              *comment.DocumentWithComments
	filename         string
	documentSections *markdown.DocumentStructure // Parsed section hierarchy

	// UI components
	documentViewport viewport.Model
	commentViewport  viewport.Model
	threadViewport   viewport.Model
	commentInput     textarea.Model

	// Selection state
	selectedLine       int              // For line selection mode
	selectedComment    int              // For comment navigation
	selectedThread     *comment.Comment // Thread root (v2.0)
	returnToLineSelect bool             // Thread view was entered from line-select; Esc returns there
	verdictReturnMode  ViewMode         // Mode to restore when leaving the verdict dialog
	VerdictDecision    string           // Set when the user exits via the verdict dialog
	showResolved       bool

	// Input state
	author      string // User name for comments
	priority    string // Priority for new comment: low, medium (default), high
	commentType string // Type for new comment: Q, S, B, T, E, or empty for no type

	// Section input support
	targetIsSection bool // True if user wants to comment on section, false for line only

	// Suggestion creation state
	suggestionOriginalText string         // Original text for suggestion being created
	proposedTextInput      textarea.Model // For entering proposed text

	// Multi-line suggestion support
	rangeStartLine      int  // Start line for range selection
	rangeEndLine        int  // End line for range selection
	rangeActive         bool // True if range selection is active
	suggestionIsSection bool // True if suggestion is section-based

	// Review pack state
	suggestionQueue   map[string]bool // suggestion ID -> accept(true)/reject(false); applied at verdict
	sidebarDensity    int             // densityFull / densityCondensed / densityHidden (S cycles)
	showLineSummaries bool            // dimmed end-of-line thread summaries (L toggles)
	helpReturnMode    ViewMode        // mode to restore when closing the help overlay
	tocReturnMode     ViewMode        // mode to restore when closing the TOC overlay
	tocEntries        []tocEntry      // flattened section list for the TOC overlay
	tocSelected       int             // selected row in the TOC overlay
	restoredYOffset   int             // document scroll restored from persisted view state

	// Dimensions
	width  int
	height int
	ready  bool

	// Error handling
	err error
}

// NewModel creates a new TUI model with file picker
func NewModel() Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".md", ".markdown"}
	fp.CurrentDirectory, _ = os.Getwd()

	ta := textarea.New()
	ta.Placeholder = "Enter your comment..."
	ta.Focus()

	proposedTA := textarea.New()
	proposedTA.Placeholder = "Enter proposed text (edit the pre-filled original)..."

	// Get author from environment or use default
	author := os.Getenv("USER")
	if author == "" {
		author = "user"
	}

	return Model{
		mode:              ModeFilePicker,
		filePicker:        fp,
		commentInput:      ta,
		proposedTextInput: proposedTA,
		author:            author,
		priority:          "medium",
		commentType:       "",
		showResolved:      false,
		startedWithFile:   false,
		suggestionQueue:   map[string]bool{},
		showLineSummaries: true,
	}
}

// NewModelWithFile creates a model with a pre-loaded file
func NewModelWithFile(doc *comment.DocumentWithComments, filename string) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment..."
	ta.Focus()

	proposedTA := textarea.New()
	proposedTA.Placeholder = "Enter proposed text (edit the pre-filled original)..."

	// Get author from environment or use default
	author := os.Getenv("USER")
	if author == "" {
		author = "user"
	}

	m := Model{
		mode:              ModeBrowse,
		doc:               doc,
		filename:          filename,
		commentInput:      ta,
		proposedTextInput: proposedTA,
		author:            author,
		priority:          "medium",
		commentType:       "",
		showResolved:      false,
		startedWithFile:   true,
		suggestionQueue:   map[string]bool{},
		showLineSummaries: true,
	}

	// Parse sections
	if doc != nil {
		m.documentSections = markdown.ParseDocument(doc.Content)
	}

	// Resume the previous review position, if one was persisted
	if st, ok := loadViewState(filename); ok {
		m.selectedLine = st.SelectedLine
		m.restoredYOffset = st.YOffset
	}

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.mode == ModeFilePicker {
		return m.filePicker.Init()
	}
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.handleResize()

	case tea.KeyMsg:
		// Ctrl+C is the universal silent escape: persist the reading
		// position, quit without a verdict (queued decisions stay unapplied)
		if msg.String() == "ctrl+c" {
			m.saveViewStateNow()
			return m, tea.Quit
		}
		return m.handleKeyPress(msg)
	}

	// Delegate to mode-specific updates
	return m.updateByMode(msg)
}

// handleResize adjusts viewports based on window size
func (m *Model) handleResize() {
	if m.mode == ModeFilePicker {
		return
	}

	// Split screen: 60% document / 40% comments, unless the sidebar is hidden
	docWidth := m.docPaneWidth()
	panelWidth := m.width - docWidth - 4
	if panelWidth < 0 {
		panelWidth = 0
	}

	// Set textarea width to use most of the screen width
	// Account for modal borders (2), padding (4), and some margin (10)
	textareaWidth := m.width - 16
	if textareaWidth < 40 {
		textareaWidth = 40 // Minimum width
	}
	m.commentInput.SetWidth(textareaWidth)

	if !m.ready {
		m.documentViewport = viewport.New(docWidth, m.height-2)
		m.commentViewport = viewport.New(panelWidth, m.height-2)
		m.threadViewport = viewport.New(m.width-4, m.height-2)

		if m.doc != nil {
			m.documentViewport.SetContent(m.renderDocument())
			// Resume the persisted scroll position (0 when none was saved)
			m.documentViewport.SetYOffset(m.restoredYOffset)
			m.commentViewport.SetContent(m.renderComments())
			m.commentViewport.YOffset = 0 // Explicitly start at top
		}
		m.ready = true
	} else {
		m.documentViewport.Width = docWidth
		m.documentViewport.Height = m.height - 2
		m.commentViewport.Width = panelWidth
		m.commentViewport.Height = m.height - 2
		m.threadViewport.Width = m.width - 4
		m.threadViewport.Height = m.height - 2
	}
}

// Sidebar density levels (S cycles: full → condensed → hidden)
const (
	densityFull      = iota // expanded threads at the focused line
	densityCondensed        // one line per thread, counts only
	densityHidden           // sidebar gone; document takes the full width
)

// docPaneWidth returns the document pane width for the current sidebar density
func (m *Model) docPaneWidth() int {
	if m.sidebarDensity == densityHidden {
		width := m.width - 2
		if width < 1 {
			width = 1
		}
		return width
	}
	return int(float64(m.width) * 0.6)
}

// cycleSidebarDensity advances full → condensed → hidden → full and
// reflows both panes at the new widths
func (m *Model) cycleSidebarDensity() {
	m.sidebarDensity = (m.sidebarDensity + 1) % 3
	m.handleResize()
	m.refreshDocumentPane()
	m.commentViewport.SetContent(m.renderComments())
}

// toggleLineSummaries flips the virtual-text summaries and re-renders
func (m *Model) toggleLineSummaries() {
	m.showLineSummaries = !m.showLineSummaries
	m.refreshDocumentPane()
}

// refreshDocumentPane re-renders the document viewport for the current mode
func (m *Model) refreshDocumentPane() {
	if m.doc == nil {
		return
	}
	if m.mode == ModeLineSelect || m.mode == ModeSelectRange {
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
	} else {
		m.documentViewport.SetContent(m.renderDocument())
	}
}

// handleKeyPress handles keyboard input based on current mode
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeFilePicker:
		return m.handleFilePickerKeys(msg)
	case ModeBrowse:
		return m.handleBrowseKeys(msg)
	case ModeLineSelect:
		return m.handleLineSelectKeys(msg)
	case ModeAddComment:
		return m.handleAddCommentKeys(msg)
	case ModeThreadView:
		return m.handleThreadViewKeys(msg)
	case ModeVerdict:
		return m.handleVerdictKeys(msg)
	case ModeReply:
		return m.handleReplyKeys(msg)
	case ModeResolve:
		return m.handleResolveKeys(msg)
	case ModeAddSuggestion:
		return m.handleAddSuggestionKeys(msg)
	case ModeChooseTarget:
		return m.handleChooseTargetKeys(msg)
	case ModeSelectSuggestionType:
		return m.handleSelectSuggestionTypeKeys(msg)
	case ModeSelectRange:
		return m.handleSelectRangeKeys(msg)
	case ModeHelp:
		return m.handleHelpKeys(msg)
	case ModeTOC:
		return m.handleTOCKeys(msg)
	default:
		return m, nil
	}
}

// handleFilePickerKeys handles keys in file picker mode
func (m Model) handleFilePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	// Check if a file was selected
	if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
		return m.loadFile(path)
	}

	return m, cmd
}

// handleBrowseKeys handles keys in browse mode
func (m Model) handleBrowseKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		// Exit is a verdict (approved TUI design): q opens the verdict dialog
		if m.doc != nil {
			m.verdictReturnMode = ModeBrowse
			m.mode = ModeVerdict
			return m, nil
		}
		if m.startedWithFile {
			return m, tea.Quit
		}
		m.mode = ModeFilePicker
		m.doc = nil
		m.filename = ""
		m.ready = false
		return m, nil

	case "c":
		// Enter line selection mode to add comment; keep a restored or
		// previously used cursor position instead of jumping to the top
		m.mode = ModeLineSelect
		if m.selectedLine < 1 {
			m.selectedLine = 1
		}

		// Completely reset the viewport to fix scroll offset issues
		m.documentViewport = viewport.New(m.docPaneWidth(), m.height-2)
		m.documentViewport.YOffset = 0
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
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
		// Navigate comments
		visibleComments := m.visibleComments()
		if m.selectedComment < len(visibleComments)-1 {
			m.selectedComment++
			m.commentViewport.SetContent(m.renderComments())
			// Scroll document to center the selected comment
			m.scrollToComment(visibleComments[m.selectedComment])
		}
		return m, nil

	case "k", "up":
		visibleComments := m.visibleComments()
		if m.selectedComment > 0 {
			m.selectedComment--
			m.commentViewport.SetContent(m.renderComments())
			// Scroll document to center the selected comment
			m.scrollToComment(visibleComments[m.selectedComment])
		}
		return m, nil

	case "enter":
		// Expand selected comment thread
		visibleComments := m.visibleComments()
		if len(visibleComments) > 0 && m.selectedComment < len(visibleComments) {
			selectedThread := visibleComments[m.selectedComment]
			m.selectedThread = selectedThread
			m.mode = ModeThreadView
			m.threadViewport.SetContent(m.renderThread())
			// Scroll document to center the thread's comment
			m.scrollToComment(selectedThread)
			return m, nil
		}
		return m, nil

	case "R":
		// Toggle showing resolved comments
		m.showResolved = !m.showResolved
		m.commentViewport.SetContent(m.renderComments())
		return m, nil
	}

	return m, nil
}

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
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			m.scrollToLine(m.selectedLine)
			m.refreshSidebar()
		}
		return m, nil

	case "k", "up":
		// Move cursor up
		if m.selectedLine > 1 {
			m.selectedLine--
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			m.scrollToLine(m.selectedLine)
			m.refreshSidebar()
		}
		return m, nil

	case "ctrl+d":
		// Page down (half page)
		pageSize := m.documentViewport.Height / 2
		m.selectedLine += pageSize
		if m.selectedLine > totalLines {
			m.selectedLine = totalLines
		}
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
		return m, nil

	case "ctrl+u":
		// Page up (half page)
		pageSize := m.documentViewport.Height / 2
		m.selectedLine -= pageSize
		if m.selectedLine < 1 {
			m.selectedLine = 1
		}
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
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
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
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
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
		return m, nil

	case "[":
		// Jump to previous line with an open thread
		visible := m.visibleComments()
		for i := len(visible) - 1; i >= 0; i-- {
			if !visible[i].Resolved && visible[i].Line < m.selectedLine {
				m.selectedLine = visible[i].Line
				break
			}
		}
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
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
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
		return m, nil

	case "N":
		// Jump to previous line whose threads have NEW activity
		since := lastSignoffTime(m.doc.Reviews)
		visible := m.visibleComments()
		for i := len(visible) - 1; i >= 0; i-- {
			if visible[i].Line < m.selectedLine && threadHasNewActivity(visible[i], since) {
				m.selectedLine = visible[i].Line
				break
			}
		}
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.scrollToLine(m.selectedLine)
		m.refreshSidebar()
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

// calculateDisplayRow calculates the actual display row for a line number, accounting for wrapped lines
func (m *Model) calculateDisplayRow(targetLineNum int) int {
	if m.doc == nil {
		return 0
	}

	lines := strings.Split(m.doc.Content, "\n")

	// Calculate available width for text
	availableWidth := m.documentViewport.Width - 12
	if availableWidth < 40 {
		availableWidth = 40
	}

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
	if m.documentViewport.YOffset < 0 {
		m.documentViewport.YOffset = 0
	}
}

// handleAddCommentKeys handles keys in add comment mode
func (m Model) handleAddCommentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel comment creation
		m.mode = ModeLineSelect
		m.commentInput.Reset()
		// Reset priority and type to defaults
		m.priority = "medium"
		m.commentType = ""
		return m, nil

	case "ctrl+p":
		// Cycle priority: medium -> high -> low -> medium
		switch m.priority {
		case "medium":
			m.priority = "high"
		case "high":
			m.priority = "low"
		case "low":
			m.priority = "medium"
		default:
			m.priority = "medium"
		}
		return m, nil

	case "ctrl+t":
		// Cycle type: none -> Q -> S -> B -> T -> E -> none
		switch m.commentType {
		case "":
			m.commentType = "Q"
		case "Q":
			m.commentType = "S"
		case "S":
			m.commentType = "B"
		case "B":
			m.commentType = "T"
		case "T":
			m.commentType = "E"
		case "E":
			m.commentType = ""
		default:
			m.commentType = ""
		}
		return m, nil

	case "ctrl+s":
		// Save comment
		text := strings.TrimSpace(m.commentInput.Value())
		if text == "" {
			// Empty comment, just cancel
			m.mode = ModeLineSelect
			m.commentInput.Reset()
			// Reset priority and type to defaults
			m.priority = "medium"
			m.commentType = ""
			return m, nil
		}

		// Create new comment with type if specified
		var newComment *comment.Comment
		if m.commentType != "" {
			// Auto-prefix text with type like CLI does
			commentText := "[" + m.commentType + "] " + text
			newComment = comment.NewCommentWithType(m.author, m.selectedLine, commentText, m.commentType)
		} else {
			newComment = comment.NewComment(m.author, m.selectedLine, text)
		}

		// Set priority and status
		newComment.Priority = m.priority
		newComment.Status = "active"

		// Add section metadata if targeting section
		if m.targetIsSection {
			comment.UpdateCommentSection(newComment, m.doc.Content)
		}

		m.doc.Threads = append(m.doc.Threads, newComment)

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.commentViewport.SetContent(m.renderComments())

		// Return to line select mode and reset to defaults
		m.mode = ModeLineSelect
		m.commentInput.Reset()
		m.priority = "medium"
		m.commentType = ""
		return m, nil
	}

	// Handle textarea input
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return m, cmd
}

// handleThreadViewKeys handles keys in thread view mode
func (m Model) handleThreadViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Return to where the thread was opened from
		if m.returnToLineSelect {
			m.returnToLineSelect = false
			m.selectedThread = nil
			m.mode = ModeLineSelect
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			m.scrollToLine(m.selectedLine)
			m.refreshSidebar()
			return m, nil
		}
		m.mode = ModeBrowse
		m.selectedThread = nil
		return m, nil

	case "q":
		// If file was provided directly, quit the app
		// Otherwise, go back to file picker
		if m.startedWithFile {
			m.saveViewStateNow()
			return m, tea.Quit
		}
		m.mode = ModeFilePicker
		m.selectedThread = nil
		m.doc = nil
		m.filename = ""
		m.ready = false
		return m, nil

	case "r":
		// Enter reply mode
		m.mode = ModeReply
		m.commentInput.Reset()
		m.commentInput.Focus()
		return m, textarea.Blink

	case "a":
		// Queue an accept for this pending suggestion; nothing mutates
		// until the verdict dialog applies the queue ("queue until verdict")
		if m.selectedThread != nil && m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
			m.queueDecision(m.selectedThread.ID, true)
			m.threadViewport.SetContent(m.renderThread())
		}
		return m, nil

	case "x":
		// Queue a reject for a pending suggestion; otherwise resolve thread
		if m.selectedThread != nil && m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
			m.queueDecision(m.selectedThread.ID, false)
			m.threadViewport.SetContent(m.renderThread())
			return m, nil
		}
		// Otherwise, enter resolve mode for regular threads
		m.mode = ModeResolve
		return m, nil
	}

	// Scroll thread viewport
	var cmd tea.Cmd
	m.threadViewport, cmd = m.threadViewport.Update(msg)
	return m, cmd
}

// handleReplyKeys handles keys in reply mode
func (m Model) handleReplyKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel reply
		m.mode = ModeThreadView
		m.commentInput.Reset()
		return m, nil

	case "ctrl+s":
		// Save reply
		text := strings.TrimSpace(m.commentInput.Value())
		if text == "" {
			// Empty reply, just cancel
			m.mode = ModeThreadView
			m.commentInput.Reset()
			return m, nil
		}

		// Add reply to thread using helper
		if err := comment.AddReplyToThread(m.doc.Threads, m.selectedThread.ID, m.author, text); err != nil {
			m.err = err
			return m, nil
		}

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.threadViewport.SetContent(m.renderThread())
		m.commentViewport.SetContent(m.renderComments())

		// Return to thread view
		m.mode = ModeThreadView
		m.commentInput.Reset()
		return m, nil
	}

	// Handle textarea input
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return m, cmd
}

// handleResolveKeys handles keys in resolve confirmation mode
func (m Model) handleResolveKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		// Cancel resolution
		m.mode = ModeThreadView
		return m, nil

	case "y", "enter":
		// Confirm resolution
		if err := comment.ResolveThread(m.doc.Threads, m.selectedThread.ID); err != nil {
			m.err = err
			return m, nil
		}

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.commentViewport.SetContent(m.renderComments())

		// Return to browse mode (thread is now resolved)
		m.mode = ModeBrowse
		m.selectedThread = nil
		return m, nil
	}

	return m, nil
}

// handleAddSuggestionKeys handles keys in add suggestion mode
func (m Model) handleAddSuggestionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel suggestion creation
		m.mode = ModeLineSelect
		m.suggestionOriginalText = ""
		m.rangeActive = false
		m.suggestionIsSection = false
		m.proposedTextInput.Reset()
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		return m, nil

	case "ctrl+s", "ctrl+d":
		// Submit suggestion
		proposedText := m.proposedTextInput.Value()
		if proposedText == "" {
			// Don't create empty suggestion
			m.mode = ModeLineSelect
			m.suggestionOriginalText = ""
			m.rangeActive = false
			m.suggestionIsSection = false
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
			return m, nil
		}

		// Use range if set, otherwise fall back to selectedLine
		startLine := m.selectedLine
		endLine := m.selectedLine
		if m.rangeStartLine > 0 && m.rangeEndLine > 0 {
			startLine = m.rangeStartLine
			endLine = m.rangeEndLine
		}

		// Create suggestion using helper (multi-line)
		suggestion := comment.NewSuggestion(
			m.author,
			startLine,
			endLine,
			"Suggestion",
			m.suggestionOriginalText,
			proposedText,
		)

		// Add section metadata if section-based
		if m.suggestionIsSection {
			comment.UpdateCommentSection(suggestion, m.doc.Content)
		}

		// Add to document
		m.doc.Threads = append(m.doc.Threads, suggestion)

		// Reset state
		m.rangeActive = false
		m.suggestionIsSection = false

		// Save document
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.documentViewport.SetContent(m.renderDocument())
		m.commentViewport.SetContent(m.renderComments())

		// Return to browse mode
		m.mode = ModeBrowse
		m.suggestionOriginalText = ""
		m.proposedTextInput.Reset()
		return m, nil
	}

	// Handle textarea input
	var cmd tea.Cmd
	m.proposedTextInput, cmd = m.proposedTextInput.Update(msg)
	return m, cmd
}

// updateByMode delegates updates to mode-specific logic
func (m Model) updateByMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
	case ModeFilePicker:
		m.filePicker, cmd = m.filePicker.Update(msg)
	case ModeBrowse:
		// Only allow viewport updates in browse mode, not line select
		m.documentViewport, cmd = m.documentViewport.Update(msg)
	case ModeLineSelect:
		// Don't update viewport in line select mode - we control scrolling manually
		cmd = nil
	case ModeAddComment, ModeReply:
		m.commentInput, cmd = m.commentInput.Update(msg)
	case ModeThreadView:
		m.threadViewport, cmd = m.threadViewport.Update(msg)
	case ModeHelp, ModeTOC:
		// Overlays are static; keys are handled in their key handlers
		cmd = nil
	}

	return m, cmd
}

// loadFile loads a markdown file and transitions to browse mode
func (m Model) loadFile(path string) (tea.Model, tea.Cmd) {
	// Load document from sidecar
	doc, err := comment.LoadFromSidecar(path)
	if err != nil {
		m.err = err
		return m, nil
	}

	// Update model
	m.doc = doc
	m.filename = path
	m.mode = ModeBrowse
	m.selectedComment = 0
	m.ready = false

	// Parse sections
	m.documentSections = markdown.ParseDocument(m.doc.Content)

	// If we have dimensions, initialize viewports now
	if m.width > 0 && m.height > 0 {
		m.handleResize()
	}

	return m, nil
}

// saveDocument saves the current document back to file
func (m *Model) saveDocument() error {
	if err := comment.SaveToSidecar(m.filename, m.doc); err != nil {
		return fmt.Errorf("saving document: %w", err)
	}

	return nil
}

// View renders the UI based on current mode
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit", m.err)
	}

	switch m.mode {
	case ModeFilePicker:
		return m.viewFilePicker()
	case ModeBrowse, ModeLineSelect:
		return m.viewBrowse()
	case ModeVerdict:
		return m.viewVerdict()
	case ModeAddComment:
		return m.viewAddComment()
	case ModeThreadView:
		return m.viewThread()
	case ModeReply:
		return m.viewReply()
	case ModeResolve:
		return m.viewResolve()
	case ModeAddSuggestion:
		return m.viewAddSuggestion()
	case ModeChooseTarget:
		return m.viewChooseTarget()
	case ModeSelectSuggestionType:
		return m.viewSelectSuggestionType()
	case ModeSelectRange:
		return m.viewSelectRange()
	case ModeHelp:
		return m.viewHelp()
	case ModeTOC:
		return m.viewTOC()
	default:
		return "Unknown mode"
	}
}

// viewFilePicker renders the file picker view
func (m Model) viewFilePicker() string {
	title := titleStyle.Render("comments - Select a markdown file")
	help := helpStyle.Render("↑/↓: navigate • Enter: select • q: quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.filePicker.View(),
		"",
		help,
	)
}

// viewBrowse renders the browse/line-select view
func (m Model) viewBrowse() string {
	if !m.ready {
		return "Loading..."
	}

	modeStr := m.mode.String()
	title := titleStyle.Render(fmt.Sprintf("📄 %s - %s", m.filename, modeStr))

	var helpText string
	if m.mode == ModeLineSelect {
		helpText = "j/k: move • r: open thread • n/N: next/prev NEW • Tab: cycle threads • c: comment • s: suggest • t: TOC • ?: help • Esc: cancel"
	} else {
		quitText := "back"
		if m.startedWithFile {
			quitText = "quit"
		}
		helpText = fmt.Sprintf("j/k: navigate • c: comment • Enter: expand • t: TOC • S: sidebar • ?: help • q: %s", quitText)
	}
	help := helpStyle.Render(helpText)

	// Layout: document on left, comments on right (unless sidebar is hidden)
	var content string
	if m.sidebarDensity == densityHidden {
		content = m.documentViewport.View()
	} else {
		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.documentViewport.View(),
			commentPanelStyle.Render(m.commentViewport.View()),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		help,
	)
}

// viewAddComment renders the add comment modal
func (m Model) viewAddComment() string {
	if !m.ready {
		return "Loading..."
	}

	// Base layout with document
	modeStr := "Adding Comment"
	title := titleStyle.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		commentPanelStyle.Render(m.commentViewport.View()),
	)

	// Get section-aware context
	var contextText string
	sectionContext := m.getSectionContext(m.selectedLine)
	if sectionContext != "" {
		// Use section-aware context
		contextText = sectionContext
	} else {
		// Fall back to line-based context if no section found
		contextLines := m.getContextLines(m.selectedLine, 2)
		var builder strings.Builder

		contextStyle := lipgloss.NewStyle().
			Foreground(activeTheme.MetaText).
			Italic(true)
		lineNumStyle := lipgloss.NewStyle().Foreground(activeTheme.LineNumber)
		highlightStyle := lipgloss.NewStyle().
			Background(activeTheme.SelectionBg).
			Foreground(activeTheme.SelectionFg).
			Bold(true)

		builder.WriteString(contextStyle.Render("Document Context:"))
		builder.WriteString("\n")

		for _, cl := range contextLines {
			linePrefix := fmt.Sprintf("%4d │ ", cl.LineNum)
			if cl.LineNum == m.selectedLine {
				builder.WriteString(lineNumStyle.Bold(true).Render(linePrefix))
				builder.WriteString(highlightStyle.Render(cl.Text))
			} else {
				builder.WriteString(lineNumStyle.Render(linePrefix))
				builder.WriteString(cl.Text)
			}
			builder.WriteString("\n")
		}
		contextText = builder.String()
	}

	// Current selection display
	selectionStyle := lipgloss.NewStyle().
		Foreground(activeTheme.Title).
		Bold(true)

	priorityLabel := selectionStyle.Render(fmt.Sprintf("Priority: %s", strings.ToUpper(m.priority)))

	typeLabel := "None"
	if m.commentType != "" {
		typeLabel = fmt.Sprintf("[%s]", m.commentType)
		switch m.commentType {
		case "Q":
			typeLabel += " Question"
		case "S":
			typeLabel += " Suggestion"
		case "B":
			typeLabel += " Bug"
		case "T":
			typeLabel += " TODO"
		case "E":
			typeLabel += " Enhancement"
		}
	}
	typeDisplay := selectionStyle.Render(fmt.Sprintf("Type: %s", typeLabel))

	selectionInfo := lipgloss.NewStyle().
		Foreground(activeTheme.MetaText).
		Render(fmt.Sprintf("%s  •  %s", priorityLabel, typeDisplay))

	// Modal overlay for comment input
	var titleText string
	if m.targetIsSection {
		section := m.getSectionAtLine(m.selectedLine)
		if section != nil {
			titleText = fmt.Sprintf("📍 Add Section Comment: %s", section.Title)
		} else {
			titleText = fmt.Sprintf("Add Comment at Line %d", m.selectedLine)
		}
	} else {
		titleText = fmt.Sprintf("💬 Add Comment at Line %d", m.selectedLine)
	}

	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeTheme.Title).
		Render(titleText)

	modalHelp := helpStyle.Render("Ctrl+S: save • Ctrl+P: cycle priority • Ctrl+T: cycle type • Esc: cancel")

	modal := modalOverlayStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			contextText,
			"",
			m.commentInput.View(),
			"",
			selectionInfo,
			"",
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

// viewThread renders the thread view
func (m Model) viewThread() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	title := titleStyle.Render(fmt.Sprintf("Thread at Line %d", m.selectedThread.Line))

	quitText := "file picker"
	if m.startedWithFile {
		quitText = "quit"
	}
	actionText := "x: resolve"
	if m.selectedThread.IsSuggestion && m.selectedThread.IsPending() {
		actionText = "a/x: queue accept/reject"
	}
	help := helpStyle.Render(fmt.Sprintf("r: reply • %s • Esc: back • q: %s", actionText, quitText))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.threadViewport.View(),
		"",
		help,
	)
}

// viewReply renders the reply modal
func (m Model) viewReply() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	title := titleStyle.Render(fmt.Sprintf("Thread at Line %d", m.selectedThread.Line))

	// Thread content as background
	threadContent := m.threadViewport.View()

	// Build thread context to show in modal
	var threadContext strings.Builder
	contextStyle := lipgloss.NewStyle().
		Foreground(activeTheme.MetaText).
		Italic(true)

	threadContext.WriteString(contextStyle.Render("Thread Context:"))
	threadContext.WriteString("\n\n")

	// Root comment (selectedThread IS the root comment in v2.0)
	threadContext.WriteString(fmt.Sprintf("┌ @%s · %s\n",
		m.selectedThread.Author,
		m.selectedThread.Timestamp.Format("2006-01-02 15:04")))

	// Truncate root comment if too long
	rootText := m.selectedThread.Text
	if len(rootText) > 60 {
		rootText = rootText[:57] + "..."
	}
	threadContext.WriteString(fmt.Sprintf("│ %s\n", rootText))

	// Show recent replies (last 2)
	replyCount := len(m.selectedThread.Replies)
	if replyCount > 0 {
		startIdx := 0
		if replyCount > 2 {
			threadContext.WriteString(fmt.Sprintf("│ ... (%d earlier replies)\n", replyCount-2))
			startIdx = replyCount - 2
		}

		for i := startIdx; i < replyCount; i++ {
			reply := m.selectedThread.Replies[i]
			threadContext.WriteString(fmt.Sprintf("├ @%s · %s\n",
				reply.Author,
				reply.Timestamp.Format("2006-01-02 15:04")))

			// Truncate reply if too long
			replyText := reply.Text
			if len(replyText) > 60 {
				replyText = replyText[:57] + "..."
			}
			threadContext.WriteString(fmt.Sprintf("│ %s\n", replyText))
		}
	}
	threadContext.WriteString("└──────────────────────\n")

	// Modal overlay for reply input
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeTheme.Title).
		Render("Reply to Thread")

	modalHelp := helpStyle.Render("Ctrl+S: save • Esc: cancel")

	modal := modalOverlayStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			threadContext.String(),
			"",
			m.commentInput.View(),
			"",
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
			threadContent,
		),
		positioned,
	)
}

// viewResolve renders the resolve confirmation dialog
func (m Model) viewResolve() string {
	if m.selectedThread == nil {
		return "No thread selected"
	}

	title := titleStyle.Render(fmt.Sprintf("Thread at Line %d", m.selectedThread.Line))

	// Thread content as background
	threadContent := m.threadViewport.View()

	// Confirmation dialog
	confirmTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeTheme.Title).
		Render("Resolve this thread?")

	confirmText := lipgloss.NewStyle().
		Render("This will mark the entire conversation as resolved.\nResolved comments can be toggled with 'R' in browse mode.")

	confirmHelp := helpStyle.Render("y/Enter: confirm • n/Esc: cancel")

	dialog := modalOverlayStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			confirmTitle,
			"",
			confirmText,
			"",
			confirmHelp,
		),
	)

	// Position dialog over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
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
			threadContent,
		),
		positioned,
	)
}

// viewAddSuggestion renders the add suggestion modal
func (m Model) viewAddSuggestion() string {
	title := titleStyle.Render(fmt.Sprintf("Add Suggestion for Line %d", m.selectedLine))

	// Document context as background
	docContent := m.documentViewport.View()

	// Suggestion creation form
	formTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeTheme.Warning).
		Render("Create Edit Suggestion")

	originalLabel := lipgloss.NewStyle().
		Foreground(activeTheme.DimSyntax).
		Render("Original text:")

	originalText := lipgloss.NewStyle().
		Background(activeTheme.SelectionBg).
		Padding(0, 1).
		Render(m.suggestionOriginalText)

	proposedLabel := lipgloss.NewStyle().
		Foreground(activeTheme.DimSyntax).
		Render("Proposed text (edit below):")

	help := helpStyle.Render("Ctrl+S or Ctrl+D: submit • Esc: cancel")

	dialog := modalOverlayStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			formTitle,
			"",
			originalLabel,
			originalText,
			"",
			proposedLabel,
			m.proposedTextInput.View(),
			"",
			help,
		),
	)

	// Position dialog over content
	positioned := lipgloss.Place(
		m.width,
		m.height-2,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
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
			docContent,
		),
		positioned,
	)
}

// scrollToComment scrolls the document viewport to center the given comment
func (m *Model) scrollToComment(c *comment.Comment) {
	if m.doc == nil || c == nil {
		return
	}

	// Get the comment's line position (line-only tracking in v2.0)
	targetLine := c.Line
	if targetLine < 1 {
		return
	}

	// Calculate the rendered line position accounting for line wrapping
	// We need to count how many rendered lines come before our target line
	lines := strings.Split(m.doc.Content, "\n")
	renderedLinesBefore := 0

	// Calculate available width for text (same as in renderDocument)
	availableWidth := m.documentViewport.Width - 10
	if availableWidth < 40 {
		availableWidth = 40
	}

	// Count wrapped lines for all lines before the target
	for i := 0; i < len(lines) && i < targetLine-1; i++ {
		wrappedCount := len(strings.Split(wordwrap.String(lines[i], availableWidth), "\n"))
		renderedLinesBefore += wrappedCount
	}

	// Calculate target Y offset to center the line in the viewport
	viewportHeight := m.documentViewport.Height
	targetOffset := renderedLinesBefore - (viewportHeight / 2)

	// Ensure we don't scroll before the start
	if targetOffset < 0 {
		targetOffset = 0
	}

	// Set the viewport offset
	m.documentViewport.YOffset = targetOffset
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
	title := titleStyle.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		commentPanelStyle.Render(m.commentViewport.View()),
	)

	// Get section info
	section := m.getSectionAtLine(m.selectedLine)
	sectionPath := m.getSectionPath(section)

	// Build choice modal
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeTheme.Title).
		Render("Add comment to:")

	var choices strings.Builder
	choices.WriteString(fmt.Sprintf("  [s] 📍 Section: %s\n", sectionPath))
	choices.WriteString(fmt.Sprintf("      (covers lines %d-%d)\n\n", section.StartLine, section.EndLine))
	choices.WriteString(fmt.Sprintf("  [l] 💬 Line %d only (heading line)\n\n", m.selectedLine))

	modalHelp := helpStyle.Render("s: section • l: line • Esc: cancel")

	modal := modalOverlayStyle.Render(
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
	title := titleStyle.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		commentPanelStyle.Render(m.commentViewport.View()),
	)

	// Get section info
	section := m.getSectionAtLine(m.selectedLine)
	sectionPath := m.getSectionPath(section)

	// Build choice modal
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeTheme.Title).
		Render("Create suggestion for:")

	var choices strings.Builder
	choices.WriteString("  [r] Line range (manual selection)\n\n")
	choices.WriteString(fmt.Sprintf("  [s] 📍 Section: %s\n", sectionPath))
	choices.WriteString(fmt.Sprintf("      (lines %d-%d)\n\n", section.StartLine, section.EndLine))

	modalHelp := helpStyle.Render("r: range • s: section • Esc: cancel")

	modal := modalOverlayStyle.Render(
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
	title := titleStyle.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	// Layout: document on left, comments on right (background)
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		commentPanelStyle.Render(m.commentViewport.View()),
	)

	helpText := helpStyle.Render("j/k: adjust end line • Enter: confirm • Esc: cancel")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		helpText,
	)
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
		if indices := m.threadIndicesAtLine(m.selectedLine); len(indices) > 0 {
			selectedOnLine := false
			for _, idx := range indices {
				if idx == m.selectedComment {
					selectedOnLine = true
					break
				}
			}
			if !selectedOnLine {
				m.selectedComment = indices[0]
			}
		}
	}

	content := m.renderComments()
	m.commentViewport.SetContent(content)

	// Scroll the expanded ("▼") group into view
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "▼ ") {
			offset := i - 2
			if offset < 0 {
				offset = 0
			}
			maxOffset := len(lines) - m.commentViewport.Height
			if maxOffset < 0 {
				maxOffset = 0
			}
			if offset > maxOffset {
				offset = maxOffset
			}
			m.commentViewport.YOffset = offset
			return
		}
	}
}

// handleVerdictKeys handles the exit verdict dialog: a approve / c request changes / esc back
func (m Model) handleVerdictKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	record := func(decision string) (tea.Model, tea.Cmd) {
		// Apply all queued suggestion decisions atomically, then record the
		// signoff — one save covers content, threads, and the review record
		if err := m.applySuggestionQueue(); err != nil {
			m.err = err
			return m, nil
		}
		comment.AddReviewRecord(m.doc, m.author, decision, "", false)
		if err := comment.SaveToSidecar(m.filename, m.doc); err != nil {
			m.err = err
			return m, nil
		}
		m.saveViewStateNow()
		m.VerdictDecision = decision
		return m, tea.Quit
	}
	switch msg.String() {
	case "a":
		return record(comment.DecisionApproved)
	case "c":
		return record(comment.DecisionChangesRequested)
	case "esc", "q":
		// Back to review; queued decisions are kept, not discarded
		m.mode = m.verdictReturnMode
		return m, nil
	}
	return m, nil
}

// viewVerdict renders the exit verdict dialog over a dimmed summary
func (m *Model) viewVerdict() string {
	result := comment.EvaluateGate(m.doc, false)
	queueLine := ""
	if n := len(m.suggestionQueue); n > 0 {
		queueLine = fmt.Sprintf("\n%d queued suggestion decision(s) — applied on submit; Esc keeps them\n", n)
	}
	dialog := fmt.Sprintf(
		"Submit review for %s\n\n%d blocking · %d open · %d pending suggestions\n%s\n[a] Approve (signoff, exit 0)\n[c] Request changes (signoff, exit 10)\n[Esc] Back to review",
		m.filename, len(result.Blocking), len(result.NonBlocking), len(result.PendingSuggestions), queueLine)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 3).Render(dialog)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
