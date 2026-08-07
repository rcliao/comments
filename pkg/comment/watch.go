package comment

import (
	"encoding/json"
	"os"
	"strings"
)

// WatchEvent is one observed change in a document's review state.
// Emitted as NDJSON by `comments watch`; consumed by agents, notifiers, and UIs.
type WatchEvent struct {
	Event    string `json:"event"` // comment_added, reply_added, thread_resolved, thread_unresolved, suggestion_accepted, suggestion_rejected, signoff, gate_changed
	File     string `json:"file"`
	ID       string `json:"id,omitempty"` // comment/thread ID
	Author   string `json:"author,omitempty"`
	Line     int    `json:"line,omitempty"`
	Text     string `json:"text,omitempty"` // truncated comment text for context
	Blocking bool   `json:"blocking,omitempty"`
	Decision string `json:"decision,omitempty"` // signoff / gate_changed
	Note     string `json:"note,omitempty"`     // signoff: the reviewer's note, if any
}

// MatchesUntil reports whether an event type matches an --until spec: a
// comma-separated list of event names (e.g. "signoff" or "signoff,gate_changed").
// An empty spec never matches. Whitespace around names is ignored.
func MatchesUntil(eventName, untilSpec string) bool {
	if untilSpec == "" {
		return false
	}
	for want := range strings.SplitSeq(untilSpec, ",") {
		if strings.TrimSpace(want) == eventName {
			return true
		}
	}
	return false
}

// threadState is the per-thread snapshot used for diffing
type threadState struct {
	author     string
	line       int
	text       string
	blocking   bool
	resolved   bool
	replyCount int
	accepted   *bool
}

// WatchSnapshot captures a document's review state at one point in time
type WatchSnapshot struct {
	threads      map[string]threadState
	reviews      int
	lastDecision string // decision of newest review
	lastAuthor   string // author of newest review
	lastNote     string // note on newest review (signoff --note / TUI verdict note)
	gate         string
	valid        bool // false when the sidecar was missing/unreadable
}

// TakeSnapshot reads a sidecar RAW: no re-anchoring or validation work, just
// the stored threads/reviews. Keeps polling cheap and strictly side-effect-free.
func TakeSnapshot(mdPath string) WatchSnapshot {
	snap := WatchSnapshot{threads: map[string]threadState{}}
	data, err := os.ReadFile(GetSidecarPath(mdPath))
	if err != nil {
		return snap
	}
	var storage StorageFormat
	if err := json.Unmarshal(data, &storage); err != nil {
		return snap
	}
	snap.valid = true
	for _, t := range storage.Threads {
		snap.threads[t.ID] = threadState{
			author:     t.Author,
			line:       t.Line,
			text:       truncate(t.Text, 80),
			blocking:   t.Blocking,
			resolved:   t.Resolved,
			replyCount: t.CountReplies(),
			accepted:   t.Accepted,
		}
	}
	snap.reviews = len(storage.Reviews)
	if snap.reviews > 0 {
		latest := storage.Reviews[snap.reviews-1]
		snap.lastDecision = latest.Decision
		snap.lastAuthor = latest.Author
		snap.lastNote = latest.Note
	}
	doc := &DocumentWithComments{Threads: storage.Threads, Reviews: storage.Reviews}
	snap.gate = EvaluateGate(doc, false).Decision
	return snap
}

// DiffSnapshots computes the events that occurred between two snapshots
func DiffSnapshots(file string, old, new WatchSnapshot) []WatchEvent {
	events := []WatchEvent{}
	if !new.valid {
		return events
	}

	for id, n := range new.threads {
		o, existed := old.threads[id]
		if !existed {
			if old.valid { // suppress the initial-load flood
				events = append(events, WatchEvent{Event: "comment_added", File: file, ID: id,
					Author: n.author, Line: n.line, Text: n.text, Blocking: n.blocking})
			}
			continue
		}
		if n.replyCount > o.replyCount {
			events = append(events, WatchEvent{Event: "reply_added", File: file, ID: id, Line: n.line, Text: n.text})
		}
		if n.resolved != o.resolved {
			event := "thread_resolved"
			if !n.resolved {
				event = "thread_unresolved"
			}
			events = append(events, WatchEvent{Event: event, File: file, ID: id, Line: n.line, Text: n.text, Blocking: n.blocking})
		}
		if o.accepted == nil && n.accepted != nil {
			event := "suggestion_accepted"
			if !*n.accepted {
				event = "suggestion_rejected"
			}
			events = append(events, WatchEvent{Event: event, File: file, ID: id, Line: n.line})
		}
	}

	if old.valid && new.reviews > old.reviews {
		// Carry the reviewer's note: an agent waiting on --until signoff
		// gets the decision AND the message in one event
		events = append(events, WatchEvent{Event: "signoff", File: file,
			Author: new.lastAuthor, Decision: new.lastDecision, Note: new.lastNote})
	}
	if old.valid && new.gate != old.gate {
		events = append(events, WatchEvent{Event: "gate_changed", File: file, Decision: new.gate})
	}
	return events
}
