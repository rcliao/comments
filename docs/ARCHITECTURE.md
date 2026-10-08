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
the TUI view state. A `commented` pass does not touch it. The human surfaces
(TUI verdict, web review) save it after the record
lands (`comment.RecordVerdict`); the write is best-effort so a baseline failure never reports a landed
signoff as failed. Readers diff the current document against it
(`comment.ChangedSince`: line-level LCS, then innermost-section rollup) to
answer "what moved since my last verdict". Edited lines are marked directly;
a pure deletion marks the line before the gap (so the blame stays in the
section that lost content), and a removal beside an edit counts once. The TUI tints changed line numbers in the gutter (or shows a bar
column when numbers are hidden); `comments inbox` (MCP `comments_inbox`)
reports `changed_lines`, `deletions` and `changed_sections` per file under
`changes`, against `--reviewer` or the latest reviewer (omitted entirely when no baseline exists, so absence means
"never signed off", not "unchanged").

Template identity is durable without creating review noise. Resolution is:
explicit argument, `comments.template` frontmatter, legacy sidecar, then an
unambiguous bundle collection. Built-ins are embedded from
`pkg/comment/templates/`; project-specific templates and `.comments/bundle.yaml`
are discovered upward from the document.

OKF is the creation default, not a migration requirement. `comments new`
discovers a bundle upward; when none exists, it initializes the standard
`.comments/bundle.yaml` at the repository root with `docs/artifacts` collections.
Existing Markdown and sidecars remain valid and are never moved implicitly.

The three persistence layers have separate authority:

- OKF-compatible frontmatter and folder placement own durable identity,
  provenance, lifecycle, and document relationships;
- Markdown owns the knowledge claim or delivery intent;
- the adjacent sidecar owns anchored discussion, suggestions, review records,
  and the mechanical gate inputs.

`comments context` reads across those layers but does not merge them. Its edges
retain inclusion reasons, review state remains distinguishable from lifecycle
status, and role modes constrain whether bodies, threads, backlinks, or tag
suggestions can enter an agent's working set. The producer config,
`comments.template`, `related`, and the sidecar are Comments extensions rather
than OKF v0.2 fields; [OKF.md](OKF.md) documents that public boundary.

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

Sections may also carry a `tier` (1..n): a reading-depth label, not extra prose.
`Template.ReadingPath()` groups sections by tier with a cumulative word budget,
and `comments template show` prints it, so a one-minute reader and a ten-minute
reader use the same document. `design-doc` leads with a human-written `Pitch`
(tier 1, `zone: human`, written before the agent drafts) that the body must
answer to. Every tier 1-2 section is `zone: human`, so the short read is the
human-owned part; deeper tiers are mostly agent-drafted detail, though
`Unresolved Questions` (3) and `Definition of Done` (4) stay `zone: human`.
Tiers are not validated and need not follow document order. `zone: human`
guards thread resolution only — reserving the Pitch *text* for the human is a
skill rule and a review criterion, not an enforced check.

The gate remains intentionally mechanical:

- unresolved blocking root threads fail;
- template violations fail when a template is selected or recorded;
- strict mode additionally fails on unresolved non-blocking threads and
  pending suggestions — which makes it the end-of-work gate, since every open
  pick is such a thread;
- semantic correctness stays with human/reviewer threads rather than an
  opaque model score.

**Picks** (`Comment.Pick`, set by `add --pick`) record a decision the agent made
alone and proceeds on. A pick is never blocking (add refuses both), so it does
not stop the default gate; `--strict`, the end-of-work gate, fails until the
human settles each one. An `approved` verdict settles them: `AddReviewRecord`
resolves every open pick with no reply from anyone but its author, adding an
"Accepted at approval" reply under the approver's name (`OpenPicks`). Any other
reply is an objection and keeps the pick open. Both human surfaces record
through that one call, and `ReplyToThreads` refuses an agent's resolve on an
open pick, so there is no agent path to settle one. Both surfaces state the
count before approval; the TUI refuses an approval when the open picks changed
after its dialog rendered, and the browser's revision check rejects a stale
verdict.

**Living docs** (`templates/living.yaml`, collection `living`) are the
lighter mode: Now, Why, Plan, Decisions, Explanation, Checks, kept current by
the agent while it builds, with chat as the interface. They have no zones and
no verdict. In `comments view`, `q` records a *seen* baseline
(`SaveSeenBaseline`, the same per-reader file as the verdict baseline) and
quits, so the next open tints what changed since the reader last looked.
Every view polls the doc and sidecar once a second and reloads in browse and
the thread panel only (`pkg/tui/livereload.go`).

**Briefs** are the default artifact (`templates/brief.yaml`): Why, What, Shape
and Checks are `zone: human`, How is the agent's. A verdict on a brief records
`template: brief` and `BriefContractHash` as its intent hash: everything but
the bodies of agent sections (How), with blank-line runs collapsed. Inside
How, a line that renders as a top-level heading (indented ATX, setext) still
counts. It is judged under the embedded template only (`loadBuiltinTemplate`),
so neither frontmatter, a project template nor the working directory can move
the zones. Agent edits to How never stale the approval; any other edit does.
A forged review record in the sidecar is the same exposure plans already have.
Plans keep the older intent hash (Status blocks excluded). The other templates
are labelled legacy and still load and gate.

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
catalog. The MCP server registers two resources and the agent surface as tools
from `pkg/mcp/server.go`.

The surface follows one rule: **one command per purpose, the same on both
surfaces.** Each agent command has an MCP tool of the same name, and each
result is a single type in `pkg/comment` that both adapters marshal
(`GateReport`, `Inbox`, `ThreadList`/`ThreadDetail`, `AddResult`, `ReplyResult`,
`TemplateBrief`, `WatchEvent`), so neither the catalog nor the shapes can drift.

