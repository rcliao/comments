package tui

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/rcliao/comments/pkg/comment"
	"github.com/rcliao/comments/pkg/markdown"
)

// Model represents the enhanced TUI application state
type Model struct {
	// View mode
	mode ViewMode

	// Styles built from the selected theme (immutable after construction)
	styles *styleSet

	// File picker
	filePicker      filepicker.Model
	startedWithFile bool // Track if file was provided directly vs picked

	// Document state
	doc              *comment.DocumentWithComments
	filename         string
	documentSections *markdown.DocumentStructure // Parsed section hierarchy

	// UI components
	documentViewport viewport.Model
	commentViewport  viewport.Model
	threadViewport   viewport.Model
	commentInput     textarea.Model
	// replyInput is the panel-docked reply composer. It is its own textarea
	// (like proposedTextInput and verdictNote) because it grows with what you
	// type — sharing commentInput would mean saving and restoring four fields
	// every time the composer opens and closes.
	replyInput textarea.Model

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

	// Review note recorded with the verdict (signoff --note parity)
	verdictNote textarea.Model

	// Suggestion creation state
	suggestionOriginalText string         // Original text for suggestion being created
	proposedTextInput      textarea.Model // For entering proposed text

	// Multi-line suggestion support
	rangeStartLine      int  // Start line for range selection
	rangeEndLine        int  // End line for range selection
	rangeActive         bool // True if range selection is active
	suggestionIsSection bool // True if suggestion is section-based

	// Reference peek state (docs/design-reference-jump.md)
	refsByLine        map[int][]resolvedRef // citations per doc line, resolved at load
	refPeekList       []resolvedRef         // references on the peeked line
	refPeekIdx        int                   // which of refPeekList is shown
	refPeekReturnMode ViewMode              // mode to restore when closing the peek
	refPeekContent    []string              // target file lines (read at peek open)
	refPeekTargetLine int                   // cited line in the target (0 = none)
	refPeekErr        string                // resolution/read error shown in the peek

	// Review pack state
	suggestionQueue   map[string]bool // suggestion ID -> accept(true)/reject(false); applied at verdict
	sidebarDensity    int             // densityFull / densityCondensed / densityHidden (S cycles)
	showLineSummaries bool            // dimmed end-of-line thread summaries (L toggles)
	hideLineNumbers   bool            // line-number column hidden (# toggles)
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
	bindShiftEnterNewline(&ta)

	proposedTA := textarea.New()
	proposedTA.Placeholder = "Enter proposed text (edit the pre-filled original)..."
	bindShiftEnterNewline(&proposedTA)

	noteTA := textarea.New()
	noteTA.Placeholder = "Optional note for the agent reading this review..."
	noteTA.SetHeight(verdictNoteRows)
	noteTA.ShowLineNumbers = false
	bindShiftEnterNewline(&noteTA)

	replyTA := newReplyTextarea()

	// Get author from environment or use default
	author := os.Getenv("USER")
	if author == "" {
		author = "user"
	}

	return Model{
		mode:              ModeFilePicker,
		styles:            newStyleSet(currentStartupTheme()),
		filePicker:        fp,
		commentInput:      ta,
		proposedTextInput: proposedTA,
		verdictNote:       noteTA,
		replyInput:        replyTA,
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
	bindShiftEnterNewline(&ta)

	proposedTA := textarea.New()
	proposedTA.Placeholder = "Enter proposed text (edit the pre-filled original)..."
	bindShiftEnterNewline(&proposedTA)

	noteTA := textarea.New()
	noteTA.Placeholder = "Optional note for the agent reading this review..."
	noteTA.SetHeight(verdictNoteRows)
	noteTA.ShowLineNumbers = false
	bindShiftEnterNewline(&noteTA)

	replyTA := newReplyTextarea()

	// Get author from environment or use default
	author := os.Getenv("USER")
	if author == "" {
		author = "user"
	}

	m := Model{
		mode:              ModeBrowse,
		styles:            newStyleSet(currentStartupTheme()),
		doc:               doc,
		filename:          filename,
		commentInput:      ta,
		proposedTextInput: proposedTA,
		verdictNote:       noteTA,
		replyInput:        replyTA,
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
		// Detect and resolve file references once; rendering and peek only read this
		m.refsByLine = buildRefMap(doc.Content, filename)
	}

	// Resume the previous review position, if one was persisted
	if st, ok := loadViewState(filename); ok {
		m.selectedLine = st.SelectedLine
		m.restoredYOffset = st.YOffset
		m.hideLineNumbers = st.HideLineNumbers
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

	case tea.KeyPressMsg:
		// Ctrl+C is the universal silent escape: persist the reading
		// position, quit without a verdict (queued decisions stay unapplied)
		if msg.String() == "ctrl+c" {
			m.saveViewStateNow()
			return m, tea.Quit
		}
		return m.handleKeyPress(msg)

	case editorFinishedMsg:
		// $EDITOR handoff ended; resume the review where it was
		m.mode = m.refPeekReturnMode
		if msg.err != nil {
			m.err = fmt.Errorf("editor: %w", msg.err)
		}
		m.refreshCursorView()
		return m, nil
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
	panelWidth := max(m.width-docWidth-4, 0)

	// Size the textarea for the centered dialogs (see dialogTextareaWidth).
	// The docked reply composer has its own textarea, sized to the panel by
	// applyComposerLayout below.
	m.commentInput.SetWidth(m.dialogTextareaWidth())
	// The verdict note sits inside the verdict box, which is sized by its
	// content rather than the screen — keep it comfortably narrower
	m.verdictNote.SetWidth(min(max(m.width-24, 40), 72))

	if !m.ready {
		m.documentViewport = newViewport(docWidth, m.height-2)
		m.commentViewport = newViewport(panelWidth, m.height-2)
		m.threadViewport = newViewport(m.width-4, m.height-2)

		if m.doc != nil {
			m.documentViewport.SetContent(m.renderDocument())
			// Resume the persisted scroll position (0 when none was saved)
			m.documentViewport.SetYOffset(m.restoredYOffset)
			m.commentViewport.SetContent(m.renderComments())
			m.commentViewport.SetYOffset(0) // Explicitly start at top
		}
		m.ready = true
	} else {
		m.documentViewport.SetWidth(docWidth)
		m.documentViewport.SetHeight(m.height - 2)
		m.commentViewport.SetWidth(panelWidth)
		m.commentViewport.SetHeight(m.height - 2)
		m.threadViewport.SetWidth(m.width - 4)
		m.threadViewport.SetHeight(m.height - 2)
	}

	// An open thread panel re-derives its geometry at the new size
	// (keys_threadpanel.go); a docked reply composer re-sizes with it
	if m.mode == ModeReply {
		m.applyComposerLayout()
	}
	if m.selectedThread != nil {
		m.applyThreadPanel()
	}
}

// dialogTextareaWidth is commentInput's width in the centered dialogs: most of
// the screen, less modal borders (2), padding (4) and margin (10), clamped to
// a usable minimum.
func (m Model) dialogTextareaWidth() int {
	return max(m.width-16, 40)
}

// newViewport constructs a viewport at the given size. Bubbles v2 made the
// dimensions options/setters (viewport.New(w, h) is gone).
func newViewport(width, height int) viewport.Model {
	return viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
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
		return max(m.width-2, 1)
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

// refreshCursorView is the standard refresh after any cursor move in
// line-select mode: re-render the cursor view, keep the cursor line visible,
// and sync the sidebar focus.
func (m *Model) refreshCursorView() {
	m.documentViewport.SetContent(m.renderDocumentWithCursor())
	m.scrollToLine(m.selectedLine)
	m.refreshSidebar()
}

// handleKeyPress handles keyboard input based on current mode
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Error escape hatch: while an error is displayed (View short-circuits to
	// the error screen), any key dismisses it and returns to a sane mode —
	// browse when a document is loaded, otherwise the file picker.
	if m.err != nil {
		m.err = nil
		if m.doc != nil {
			m.mode = ModeBrowse
			if m.ready {
				m.refreshDocumentPane()
				m.commentViewport.SetContent(m.renderComments())
			}
		} else {
			m.mode = ModeFilePicker
			m.filename = ""
			m.ready = false
		}
		return m, nil
	}

	if d, ok := modeRegistry[m.mode]; ok {
		return d.handleKeys(m, msg)
	}
	return m, nil
}

// updateByMode routes non-key messages to the current mode's active component
func (m Model) updateByMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if d, ok := modeRegistry[m.mode]; ok && d.updateViewport != nil {
		cmd = d.updateViewport(&m, msg)
	}
	return m, cmd
}

// loadFile loads a markdown file and transitions to browse mode
func (m Model) loadFile(path string) (tea.Model, tea.Cmd) {
	// Load document from sidecar
	doc, report, err := comment.LoadFromSidecar(path)
	if err != nil {
		m.err = err
		return m, nil
	}
	if report.Dirty {
		// Persist re-anchoring/orphan-status migrations discovered during load
		// (best-effort; previously done inside LoadFromSidecar)
		_ = comment.SaveToSidecar(path, doc)
	}

	// Update model
	m.doc = doc
	m.filename = path
	m.mode = ModeBrowse
	m.selectedComment = 0
	m.ready = false

	// Parse sections
	m.documentSections = markdown.ParseDocument(m.doc.Content)

	// Detect and resolve file references once; rendering and peek only read this
	m.refsByLine = buildRefMap(m.doc.Content, path)

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

// refreshDocFromDisk reloads document + threads from disk before a mutation,
// so a TUI session that has been open a while doesn't clobber comments or
// content written meanwhile by an agent (CLI/MCP write to the same files; the
// sidecar save rewrites the whole file from memory). Mutations become
// last-writer-wins per action instead of per session. The selected thread is
// re-found by ID; a missing file or ID leaves current state untouched.
func (m *Model) refreshDocFromDisk() {
	if m.filename == "" || m.doc == nil {
		return
	}
	// No sidecar on disk means nothing external could have written threads —
	// refreshing would replace real in-memory state with an empty load
	// (models constructed directly, brand-new docs)
	if _, err := os.Stat(comment.GetSidecarPath(m.filename)); err != nil {
		return
	}
	fresh, _, err := comment.LoadFromSidecar(m.filename)
	if err != nil || fresh == nil {
		return
	}
	selectedID := ""
	if m.selectedThread != nil {
		selectedID = m.selectedThread.ID
	}
	contentChanged := fresh.Content != m.doc.Content
	oldContent := m.doc.Content
	m.doc = fresh
	if selectedID != "" {
		if c := fresh.FindCommentByID(selectedID); c != nil {
			m.selectedThread = c
		}
	}
	if contentChanged {
		// The cursor follows its TEXT through the swap, not its number — a
		// mutation about to use selectedLine must hit the line the human was
		// looking at (live bug: comments landed on whatever shifted into the
		// old number)
		if m.selectedLine > 0 {
			m.selectedLine = comment.RelocateLine(oldContent, fresh.Content, m.selectedLine)
		}
		if m.rangeActive {
			m.rangeStartLine = comment.RelocateLine(oldContent, fresh.Content, m.rangeStartLine)
			m.rangeEndLine = comment.RelocateLine(oldContent, fresh.Content, m.rangeEndLine)
		}
		// Re-derive everything positioned against the old content
		m.documentSections = markdown.ParseDocument(fresh.Content)
		m.refsByLine = buildRefMap(fresh.Content, m.filename)
		m.refreshDocumentPane()
	}
	m.clampSelectedComment(len(m.visibleComments()))
	m.commentViewport.SetContent(m.renderComments())
}

// View renders the UI based on current mode. Bubbletea v2 returns a tea.View
// struct; the alt-screen flag lives here now instead of a program option.
func (m Model) View() tea.View {
	v := tea.NewView(m.viewContent())
	v.AltScreen = true
	return v
}

// viewContent renders the current mode's screen content as a string (the
// v1-era View body; mode views and tests stay string-based).
func (m Model) viewContent() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress any key to continue", m.err)
	}

	if d, ok := modeRegistry[m.mode]; ok {
		return d.view(m)
	}
	return "Unknown mode"
}
