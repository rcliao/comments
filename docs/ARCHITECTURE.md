# Architecture

**Status:** Current · **Last reviewed:** 2026-08-25 · **Sidecar format:** 2.0 ·
**Anchor behavior:** v2.1 design

`comments` is a local-first review system for markdown. The markdown remains
the content artifact; a neighboring JSON sidecar carries threads, suggestions,
template identity, hashes, and review records. Humans work primarily in the
TUI or local browser workspace, while scripts and agents use the CLI or MCP
server over the same core logic.

## System shape

```text
                         ┌─────────────────────┐
                         │    markdown file    │
                         │       doc.md        │
                         └──────────┬──────────┘
                                    │
       ┌─────────────┐       ┌──────▼──────┐       ┌─────────────┐
human ─► TUI adapter ├───────►             ◄───────┤ CLI adapter ◄─ scripts
       │  pkg/tui    │       │ pkg/comment │       │ cmd/comments│
       └─────────────┘       │ core logic  │       └─────────────┘
                             │             │
       ┌─────────────┐       └──────┬──────┘
agent ─► MCP adapter ├──────────────┘
       │   pkg/mcp   │              │
       └─────────────┘       ┌──────▼──────────────┐
                             │ doc.md.comments.json│
                             └─────────────────────┘

human ─► local web adapter ────────► pkg/comment
         pkg/webreview
```

The dependency direction is intentional:

- `pkg/comment` owns storage, threads, suggestions, anchors, templates, gates,
  citations, review records, inboxes, and watch snapshots.
- `pkg/markdown` parses headings and local references without depending on a
  user interface.
- `pkg/tui`, `pkg/webreview`, `pkg/mcp`, and `cmd/comments` translate input and
  output. Shared behavior does not live in an adapter.

This rule prevents an MCP-only capability or guard. New behavior starts in
`pkg/comment`, then both public adapters expose it.

## Storage model

For `doc.md`, collaboration state is stored at `doc.md.comments.json`:

```json
{
  "version": "2.0",
  "documentHash": "sha256...",
  "lastValidated": "2026-08-12T18:30:00Z",
  "template": "design-doc",
  "reviews": [
    {
      "author": "eric",
      "timestamp": "2026-08-12T18:30:00Z",
      "decision": "approved",
      "note": "Cache contract settled"
    }
  ],
  "threads": []
}
```

`pkg/comment.StorageFormat` is the wire envelope. `DocumentWithComments` is the
in-memory form and also carries the markdown content. Public JSON responses use
the canonical snake-case `CommentView` and `DocumentView` types rather than the
sidecar's historical field casing.

### Comment and thread model

A root `Comment` contains:

- identity: short random base36 `ID`, author, timestamp;
- content: text and optional `Q`, `S`, `B`, `T`, or `E` type;
- location: line, computed section path, and a content anchor;
- state: resolved, blocking, lifecycle status, priority, orphan details;
- nested `Replies`;
- optional suggestion fields: start/end line, original/proposed text, and
  nullable accepted state.

Replies nest recursively; there is no separate thread or parent table. Root
IDs are the normal write address. Read views include `parent_thread_id` on
nested replies so flattened consumers can route back to the root.

`Accepted == nil` means a pending suggestion, `true` accepted, and `false`
rejected. Suggestions are line-range replacements; a single-line edit is a
range whose start and end match.

### Review records and templates

`ReviewRecord` stores an author, timestamp, decision, and optional note.
Decisions are:

- `approved`;
- `changes_requested`;
- `commented`, a reply-only pass that hands the turn back without changing the
  gate outcome.

A verdict (`approved` or `changes_requested`) also stores the reviewed content
as the reviewer's **review baseline** at
`<docdir>/.comments/baselines/<doc>.<author>.md` — one file per document per
reviewer, latest verdict only, in the same gitignored local-state directory as
the TUI view state. A `commented` pass does not touch it. Both signoff writers
(TUI verdict, `comments signoff`) call `SaveReviewBaseline` after the record
lands; the write is best-effort so a baseline failure never reports a landed
signoff as failed. Readers diff the current document against it
(`comment.ChangedSince`: line-level LCS, then innermost-section rollup) to
answer "what moved since my last verdict". Edited lines are marked directly;
a pure deletion marks the line before the gap (so the blame stays in the
section that lost content), and a removal beside an edit counts once. The TUI tints changed line numbers in the gutter (or shows a bar
column when numbers are hidden); `comments status --author <reviewer>` and
MCP `comments_status {reviewer}` report `changed_lines`, `deletions` and
`changed_sections` (omitted entirely when no baseline exists, so absence means
"never signed off", not "unchanged").

The recorded template name makes structural validation durable. Once `seed`
records a template, later `validate` and `gate` calls can load it without a
flag. Built-ins are embedded from `pkg/comment/templates/`; project-specific
templates are discovered by walking upward for `.comments/templates/`.

