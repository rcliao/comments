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
	default:
		return "UNKNOWN"
	}
}

// IsModal returns true if the mode represents a modal dialog
func (m ViewMode) IsModal() bool {
	return m == ModeAddComment || m == ModeReply || m == ModeResolve
}

// IsInteractive returns true if the mode requires user input
func (m ViewMode) IsInteractive() bool {
	return m == ModeAddComment || m == ModeReply || m == ModeLineSelect || m == ModeFilePicker
}
