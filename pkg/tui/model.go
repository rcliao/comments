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
	"github.com/rcliao/comments/pkg/comment"
)

// Model represents the enhanced TUI application state
type Model struct {
	// View mode
	mode ViewMode

	// File picker
	filePicker filepicker.Model

	// Document state
	doc      *comment.DocumentWithComments
	filename string
	threads  map[string]*comment.Thread

	// UI components
	documentViewport viewport.Model
	commentViewport  viewport.Model
	threadViewport   viewport.Model
	commentInput     textarea.Model

	// Selection state
	selectedLine    int  // For line selection mode
	selectedComment int  // For comment navigation
	selectedThread  *comment.Thread
	showResolved    bool

	// Input state
	author string // User name for comments

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

	// Get author from environment or use default
	author := os.Getenv("USER")
	if author == "" {
		author = "user"
	}

	return Model{
		mode:         ModeFilePicker,
		filePicker:   fp,
		commentInput: ta,
		author:       author,
		showResolved: false,
	}
}

// NewModelWithFile creates a model with a pre-loaded file
func NewModelWithFile(doc *comment.DocumentWithComments, filename string) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment..."
	ta.Focus()

	// Get author from environment or use default
	author := os.Getenv("USER")
	if author == "" {
		author = "user"
	}

	m := Model{
		mode:         ModeBrowse,
		doc:          doc,
		filename:     filename,
		threads:      comment.BuildThreads(doc.Comments),
		commentInput: ta,
		author:       author,
		showResolved: false,
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

	// Split screen: 60% for document, 40% for comments/thread
	docWidth := int(float64(m.width) * 0.6)
	panelWidth := m.width - docWidth - 4

	if !m.ready {
		m.documentViewport = viewport.New(docWidth, m.height-3)
		m.commentViewport = viewport.New(panelWidth, m.height-3)
		m.threadViewport = viewport.New(m.width-4, m.height-3)

		if m.doc != nil {
			m.documentViewport.SetContent(m.renderDocument())
			m.commentViewport.SetContent(m.renderComments())
		}
		m.ready = true
	} else {
		m.documentViewport.Width = docWidth
		m.documentViewport.Height = m.height - 3
		m.commentViewport.Width = panelWidth
		m.commentViewport.Height = m.height - 3
		m.threadViewport.Width = m.width - 4
		m.threadViewport.Height = m.height - 3
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
	case ModeReply:
		return m.handleReplyKeys(msg)
	case ModeResolve:
		return m.handleResolveKeys(msg)
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
		// Go back to file picker
		m.mode = ModeFilePicker
		m.doc = nil
		m.filename = ""
		m.ready = false
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "c":
		// Enter line selection mode to add comment
		m.mode = ModeLineSelect
		m.selectedLine = 1
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		return m, nil

	case "j", "down":
		// Navigate comments
		visibleComments := comment.GetVisibleComments(m.doc.Comments, m.showResolved)
		if m.selectedComment < len(visibleComments)-1 {
			m.selectedComment++
			m.commentViewport.SetContent(m.renderComments())
		}
		return m, nil

	case "k", "up":
		if m.selectedComment > 0 {
			m.selectedComment--
			m.commentViewport.SetContent(m.renderComments())
		}
		return m, nil

	case "enter":
		// Expand selected comment thread
		visibleComments := comment.GetVisibleComments(m.doc.Comments, m.showResolved)
		if len(visibleComments) > 0 && m.selectedComment < len(visibleComments) {
			selectedComment := visibleComments[m.selectedComment]
			if thread, ok := m.threads[selectedComment.ThreadID]; ok {
				m.selectedThread = thread
				m.mode = ModeThreadView
				m.threadViewport.SetContent(m.renderThread())
				return m, nil
			}
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
	switch msg.String() {
	case "esc":
		// Cancel line selection
		m.mode = ModeBrowse
		m.documentViewport.SetContent(m.renderDocument())
		return m, nil

	case "j", "down":
		// Move cursor down
		lines := strings.Split(m.doc.Content, "\n")
		if m.selectedLine < len(lines) {
			m.selectedLine++
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
		}
		return m, nil

	case "k", "up":
		// Move cursor up
		if m.selectedLine > 1 {
			m.selectedLine--
			m.documentViewport.SetContent(m.renderDocumentWithCursor())
		}
		return m, nil

	case "c", "enter":
		// Trigger add comment modal
		m.mode = ModeAddComment
		m.commentInput.Reset()
		m.commentInput.Focus()
		return m, textarea.Blink
	}

	return m, nil
}

// handleAddCommentKeys handles keys in add comment mode
func (m Model) handleAddCommentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel comment creation
		m.mode = ModeLineSelect
		m.commentInput.Reset()
		return m, nil

	case "ctrl+s":
		// Save comment
		text := strings.TrimSpace(m.commentInput.Value())
		if text == "" {
			// Empty comment, just cancel
			m.mode = ModeLineSelect
			m.commentInput.Reset()
			return m, nil
		}

		// Create new comment
		newComment := comment.NewComment(m.author, m.selectedLine, text)
		m.doc.Comments = append(m.doc.Comments, newComment)
		m.doc.Positions[newComment.ID] = comment.Position{Line: m.selectedLine}

		// Rebuild threads
		m.threads = comment.BuildThreads(m.doc.Comments)

		// Save to file
		if err := m.saveDocument(); err != nil {
			m.err = err
			return m, nil
		}

		// Refresh views
		m.documentViewport.SetContent(m.renderDocumentWithCursor())
		m.commentViewport.SetContent(m.renderComments())

		// Return to line select mode
		m.mode = ModeLineSelect
		m.commentInput.Reset()
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
		// Go back to browse mode
		m.mode = ModeBrowse
		m.selectedThread = nil
		return m, nil

	case "q":
		// Go back to file picker
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

	case "x":
		// Enter resolve mode
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

		// Create reply comment
		reply := comment.NewReply(m.author, m.selectedThread.ID, text)
		m.doc.Comments = append(m.doc.Comments, reply)
		m.doc.Positions[reply.ID] = comment.Position{Line: m.selectedThread.Line}

		// Rebuild threads
		m.threads = comment.BuildThreads(m.doc.Comments)
		m.selectedThread = m.threads[m.selectedThread.ID]

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
		comment.ResolveThread(m.doc.Comments, m.selectedThread.ID)

		// Rebuild threads
		m.threads = comment.BuildThreads(m.doc.Comments)

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

// updateByMode delegates updates to mode-specific logic
func (m Model) updateByMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
	case ModeFilePicker:
		m.filePicker, cmd = m.filePicker.Update(msg)
	case ModeBrowse, ModeLineSelect:
		m.documentViewport, cmd = m.documentViewport.Update(msg)
	case ModeAddComment, ModeReply:
		m.commentInput, cmd = m.commentInput.Update(msg)
	case ModeThreadView:
		m.threadViewport, cmd = m.threadViewport.Update(msg)
	}

	return m, cmd
}