## Read and write invariants

The split between markdown and sidecar writes prevents lost updates:

1. `LoadFromSidecar` reads the markdown and sidecar, migrates in memory, runs
   anchor validation, and returns a `LoadReport`. It does not write.
2. `LoadDocument` is the shared surface prelude. If validation changed anchor
   state, it persists only the revalidated sidecar.
3. `SaveToSidecar` atomically writes only JSON. A reply, resolve, signoff, or
   anchor migration must never rewrite markdown from a stale session.
4. `SaveDocumentContent` writes markdown atomically and is called only by
   content-changing paths such as accepting a suggestion.

A hash mismatch is a staleness signal, not a reason to delete or archive the
sidecar. The loader revalidates anchors and preserves unresolved history.

## Anchoring and edit behavior

Comments retain a line for display and a content `Anchor` for recovery. On a
document change, the re-anchor cascade is:

1. exact content at the stored position;
2. exact selected-text search;
3. normalized whitespace/case search, labeled `fuzzy`;
4. section-path fallback, labeled `section-level`;
5. orphan with the original line and reason preserved.

Agents that know how their edit moved content call `reanchor` with explicit
comment-to-line or comment-to-section moves. A declared move is ground truth:
it captures a new anchor and clears orphan state. The automatic cascade is a
safety net, not a substitute for known edit mappings.

Accepting a suggestion updates affected positions in the same transaction.
`OriginalText`, when present, protects against applying a range to content that
has changed underneath it.

## Markdown and reference model

`pkg/markdown` builds an ATX-heading tree with stable section paths and line
ranges. Fenced code blocks do not create headings. Section addressing supports
the full `Parent > Child` path and descendant-aware filters.

The reference parser recognizes:

- `path/to/file.go:42` and line ranges;
- local markdown links such as `[design](design.md#decision)`;
- `thread:c7f3k` and `thread:path.md#c7f3k`.

Code fences are skipped except for comment trails, where schema examples place
evidence such as `// pkg/comment/types.go:140`. Citation validation reports
missing files, out-of-range lines, and ambiguous bare filenames. The TUI uses
the same references for `f` peek and `$EDITOR` handoff.

## Templates, gates, and authority

Templates are writing and review guardrails, not prose generators. They can
enforce required section order, word caps, minimum subsections, ambiguity
markers, prose-shape rules, citations, per-section review criteria, and
`zone: human` ownership.

The gate remains intentionally mechanical:

- unresolved blocking root threads fail;
- template violations fail when a template is selected or recorded;
- strict mode additionally fails on unresolved non-blocking threads and
  pending suggestions;
- semantic correctness stays with human/reviewer threads rather than an
  opaque model score.

Human zones are enforced by actor, not by surface. `COMMENTS_ACTOR` is the
explicit override; otherwise a real terminal means human and redirected output
means agent. CLI and MCP both call `GuardZoneResolve`, so an agent cannot bypass
the rule by switching interfaces. The TUI is a human surface by construction.

## TUI architecture

The TUI uses Bubbletea v2 and Lipgloss v2 with an Elm-style `Model`, pure views,
and mode-specific key handlers. `pkg/tui/registry.go` is the single mapping from
each mode to its key handler, view, and optional component update route.

The document remains visible during review:

- the comment list occupies the right sidebar;
- opening a thread replaces that sidebar with a thread panel;
- the reply composer docks inside the thread panel;
- dialogs and reference peeks composite over the live document;
- only the file picker legitimately owns the full screen.

Screen chrome is three rows: the title bar, the review rail, and the hint bar.
`chromeRows` in `pkg/tui/model.go` is the single definition; `contentHeight()`
and `contentTop()` derive every viewport and the thread panel from it.

The review rail states what belongs to the document rather than to any one
thread: the gate decision, thread counts, and anchor health. It derives from
`comment.EvaluateGate`, so the rail and the verdict dialog cannot disagree
about whether the document passes. `comment.DocumentAnchorHealth` counts the
unresolved threads whose anchors re-located below exact confidence — reported
once on the rail rather than per thread, where it competed with comment text
for sidebar columns.

Rendering preserves source-line identity so anchors and the gutter remain
truthful. Markdown markers are styled in place; fenced code uses Chroma; custom
DBML and minimal Mermaid lexers live in-repo. ANSI-stripped content is normally
byte-identical to source. Aligned markdown tables are the documented exception:
display-only padding changes bytes while preserving one source line per row.

View state under `.comments/` is local, ignored runtime state and must not be
committed.

## CLI and MCP surfaces

The CLI router is `cmd/comments/main.go`; `comments help` is its current command
catalog. The MCP server registers 20 tools and two resources from
`pkg/mcp/server.go`. It covers thread reads/writes, batch operations,
suggestions, templates, gate/review coordination, inbox/status, and explicit
re-anchoring.

