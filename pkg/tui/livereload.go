package tui

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/rcliao/comments/pkg/comment"
)

// reloadInterval is how often an open view checks the document and its
// sidecar for writes from elsewhere (the agent, the CLI, another view).
const reloadInterval = time.Second

type reloadTickMsg struct{}

func reloadTick() tea.Cmd {
	return tea.Tick(reloadInterval, func(time.Time) tea.Msg { return reloadTickMsg{} })
}

// diskStamp fingerprints the document and its sidecar by size and mtime.
func diskStamp(path string) string {
	stamp := ""
	for _, p := range []string{path, comment.GetSidecarPath(path)} {
		if info, err := os.Stat(p); err == nil {
			stamp += fmt.Sprintf("%d:%d;", info.Size(), info.ModTime().UnixNano())
		}
	}
	return stamp
}

// handleReloadTick keeps a view left open beside the session live. It only
// reloads in the reading modes, so nothing a human is typing is swapped out.
func (m Model) handleReloadTick() (tea.Model, tea.Cmd) {
	if m.doc == nil || m.filename == "" {
		return m, reloadTick()
	}
	stamp := diskStamp(m.filename)
	if stamp == m.diskStamp {
		return m, reloadTick()
	}
	if m.mode != ModeBrowse && m.mode != ModeThreadView {
		return m, reloadTick() // stamp kept stale: reload once the human is back
	}
	m.diskStamp = stamp
	if _, err := os.Stat(comment.GetSidecarPath(m.filename)); err == nil {
		m.refreshDocFromDisk()
	} else if data, err := os.ReadFile(m.filename); err == nil && string(data) != m.doc.Content {
		// No sidecar: nothing else wrote threads, so only the text moved
		fresh := *m.doc
		fresh.Content = string(data)
		m.applyFresh(&fresh)
	}
	if m.mode == ModeThreadView && m.selectedThread != nil {
		m.refreshThreadPane()
	}
	return m, reloadTick()
}

// markSeen records what the reader saw on a living doc; other docs keep
// their verdict baseline.
func (m *Model) markSeen() {
	if m.doc == nil || m.filename == "" || !comment.IsLivingDoc(m.doc.Content) {
		return
	}
	_ = comment.SaveSeenBaseline(m.filename, m.author, m.doc.Content)
}
