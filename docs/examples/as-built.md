# As-built: the review gate and signoff loop

## What This Describes

The gate/signoff subsystem as of main @ db63ee5: how review state becomes a machine-readable contract between a human reviewer and a drafting agent.
Deliberately not covered: the TUI's rendering, templates/validation, and the MCP transport — only the review-state loop itself.

## Data Flow

1. An **agent** finishes a draft and requests review — over MCP it blocks in `comments_request_review`; headless it arms `comments watch --until signoff` and waits on the event stream.
2. The **human** reviews in the TUI and exits through the verdict dialog (`q`): the dialog applies queued suggestion decisions, records a ReviewRecord, and quits with the decision's exit code.
3. The **sidecar** is the shared event bus: every writer persists immediately, so the watcher and the blocked MCP call observe the same file (steps 1 and 2 are concurrent — the agent waits while the human reads).
4. The **watcher** diffs sidecar snapshots and emits NDJSON events; a `signoff` event carries author, decision, and the reviewer's note in one payload.
5. The **agent** wakes, reads the inbox FIRST (replies are the payload; the decision is the envelope), then acts: `commented` → iterate; `approved` → proceed after draining; `changes_requested` → fix and re-request.
6. The **gate** answers "may implementation proceed" at any time: exit 0 or 10, derived from blocking threads, never from who asked.

Roles are greppable: verdict in pkg/tui/keys_verdict.go, watcher in pkg/comment/watch.go, gate in pkg/comment/gate.go.

## Data Model

```dbml
Table review_record {
  author string              // pkg/comment/types.go:141
  timestamp timestamp [pk]   // appended, never rewritten
  decision string            // approved | changes_requested | commented
  note string                // the reviewer's message to the agent
}

Table thread {
  id string [pk]             // base36, pkg/comment/types.go:190 lookup
  line int                   // anchor line in the document
  blocking bool              // gate-failing until resolved
  resolved bool
}

Table watch_snapshot {
  file string [pk]           // in-memory only, per watcher run
  reviews int                // count; newest decision cached
  gate string                // derived decision at snapshot time
}

Ref: review_record.timestamp > watch_snapshot.reviews  // count comparison; convention
Ref: thread.blocking > watch_snapshot.gate             // gate derives from open blocking threads
```

## What Persists Where

The sidecar (`doc.md.comments.json`) holds threads and the append-only reviews array; it is the only durable store.
Watch snapshots live in the watcher process and die with it — a restarted watcher re-baselines silently.
The verdict's queued suggestion decisions live in TUI memory until submit; Ctrl+C discards them deliberately.
Nothing else survives restart, and nothing depends on it surviving.

## Known Gaps

- A `commented` signoff exits 0, indistinguishable from `approved` by exit code alone — consumers must read the decision string (pkg/comment/gate.go:12-17). Unowned.
- Watch polls mtime; sub-interval double-writes can coalesce two events into one diff. Accepted, unowned.

## Open Questions

- Should the gate distinguish "never reviewed" from "approved with zero threads"? Both exit 0 today.
