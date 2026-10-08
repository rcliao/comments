package comment

import (
	"fmt"
	"sort"
	"time"
)

// InboxItem is one thread needing attention, with the reasons it surfaced.
type InboxItem struct {
	File string `json:"file"`
	// Reasons, most urgent first: blocking, suggestion_rejected (the human said
	// no — rework or close it), new_reply (a reply newer than since), new_thread
	// (opened after since), or unresolved (open, no news).
	Reasons   []string    `json:"reasons"`
	Thread    CommentView `json:"thread"`
	LastReply *InboxReply `json:"last_reply,omitempty"`
	Context   []string    `json:"context,omitempty"`
}

// InboxReply summarizes the newest reply on a thread.
type InboxReply struct {
	Author    string `json:"author"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}

// InboxChanges is what moved since a reviewer's last verdict. Absent when the
// reviewer has never signed off: "no key" means no baseline, not "unchanged".
type InboxChanges struct {
	Reviewer string   `json:"reviewer"`
	Lines    int      `json:"changed_lines"`
	Deleted  int      `json:"deletions"`
	Sections []string `json:"changed_sections"`
}

// InboxFile is the per-document state behind the inbox's decision.
type InboxFile struct {
	File               string        `json:"file"`
	Decision           string        `json:"decision"`
	Template           string        `json:"template,omitempty"`
	Violations         []Violation   `json:"violations,omitempty"`
	StructureUnchecked bool          `json:"structure_unchecked,omitempty"`
	LastReview         *ReviewRecord `json:"last_review,omitempty"`
	Orphaned           int           `json:"orphaned"`
	// Stale reports that the document changed since the sidecar was written, so
	// anchors were just re-validated: check Orphaned and reanchor what moved.
	Stale   bool          `json:"is_stale,omitempty"`
	Changes *InboxChanges `json:"changes,omitempty"`
	// Living is set for a living doc: its phase and Now's first line.
	Living *LivingState `json:"living,omitempty"`
}

// Inbox is the agent's single read: everything that needs attention, and the
// verdict it is working toward. It exists so an iterating agent has one path —
// the decision, every open thread, pending suggestions, structure violations,
// displaced anchors and what changed since the last verdict used to be spread
// over gate, list, status and inbox, and agents read a different subset each.
type Inbox struct {
	Since string `json:"since,omitempty"`
	// Decision is the gate's: approved only when no blocking thread is open and
	// every document conforms to its template. Done means approved AND empty.
	Decision           string       `json:"decision"`
	Count              int          `json:"count"`
	Items              []InboxItem  `json:"items"`
	PendingSuggestions []GateThread `json:"pending_suggestions"`
	Files              []InboxFile  `json:"files"`
}

// InboxOptions tunes BuildInbox. The zero value is a full, unfiltered inbox.
type InboxOptions struct {
	// Since marks activity as news: replies and threads newer than it get the
	// new_reply / new_thread reason. It never hides a thread.
	Since time.Time
	// Reviewer selects whose last-verdict baseline Changes diffs against;
	// empty means the author of each document's latest review.
	Reviewer string
	// ContextSize is the document lines shown each side of a thread.
	ContextSize int
}

// BuildInbox assembles the attention view for a file or a directory. Every
// unresolved thread is listed — blocking first, then news, then the rest — so a
// non-blocking comment with no replies is never invisible to the agent.
func BuildInbox(absPath string, opts InboxOptions) (*Inbox, error) {
	files, err := FindGateTargets(absPath)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no markdown files with comment sidecars found under %s", absPath)
	}

	report := &GateReport{Decision: DecisionApproved}
	inbox := &Inbox{Items: []InboxItem{}, PendingSuggestions: []GateThread{}, Files: []InboxFile{}}
	if !opts.Since.IsZero() {
		inbox.Since = opts.Since.Format(time.RFC3339)
	}

	for _, path := range files {
		doc, load, err := LoadDocument(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}
		fr, err := gateFileReport(path, doc, false, "", opts.ContextSize)
		if err != nil {
			return nil, err
		}
		report.add(fr)
		contextOf := map[string][]string{}
		for _, group := range [][]GateThread{fr.Blocking, fr.NonBlocking, fr.PendingSuggestions} {
			for _, gt := range group {
				contextOf[gt.ID] = gt.Context
			}
		}

		for _, thread := range doc.Threads {
			if thread.Resolved {
				continue
			}
			last := NewestReply(thread)
			news := last != nil && (opts.Since.IsZero() || last.Timestamp.After(opts.Since))
			// A pending suggestion is the human's to decide, so it lists under
			// PendingSuggestions; an accepted one is finished. Either is an item
			// only once someone replies. A REJECTED one is feedback the agent
			// must see: neither human surface resolves it, so dropping it here
			// made the human's "no" invisible.
			rejected := thread.IsSuggestion && thread.Accepted != nil && !*thread.Accepted
			if thread.IsSuggestion && !rejected && !news {
				continue
			}
			reasons := []string{}
			if rejected {
				reasons = append(reasons, "suggestion_rejected")
			}
			if thread.Blocking {
				reasons = append(reasons, "blocking")
			}
			if news {
				reasons = append(reasons, "new_reply")
			}
			if !opts.Since.IsZero() && thread.Timestamp.After(opts.Since) {
				reasons = append(reasons, "new_thread")
			}
			if len(reasons) == 0 {
				reasons = append(reasons, "unresolved")
			}
			item := InboxItem{File: fr.File, Reasons: reasons, Thread: NewCommentView(thread), Context: contextOf[thread.ID]}
			if last != nil {
				item.LastReply = &InboxReply{Author: last.Author, Text: last.Text, Timestamp: last.Timestamp.Format(time.RFC3339)}
			}
			inbox.Items = append(inbox.Items, item)
		}
		inbox.PendingSuggestions = append(inbox.PendingSuggestions, fr.PendingSuggestions...)

		file := InboxFile{
			File: fr.File, Decision: fr.Decision, Template: fr.Template, Violations: fr.Violations,
			StructureUnchecked: fr.StructureUnchecked, LastReview: fr.LastReview,
			Stale: load != nil && load.Stale,
		}
		if st, ok := ReadLivingState(doc.Content); ok {
			file.Living = &st
		}
		for _, c := range doc.GetAllComments() {
			if c.Status == "orphaned" {
				file.Orphaned++
			}
		}
		reviewer := opts.Reviewer
		if reviewer == "" && fr.LastReview != nil {
			reviewer = fr.LastReview.Author
		}
		if reviewer != "" {
			if cs, ok := ChangedSince(fr.File, reviewer, doc.Content); ok {
				sections := cs.Sections
				if sections == nil {
					sections = []string{}
				}
				file.Changes = &InboxChanges{Reviewer: reviewer, Lines: cs.Count(), Deleted: cs.Deletions(), Sections: sections}
			}
		}
		inbox.Files = append(inbox.Files, file)
	}

	sort.SliceStable(inbox.Items, func(i, j int) bool {
		return inboxRank(inbox.Items[i].Reasons) < inboxRank(inbox.Items[j].Reasons)
	})
	inbox.Count = len(inbox.Items)
	inbox.Decision = report.Decision
	return inbox, nil
}

// inboxRank orders items by urgency: blocking, then news, then the rest.
func inboxRank(reasons []string) int {
	switch reasons[0] {
	case "blocking":
		return 0
	case "new_reply", "new_thread", "suggestion_rejected":
		return 1
	}
	return 2
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
