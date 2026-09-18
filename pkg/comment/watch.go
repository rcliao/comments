package comment

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"
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

// WatchOptions tunes Watch.
type WatchOptions struct {
	Interval time.Duration // poll interval; 0 means one second
	Until    string        // comma-separated event types that end the watch
}

// Watch polls the sidecars under target and hands every review-state change to
// emit, in order. It returns nil once an event matches opts.Until, when emit
// reports stop, or when ctx ends. The sidecar is the shared event bus — every
// writer (TUI, web, CLI, MCP) persists there — so one loop observes them all,
// and both surfaces wait on a review the same way.
func Watch(ctx context.Context, target string, opts WatchOptions, emit func(WatchEvent) (stop bool)) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = time.Second
	}
	type watched struct {
		mtime time.Time
		snap  WatchSnapshot
	}
	state := map[string]watched{}
	for {
		files, err := FindGateTargets(target)
		if err != nil {
			return err
		}
		for _, file := range files {
			info, err := os.Stat(GetSidecarPath(file))
			if err != nil {
				continue
			}
			prev, seen := state[file]
			if seen && !info.ModTime().After(prev.mtime) {
				continue
			}
			snap := TakeSnapshot(file)
			if seen {
				for _, e := range DiffSnapshots(file, prev.snap, snap) {
					if emit(e) || MatchesUntil(e.Event, opts.Until) {
						return nil
					}
				}
			}
			state[file] = watched{mtime: info.ModTime(), snap: snap}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
