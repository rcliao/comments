package comment

import (
	"fmt"
	"time"
)

// InboxItem is one thread needing attention, with the reasons it surfaced.
type InboxItem struct {
	File      string      `json:"file"`
	Reasons   []string    `json:"reasons"` // new_reply and/or blocking
	Thread    CommentView `json:"thread"`
	LastReply *InboxReply `json:"last_reply,omitempty"`
}

// InboxReply summarizes the newest reply on a thread.
type InboxReply struct {
	Author    string `json:"author"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

// BuildInbox returns unresolved threads with replies newer than since (or any
// replies when since is zero), plus every unresolved blocking thread — the
// one-call attention view. absPath may be a file or a directory.
func BuildInbox(absPath string, since time.Time) ([]InboxItem, error) {
	targets, err := FindGateTargets(absPath)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no markdown files with comment sidecars found under %s", absPath)
	}

	items := []InboxItem{}
	for _, file := range targets {
		doc, _, err := LoadDocument(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		for _, thread := range doc.Threads {
			if thread.Resolved {
				continue
			}
			reasons := []string{}
			last := NewestReply(thread)
			if last != nil && (since.IsZero() || last.Timestamp.After(since)) {
				reasons = append(reasons, "new_reply")
			}
			if thread.Blocking {
				reasons = append(reasons, "blocking")
			}
			if len(reasons) == 0 {
				continue
			}
			item := InboxItem{
				File:    file,
				Reasons: reasons,
				Thread:  NewCommentView(thread),
			}
			if last != nil {
				item.LastReply = &InboxReply{
					Author:    last.Author,
					Text:      last.Text,
					Timestamp: last.Timestamp.Format(time.RFC3339),
				}
			}
			items = append(items, item)
		}
	}
	return items, nil
}

// NewestReply returns the reply with the newest timestamp anywhere in the
// thread's nested reply tree, or nil when the thread has no replies.
func NewestReply(thread *Comment) *Comment {
	var newest *Comment
	var walk func(replies []*Comment)
	walk = func(replies []*Comment) {
		for _, r := range replies {
			if newest == nil || r.Timestamp.After(newest.Timestamp) {
				newest = r
			}
			walk(r.Replies)
		}
	}
	walk(thread.Replies)
	return newest
}