The MCP document and thread resources are read views. Long waits use
`comments_request_review`; durable non-blocking waits return a timestamp handle
consumed by `comments_check_review`. Without MCP, `comments watch --until
signoff` observes the same sidecar review record.

## Local web review surface

`comments serve <file-or-dir>` mounts `pkg/webreview` on a loopback listener.
It is a human adapter over the same `pkg/comment` operations as the TUI: add,
reply, resolve/reopen, suggestion accept/reject, and verdict records. Goldmark
renders GFM without enabling raw HTML; a separate source view keeps line
anchors exact. Approved and changes-requested verdicts also update the same
per-reviewer baseline used by the TUI and `comments signoff`.

The renderer wraps each top-level Goldmark block with its source-line range.
The client assigns every root thread to the containing (or nearest) block and
draws a compact comment bubble on that block's right edge; source mode uses the
exact line directly. Open, blocking, and resolved states have distinct bubble
treatments, and either gutter focuses the matching thread card without changing
storage state. Hover/focus linkage is bidirectional: document anchors highlight
their visible thread cards, while a thread card highlights both its rendered
block and exact source row. The linkage is entirely client-side presentation.

The reviewer identity is a workspace-level browser preference, initialized
from the server's `--author` value and sent explicitly with every new thread,
reply, and verdict. The server trims it and enforces an 80-character limit
before calling core operations. Theme is also client-only: the first visit
follows `prefers-color-scheme`, while an explicit light/dark choice is retained
in local storage. Neither preference changes the document sidecar schema.

The initial URL contains 256 bits of random capability token. A successful
bootstrap exchanges it for an HttpOnly, SameSite=Strict cookie and redirects
to a token-free URL. The handler pins accepted Host values to the listener,
checks mutation origins, sends a restrictive CSP, and never accepts a
non-loopback CLI address. Directory mode resolves every document under the
selected root and addresses it by an enumerated relative ID, so request input
cannot traverse to arbitrary files.

Each state response includes a revision over both markdown and sidecar bytes.
Mutations lock per document, compare the client revision under that lock, and
return HTTP 409 plus refreshed state on mismatch. A lightweight SSE stream
announces external file changes; the client then refetches canonical state.
This makes a browser session safe alongside CLI, MCP, or TUI writes without
introducing a second storage system.

## Concurrency and consistency

There is no central multi-process server or lock manager. The sidecar is the shared event
bus, and writes use temporary-file-plus-rename replacement. Before every TUI
mutation, the model refreshes from disk so an open session does not overwrite
agent changes with an old in-memory copy. Suggestion decisions queue in the TUI
and apply together at verdict.

Within one `comments serve` process, mutations are serialized by document and
guarded by the composite revision described above. That closes browser
lost-update races, but it is not a cross-process distributed lock.

This is last-writer-wins per action, not real-time multi-user collaboration.
Network sync, comment edit history, and cross-machine conflict resolution are
outside the current architecture.

## Testing and development boundaries

The repository tests pure logic heavily and pins adapter parity with CLI/MCP
tests. TUI coverage includes render tests, mode dispatch, compositor behavior,
thread panels, reference peeks, and Bubbletea integration tests. The required
gate is `./scripts/ci.sh`, which runs formatting, build, vet, race tests, lint,
and the end-to-end review-flow smoke test.

Key locations:

```text
cmd/comments/              CLI routing and formatting
pkg/comment/               Core domain, storage, templates, gates, anchors
pkg/comment/templates/     Embedded template YAML
pkg/markdown/              Heading and reference parsing
pkg/mcp/                   MCP adapters and resources
pkg/tui/                   Bubbletea review UI
pkg/webreview/             Local HTTP review adapter and embedded UI
skills/review-comments/    Agent workflow
scripts/eval/              Template/eval harness and logs
docs/examples/             Maintained template examples
```

The module currently targets Go 1.25 and uses `charm.land/bubbletea/v2`,
`charm.land/lipgloss/v2`, the Model Context Protocol Go SDK, Chroma, and YAML
v3; Goldmark powers safe browser rendering. Exact versions live in `go.mod`.

## Durable design constraints

These constraints should survive refactors unless a new design record replaces
them:

- markdown and collaboration state stay separate;
- sidecar-only actions never rewrite markdown;
- anchors preserve history and degrade to orphan rather than silent deletion;
- adapters share core behavior and guards;
- the review gate blocks on explicit state, not unreviewable model judgment;
- line identity remains truthful in the TUI;
- a TUI verdict and non-interactive signoff produce the same review record.

Historical rationale and active proposals are indexed in
[docs/README.md](README.md). User workflows live in [USAGE.md](../USAGE.md),
and implementation conventions live in [CLAUDE.md](../CLAUDE.md) and
[pkg/tui/CLAUDE.md](../pkg/tui/CLAUDE.md).