| Purpose | Command and tool | Core entry point |
|---|---|---|
| Create a doc under a template | `new`, `context` (carries the brief) | `CreateBundleDocument`, `BuildDocumentContext` |
| Check a draft | `validate`, `analyze` | `ValidateManagedDocument`, `AnalyzeDocument` |
| Annotate | `add` | `AddComments` |
| Wait for the human | `watch` | `Watch` |
| See what needs attention | `inbox` | `BuildInbox` |
| Look up a thread | `get` | `ListThreads`, `GetThread` |
| Respond, optionally resolving | `reply` | `ReplyToThreads` (carries `GuardZoneResolve`) |
| Propose an edit | `suggest` | `NewSuggestion` |
| Fix anchors after edits | `reanchor` | `ApplyMoves` |

CLI-only by design: `view` and `serve` (the human surfaces), `gate` (exit-code
contract for scripts), `template`, `doctor`, `bundle`, `serve-mcp`, `help`.

**Human decisions have no command.** Accepting or rejecting a suggestion and
recording a verdict happen only inside `view` and `serve`, which call the core
helpers directly (`AcceptSuggestion`, `RejectSuggestion`, `RecordVerdict`). An
end-to-end review (`docs/review-surface-e2e-2026-09-18.md`) showed an agent
rewriting a `zone: human` section by accepting its own suggestion, and recording
an approval under the human's name with `signoff`; removing the commands closes
both without new enforcement. `cmd/comments/parity_test.go` reads the tool
catalog from registration and the command catalog from the dispatch switch and
fails if a tool lacks a twin, a command lacks a twin or a recorded reason, or
one of those decisions becomes reachable again.

`comment.ValidateManagedDocument` is the shared path-aware entry point.
CLI validate/gate and MCP validate/gate all call it, so citation resolution and
structural rule ordering cannot differ by surface; documents inside a bundle
also receive the OKF metadata and collection-type checks. `comments analyze` /
`comments_analyze` adds an advisory manifest: declared question-to-finding
coverage plus research findings cited, explicitly excluded, or uncovered by a
plan. Its `ready` field never changes gate state; semantic claim judgment stays
in review threads.

The MCP document and thread resources are read views. Agents normally hand the
document to the human and listen with `comments watch --until signoff`; the MCP
request/check pair remains a transport-equivalent blocking or durable poll over
the same sidecar review record.

## Local web review surface

`comments serve <file-or-dir>` mounts `pkg/webreview` on a loopback listener.
It is a human adapter over the same `pkg/comment` operations as the TUI: add,
reply, resolve/reopen, suggestion accept/reject, and verdict records. Goldmark
renders GFM without enabling raw HTML; a separate source view keeps line
anchors exact. Approved and changes-requested verdicts also update the same
per-reviewer baseline used by the TUI and the web review.

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

## Claude Code mod (experimental)

`mods/comments-review/` is a Claude Code mod (a plugin of function hooks,
Claude Code 2.1.287+), loaded with `claude --plugin-dir mods/comments-review`
and not yet shipped in the published plugin. It adds no storage and no agent
tool: every read and write goes through the `comments` binary or the
`comments serve` API, so the parity rule and the human-only verdict hold.
The design is `docs/artifacts/plans/plan-contract-loop.md`.

- **Review surface.** `/review-doc <doc>` opens `comments view` in a herdr side
  pane when the session runs inside herdr (`HERDR_ENV`), else an in-Claude pane
  that talks to a `comments serve` child whose token stays in mod memory.
- **Wake-up.** A `comments watch --until signoff` loop started at session start
  turns the human's verdict into a prompt to Claude carrying the gate state.
- **Hand-off.** Claude's own blocking `comments watch <plan> --until signoff`
  on a `plan`-template doc is answered at once: review opens beside the
  session, and until the verdict only read tools run, so ending the turn is the
  agent's only move. Other docs keep the blocking watch.
- **Plan mode.** `ExitPlanMode` is denied while the plan is saved as a comments
  doc and reviewed; after approval with a passing gate, the next call is
  allowed with the reviewed doc as the plan.
- **Locks.** Claude's Edit/Write on a doc under review are refused.
- **Contract gate.** A handed-off plan becomes active; Claude's Edit/Write
  outside it are refused until the last verdict is approved, the gate passes,
  and the approval is `current` (`comments context --for implementation`,
  which accepts plans and briefs). The intent hash leaves out every Status block and blank
  line, so appending progress never makes an approval stale; hashes recorded
  before that rule still count. The check fails closed; only
  the person's own `/review-doc --unlock` (a `composer` origin) bypasses it.
- **Living docs.** Writing a `template: living` doc makes it the session's
  living doc; every other file edit counts as drift until the doc changes
  again. The count shows in the status line and in a `[comments living doc]`
  note after compaction or a restart (path and count only). A living doc is
  never locked by the contract gate, and `/review-doc --park` (person only)
  sets a parked plan aside so its lock lifts. The doc and its drift count are
  kept per working directory across restarts; `/review-doc --done` ends it.
  `./scripts/ci.sh` validates and tests the mod when the `claude` CLI is on
  PATH, and says so loudly when it is not.
- **Contract memory.** The active plan is kept per working directory; on
  compaction the mod keeps exactly one `[comments contract]` note (plan, gate,
  lock state), and a restarted session gets the same note once (a `$.state`
  flag keeps hot reloads, which also fire `session.start`, from repeating it). The note holds
  gate-derived state only, never thread text or plan prose.

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
