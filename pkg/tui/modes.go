package tui

// ViewMode represents the current view state of the TUI
type ViewMode int

const (
	// ModeFilePicker shows the file browser to select a markdown file
	ModeFilePicker ViewMode = iota

	// ModeBrowse shows the document with comments (read-only navigation)
	ModeBrowse

	// ModeLineSelect allows selecting a line to add a comment
	ModeLineSelect

	// ModeAddComment shows a modal to add a new comment
	ModeAddComment

	// ModeThreadView shows an expanded thread with all replies
	ModeThreadView

	// ModeReply shows a modal to reply to a comment
	ModeReply

	// ModeResolve shows a confirmation dialog to resolve a thread
	ModeResolve

	// ModeReviewSuggestion shows a suggestion with preview and accept/reject options
	ModeReviewSuggestion

	// ModeAddSuggestion shows a modal to add a new suggestion
	ModeAddSuggestion

	// ModeChooseTarget shows a modal for choosing section vs line
	ModeChooseTarget

	// ModeSelectSuggestionType shows a modal for choosing range vs section suggestion
	ModeSelectSuggestionType

	// ModeSelectRange shows visual range selection for multi-line suggestions
	ModeSelectRange
)

// String returns the string representation of the view mode
func (m ViewMode) String() string {
	switch m {
	case ModeFilePicker:
		return "FILE_PICKER"
	case ModeBrowse:
		return "BROWSE"
	case ModeLineSelect:
		return "LINE_SELECT"
	case ModeAddComment:
		return "ADD_COMMENT"
	case ModeThreadView:
		return "THREAD_VIEW"
	case ModeReply:
		return "REPLY"
	case ModeResolve:
		return "RESOLVE"
	case ModeReviewSuggestion:
		return "REVIEW_SUGGESTION"
	case ModeAddSuggestion:
		return "ADD_SUGGESTION"
	case ModeChooseTarget:
		return "CHOOSE_TARGET"
	case ModeSelectSuggestionType:
		return "SELECT_SUGGESTION_TYPE"
	case ModeSelectRange:
		return "SELECT_RANGE"
	default:
		return "UNKNOWN"
	}
}

// IsModal returns true if the mode represents a modal dialog
func (m ViewMode) IsModal() bool {
	return m == ModeAddComment || m == ModeReply || m == ModeResolve || m == ModeReviewSuggestion || m == ModeAddSuggestion || m == ModeChooseTarget || m == ModeSelectSuggestionType
}

// IsInteractive returns true if the mode requires user input
func (m ViewMode) IsInteractive() bool {
	return m == ModeAddComment || m == ModeReply || m == ModeLineSelect || m == ModeFilePicker || m == ModeAddSuggestion
}
