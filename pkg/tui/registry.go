package tui

// Mode-descriptor registry: each view mode registers its key handler, view
// renderer, and (optionally) a non-key message router in ONE place — the
// modeRegistry table below. handleKeyPress, View, and updateByMode all
// dispatch through it, so adding a mode never touches those switches.

import tea "charm.land/bubbletea/v2"

// modeDescriptor bundles everything a view mode plugs into the TUI loop.
type modeDescriptor struct {
	// handleKeys processes key presses while the mode is active.
	handleKeys func(Model, tea.KeyMsg) (tea.Model, tea.Cmd)

	// view renders the mode's screen.
	view func(Model) string

	// updateViewport routes non-key messages (mouse wheel, blink ticks,
	// component events) to the mode's active component. nil = static mode:
	// nothing consumes non-key messages.
	updateViewport func(*Model, tea.Msg) tea.Cmd
}

// modeRegistry is the single registration point for view modes. A mode not
// listed here has dead keys and renders "Unknown mode" — see the checklist in
// pkg/tui/CLAUDE.md when adding one.
var modeRegistry = map[ViewMode]modeDescriptor{
	ModeFilePicker: {
		handleKeys:     Model.handleFilePickerKeys,
		view:           Model.viewFilePicker,
		updateViewport: updateFilePicker,
	},
	ModeBrowse: {
		handleKeys:     Model.handleBrowseKeys,
		view:           Model.viewBrowse,
		updateViewport: updateDocumentViewport,
	},
	ModeLineSelect: {
		handleKeys: Model.handleLineSelectKeys,
		view:       Model.viewBrowse,
		// No viewport passthrough: line-select controls scrolling manually
	},
	ModeAddComment: {
		handleKeys:     Model.handleAddCommentKeys,
		view:           Model.viewAddComment,
		updateViewport: updateCommentInput,
	},
	ModeThreadView: {
		handleKeys:     Model.handleThreadViewKeys,
		view:           Model.viewThread,
		updateViewport: updateThreadViewport,
	},
	ModeReply: {
		handleKeys:     Model.handleReplyKeys,
		view:           Model.viewReply,
		updateViewport: updateCommentInput,
	},
	ModeResolve: {
		handleKeys: Model.handleResolveKeys,
		view:       Model.viewResolve,
	},
	ModeAddSuggestion: {
		handleKeys: Model.handleAddSuggestionKeys,
		view:       Model.viewAddSuggestion,
	},
	ModeChooseTarget: {
		handleKeys: Model.handleChooseTargetKeys,
		view:       Model.viewChooseTarget,
	},
	ModeSelectSuggestionType: {
		handleKeys: Model.handleSelectSuggestionTypeKeys,
		view:       Model.viewSelectSuggestionType,
	},
	ModeSelectRange: {
		handleKeys: Model.handleSelectRangeKeys,
		view:       Model.viewSelectRange,
	},
	ModeVerdict: {
		handleKeys: Model.handleVerdictKeys,
		view:       Model.viewVerdict,
	},
	ModeHelp: {
		handleKeys: Model.handleHelpKeys,
		view:       Model.viewHelp,
	},
	ModeTOC: {
		handleKeys: Model.handleTOCKeys,
		view:       Model.viewTOC,
	},
	ModeRefPeek: {
		handleKeys: Model.handleRefPeekKeys,
		view:       Model.viewRefPeek,
	},
}

// updateFilePicker forwards non-key messages to the file picker component
func updateFilePicker(m *Model, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)
	return cmd
}

// updateDocumentViewport forwards non-key messages to the document viewport
func updateDocumentViewport(m *Model, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.documentViewport, cmd = m.documentViewport.Update(msg)
	return cmd
}

// updateCommentInput forwards non-key messages to the comment textarea
func updateCommentInput(m *Model, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return cmd
}

// updateThreadViewport forwards non-key messages to the thread viewport
func updateThreadViewport(m *Model, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.threadViewport, cmd = m.threadViewport.Update(msg)
	return cmd
}
