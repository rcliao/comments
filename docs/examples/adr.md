# ADR: sidecar JSON files over inline comment annotations

## Context and Problem Statement

Comments on a markdown document need somewhere to live.
Inline annotations (HTML comments, footnote syntax) keep everything in one file but change the document's bytes on every review action — every comment pollutes diffs, breaks word counts, and makes the artifact hostage to its review.
A separate store keeps the artifact clean but must survive the document being edited underneath it.
This decision predates everything else in the tool: storage shape constrains anchoring, review flow, and git ergonomics.

## Considered Options

### Option A — sidecar JSON (`doc.md.comments.json`)

- Pros: the artifact stays byte-clean; review state diffs separately; one file watcher observes every writer (TUI, CLI, MCP); structured data with no markdown-parsing ambiguity.
- Cons: two files travel together; anchors can go stale when the doc changes without the sidecar.

### Option B — inline HTML comments in the document

- Pros: one file; anchors cannot detach from their text.
- Cons: every reply rewrites the artifact; word caps and diffs count review chatter; agents editing the doc must parse around annotations; resolved history bloats the document forever.

## Decision Outcome

Option A: sidecar JSON, with content anchoring (text + context captured at creation) and a re-anchor cascade to absorb document edits.
The stale-anchor weakness became its own subsystem rather than a reason to pollute the artifact.

## Consequences

The document remains a reviewable artifact with honest diffs and word counts.
Review state becomes a shared event bus: `watch` observes sidecar changes from any writer.
The cost is real: anchor migration is a required post-edit step for agents (`comments_reanchor`), and orphaned threads exist as a lifecycle state (pkg/comment/types.go:12-15).