// loadFile loads a markdown file and transitions to browse mode
func (m Model) loadFile(path string) (tea.Model, tea.Cmd) {
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		m.err = err
		return m, nil
	}

	// Parse document
	parser := comment.NewParser()
	doc, err := parser.Parse(string(content))
	if err != nil {
		m.err = err
		return m, nil
	}

	// Update model
	m.doc = doc
	m.filename = path
	m.threads = comment.BuildThreads(doc.Comments)
	m.mode = ModeBrowse
	m.selectedComment = 0
	m.ready = false

	// If we have dimensions, initialize viewports now
	if m.width > 0 && m.height > 0 {
		m.handleResize()
	}

	return m, nil
}

// saveDocument saves the current document back to file
func (m *Model) saveDocument() error {
	serializer := comment.NewSerializer()
	updatedContent, err := serializer.Serialize(m.doc.Content, m.doc.Comments, m.doc.Positions)
	if err != nil {
		return fmt.Errorf("serializing document: %w", err)
	}

	if err := os.WriteFile(m.filename, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
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
	case ModeAddComment:
		return m.viewAddComment()
	case ModeThreadView:
		return m.viewThread()
	case ModeReply:
		return m.viewReply()
	case ModeResolve:
		return m.viewResolve()
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
	title := titleStyle.Render(fmt.Sprintf("📄 %s - Mode: %s", m.filename, modeStr))

	var helpText string
	if m.mode == ModeLineSelect {
		helpText = "j/k: move cursor • c/Enter: add comment • Esc: cancel"
	} else {
		helpText = "j/k: navigate • c: comment • Enter: expand • R: toggle resolved • q: back"
	}
	help := helpStyle.Render(helpText)

	// Layout: document on left, comments on right
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.documentViewport.View(),
		commentPanelStyle.Render(m.commentViewport.View()),
	)

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

	// Modal overlay for comment input
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Render(fmt.Sprintf("Add Comment at Line %d", m.selectedLine))

	modalHelp := helpStyle.Render("Ctrl+S: save • Esc: cancel")

	modal := modalOverlayStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			m.commentInput.View(),
			"",
			modalHelp,
		),
	)

	// Position modal over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-3,
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
			m.height-3,
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
	help := helpStyle.Render("r: reply • x: resolve • Esc: back • q: file picker")

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

	// Modal overlay for reply input
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Render("Reply to Thread")

	modalHelp := helpStyle.Render("Ctrl+S: save • Esc: cancel")

	modal := modalOverlayStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			modalTitle,
			"",
			m.commentInput.View(),
			"",
			modalHelp,
		),
	)

	// Position modal over content (centered)
	positioned := lipgloss.Place(
		m.width,
		m.height-3,
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
			m.height-3,
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
		Foreground(lipgloss.Color("170")).
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
		m.height-3,
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
			m.height-3,
			lipgloss.Left,
			lipgloss.Top,
			threadContent,
		),
		positioned,
	)
}
