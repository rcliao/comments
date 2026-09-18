# Comments CLI usage

This guide covers the current human and agent workflows. `comments help` is the
canonical flag reference shipped with the binary; use it when a flag here and
your installed version differ.

## Quick start

```bash
comments view doc.md
comments serve doc.md --author eric
comments add doc.md --anchor "sentence under review" --author eric --text "Tighten this" --blocking
comments gate doc.md
```

`comments view` is the terminal review surface. Press `q` to open the verdict and
then `a` to approve, `c` to request changes, or `r` to submit a reply-only pass.
All three choices record a review in the sidecar. This and `comments serve` are
the only ways to record a verdict or decide a suggestion: no command does
either, so an agent cannot approve its own document.

`comments serve` provides the same review loop in a browser. Open the one-time
URL printed by the command; it exchanges its random token for a local HttpOnly
session cookie. The server binds to loopback only, watches the markdown and
sidecar for changes, and rejects stale writes rather than overwriting a newer
review action. A directory target provides a document queue for every markdown
file under it that already has a comment sidecar.

## Command map

One command per purpose. Every agent command has an MCP tool of the same name
(`comments inbox` = `comments_inbox`) returning the same JSON.

| Purpose | Command | Who |
|---|---|---|
| Review and decide | `view`, `serve` | human |
| Create a doc under a template | `new`, then `context` (carries the writing brief) | agent |
| Check a draft | `validate` (structure), `analyze` (coverage, advisory) | agent |
| Annotate | `add` — one comment by flags, many with `--json` | agent |
| Wait for the human | `watch --until signoff --since <hand-off time>` | agent |
| See what needs attention | `inbox` | agent |
| Look up a thread, resolved ones included | `get` | agent |
| Respond, optionally closing the thread | `reply [--resolve]` | agent |
| Propose an edit | `suggest` | agent |
| Fix anchors after editing | `reanchor` | agent |
| Exit-code contract for scripts and CI | `gate` | scripts |
| Templates, bundles, diagnostics, integration | `template list\|show`, `bundle index`, `doctor`, `serve-mcp` | maintenance |

Run `comments help` for the complete flag list and examples.

## Browser review workspace

```bash
comments serve doc.md
comments serve docs/ --author eric
comments serve doc.md --addr 127.0.0.1:8080
```

Rendered mode is optimized for reading. Source mode preserves exact line
identity: select any line number to start a thread there. The review rail can
reply, resolve or reopen threads, accept or reject suggestions, and submit an
approve, changes-requested, or reply-only verdict. Changes made by an agent,
the TUI, or another CLI command appear through the live event stream.

Both modes expose a comment gutter. Rendered blocks show a compact comment
bubble on the document's right edge; source rows show a count badge beside the
line number. Blocking and resolved bubbles use distinct treatments. Select any
bubble to highlight the passage and focus its thread in the review rail.
Hovering a commented passage or source row highlights its thread cards; hovering
a thread card highlights the corresponding rendered passage and source row.
Replies use a multiline composer: Enter or Cmd+Enter sends, while Shift+Enter
inserts a newline.

The browser also mirrors the TUI's core keyboard review loop. Use `j/k` (or the
arrow keys) to select threads, `g/G` for first/last, `Enter` to focus the
selected thread, `r` to reply, and `a/x` to accept or reject a suggestion (or
resolve a regular thread). `Esc` closes the reply composer and returns focus to
its thread; a second `Esc` leaves thread focus. `1/2/3` select open, blocking,
or all threads. Press
`c` to enter source line-select mode, move with `j/k`, `g/G`, or `[/]`, and
press `Enter` to compose at the selected line. `s` toggles rendered/source, and
`q` enters verdict mode.

While composing a new comment, `Ctrl+T` cycles type, `Ctrl+P` cycles priority,
`Ctrl+B` toggles blocking, and `Ctrl+S` or `Ctrl/⌘+Enter` saves. A leading
`[Q]`, `[S]`, `[B]`, `[T]`, or `[E]` marker also selects the matching type as
you type; cycling the type updates an existing leading marker.
Press `?` in the workspace for the complete list. Shortcuts are suspended while
typing or while a form dialog is open.

Set **Commenting as** once in the top bar to use the same review name for every
new thread, reply, and verdict across the workspace. The browser remembers that
name. The adjacent theme control switches between light and dark mode, starts
from the operating-system preference, and remembers an explicit choice.
After a verdict is submitted, its decision, reviewer, time, and note remain
visible in the finish-review card; the document header also carries the latest
decision. **Update review** reopens the verdict controls for a later pass.

The server intentionally refuses non-loopback `--addr` values. Its review token
is a capability: do not paste the printed URL into logs or messages. Markdown
HTML is not trusted; raw HTML is omitted by the renderer and the page ships a
restrictive Content Security Policy.

## Targeting document content

Comments accept exactly one target:

```bash
# Preferred for agents: quote a unique line or substring.
comments add doc.md --anchor "The cache is process-local" \
  --author claude --text "What invalidates it?" --type Q

# Stable for named sections.
comments add doc.md --section "Design > Cache" \
  --author claude --text "Add the failure path" --blocking

# Useful when a human already knows the line.
comments add doc.md --line 42 \
  --author eric --text "This is the decision" --priority high
```

`--line`, `--section`, and `--anchor` are mutually exclusive. Section paths use
the full heading hierarchy with ` > ` separators. Anchor text must identify one
line uniquely; ambiguity is reported instead of choosing silently.

`--type Q|S|B|T|E` records a question, suggestion, bug, TODO, or enhancement.
`--priority low|medium|high` controls walkthrough order. `--blocking` keeps the
review gate closed until the root thread is resolved.

Long text flags support `@filename` input:

```bash
comments reply doc.md --thread c7f3k --author claude --text @reply.txt
```

## Reading and replying to threads

```bash
comments inbox doc.md                      # what needs attention, most urgent first
comments inbox doc.md --since 2026-08-12T18:30:00Z --json

comments get doc.md                        # every thread, resolved included
comments get doc.md --unresolved --json
comments get doc.md --thread c7f3k
comments get 'thread:research.md#c7f3k' --from plan.md

comments reply doc.md --thread c7f3k --author claude --text "Applied in the draft" --resolve
```

`inbox` is the one read an iterating agent needs. It returns the gate `decision`,
every unresolved thread (blocking first) with replies and document context,
suggestions awaiting the human, and per file the template `violations`,
`orphaned` anchors, `is_stale`, and `changes` since the reviewer's last verdict.
`--since` flags news as `new_reply` / `new_thread`; it never hides a thread. A
suggestion the human rejected lists as `suggestion_rejected` until it is closed.
Done means `decision: approved` and no items.

`get` is for looking something up. It accepts a document plus `--thread`, or a
thread citation copied directly from prose. `thread:c7f3k` means the citing
document; use `--from` so same-document and relative-path citations resolve
correctly.

`reply --resolve` closes the thread after the reply lands. For an agent it is
refused in a template's `zone: human` section, and a refused resolve posts
nothing: reply without it and leave the resolve to the human.

### Many at once

`add --json` and `reply --json` validate the whole input before writing; an error
names the failing item and nothing is applied. Each comment needs `author`,
`text`, and exactly one of `anchor`, `section`, or `line`.

```json
[
  {
    "anchor": "The cache is process-local",
    "author": "claude",
    "text": "State the invalidation rule",
    "type": "Q",
    "priority": "high",
    "blocking": true
  },
  {
    "section": "Doc Title > Risks",
    "author": "claude",
    "text": "Add the rollback risk",
    "type": "S"
  }
]
```

```bash
comments add doc.md --json comments.json
comments reply doc.md --json replies.json
```

Replies are objects shaped like
`{"thread_id":"c7f3k","author":"claude","text":"Applied","resolve":true}`.
Section paths are full paths from the document title, as `get` prints them.
Pass `--json -` to either command to read from standard input.

## Edit suggestions

Suggestions target a line range, a whole section, or an anchor. With
`--anchor`, the number of lines in `--original` determines the range.

```bash
comments suggest doc.md --anchor "The old first line" \
  --author claude --text "Clarify the contract" \
  --original @old.txt --proposed @new.txt
```

A suggestion is a proposal for the human. There is no `accept` or `reject`
command: in the TUI, `a` and `x` queue accept/reject decisions, and the queue is
applied atomically when a verdict is submitted (discarded by `Ctrl+C`). Accepting
updates the markdown, marks the suggestion accepted, shifts affected positions,
and refreshes the sidecar. A rejected suggestion returns to the agent's inbox.

A suggestion's line range travels with its anchor when the document is edited.
If its target text is removed altogether it is orphaned and can no longer be
applied, rather than replacing whatever now sits on those lines.

## Templates and review gates

Built-ins are `design-doc`, `mini`, `research`, `research-deep`, `plan`, `adr`,
`rfc`, and `as-built`.

```bash
comments template list
comments template show design-doc
comments validate draft.md --template design-doc
comments gate draft.md --json
comments analyze plan.md --against research.md --json
```

Templates define required sections, ordering, word caps, minimum alternatives,
review criteria, citation checks, and human-owned zones. Template identity
resolves in this order: explicit flag, `comments.template` frontmatter, legacy
sidecar, then a bundle collection with exactly one template. Agents post their
own specific self-review callouts with `add` (`--json` for many); generic criterion
threads are intentionally not generated.

### OKF bundles and agent context

OKF is the default for newly created artifacts. On its first run in a project,
`comments new` writes `.comments/bundle.yaml` at the repository root and creates
the standard `docs/artifacts` collections. Edit that committed config to change
the taxonomy or knowledge root. Existing Markdown and sidecars are not moved or
rewritten. Concept files use OKF-compatible frontmatter. `comments.template`
and `related` are Comments producer extensions used for validation and
deterministic navigation; they do not change the OKF v0.2 conformance floor.

```bash
comments new cache-policy --template research-deep --description "Evidence for cache invalidation policy"
comments new cache-policy --template plan --from docs/artifacts/research/cache-policy.md
comments context docs/artifacts/plans/cache-policy.md --for drafting --include-threads
comments bundle index
```

`new` initializes a missing default bundle, selects the folder from the
template, emits required section headings, creates an empty review sidecar, and
refreshes root and collection indexes.
`context` returns explicit frontmatter relations, Markdown links, backlinks,
sources, review state, and up to five tag-based suggestions. Every edge names
why it was included. `coverage-scout` exposes only the Research Question as its
`focus` while forcibly excluding bodies, threads, and draft-derived links;
`evidence-verifier` and review modes do not broaden the working set with tag
suggestions.

A generated concept is self-describing:

```yaml
---
comments:
  template: plan
description: Implementation and verification strategy for cache invalidation.
related:
  - path: ../research/cache-policy.md
    relation: informed_by
status: draft
title: Cache Policy
type: Plan
---
```

OKF v0.2 requires only `type` for a concept. Comments generates the additional
fields because they improve navigation and review. The bundle configuration,
template namespace, relation vocabulary, and sidecar schema belong to Comments,
not the OKF specification. See [docs/OKF.md](docs/OKF.md) for the full boundary,
default folder map, context-mode guarantees, and migration behavior.

Gate results:

- exit `0`: approved;
- exit `10`: changes requested;
- exit `1`: command or input error.

The normal gate fails on unresolved blocking threads and template violations.
`--strict` also fails on any unresolved thread or pending suggestion. A gate on
a directory scans markdown files that have sidecars.

### Research and plan analysis

`analyze` is the deterministic input to an agent research loop, not a semantic
judge and not a second gate:

```bash
comments analyze research.md --json
comments analyze plan.md --against research.md --json
```

For research it returns the numbered questions, finding headings, their `Qn`
mapping, and citation violations even when clean. With `--against`, every
research finding is classified `cited`, `excluded` (a citation under an
explicit non-goal section), or `uncovered`. Citation ranges may cover several
findings; comment trails inside fenced schema examples remain evidence.

`ready: false` always exits 0 because analysis is advisory. Agents fix or
explain its findings through threads; only `gate` and human plan signoff
authorize implementation. CLI `validate`, MCP `comments_validate`, and both
gate surfaces share the same path-aware template and citation validator.

## Waiting for the verdict

```bash
# Block until the human's verdict; prints it with their decision and note.
comments watch doc.md --until signoff --since 2026-08-12T18:30:00Z

# Then read everything that needs attention.
comments inbox docs/ --since 2026-08-12T18:30:00Z --json
```

`watch` emits NDJSON events including `comment_added`, `reply_added`,
`thread_resolved`, `suggestion_accepted`, `signoff`, and `gate_changed`.
`--until` accepts a comma-separated event list.

`watch --until signoff` is the agent handoff: it lets the human review in the TUI
or browser without sending a separate nudge. Pass `--since` with the time of the
hand-off. A watch reports only what changes after its first look, so a human who
reviews before the watch starts would otherwise be missed; with `--since`, a
verdict recorded after that time is returned at once. Over MCP, `comments_watch`
runs the same loop and returns `status: timeout` after `timeout_seconds` — call
it again, with the same `since`.

The gate is mechanical, not proof that a person reviewed the document. A clean
document can have gate decision `approved` before its first verdict. Treat the
signoff event or latest review record as human authorization.

## TUI reference

Press `?` inside `comments view` for the authoritative key list.

| Activity | Keys |
|---|---|
| Move | `j/k`, `Ctrl+D/U`, `g/G`, `]/[`, `n/N` |
| Find | `/` search, `t` table of contents, `f` peek citation, `#` line numbers |
| Threads | `Enter` expand, `r` reply/dive, `Tab` cycle stacked threads, `R` resolved toggle, `P` priority order, `x` resolve |
| Compose | `c` comment, `s` suggest, `Ctrl+S` save, `Ctrl+P/T` priority/type, `Esc` cancel |
| Review | `a/x` queue suggestion decision, `S` sidebar density, `L` line summaries |
| Exit | `q` verdict, `n` add verdict note, `Ctrl+C` quit without verdict |

The citation peek understands `path:line`, local markdown links, and
`thread:` citations. `Enter` from the peek opens `$EDITOR` at the target.

## Anchors and document changes

Sidecars store a SHA-256 hash of the markdown. On a mismatch, loading runs the
re-anchor cascade: exact position, exact text, normalized text, section
fallback, then orphan. It does not archive or discard the sidecar.

After an agent edits a document with comments, explicitly migrate anchors it
knows it displaced:

```bash
comments reanchor doc.md --comment c7f3k --line 58
comments reanchor doc.md --json moves.json --json-out
```

Each batch move is
`{"comment_id":"c7f3k","line":58}` or
`{"comment_id":"c7f3k","section":"Design > Cache"}`.
The load-time cascade remains the safety net for edits without a declared map.

## Storage and environment

Collaboration data lives in `doc.md.comments.json`; the markdown stays clean.
The format version is `2.0`, while content-anchor behavior is the v2.1 design.
See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the schema and write
invariants.

Environment variables:

- `USER`: default reviewer/TUI author;
- `EDITOR`: target editor for citation peek;
- `COMMENTS_THEME`: `nord`, `dracula`, `gruvbox`, or `ansi`;
- `COMMENTS_ACTOR`: explicit `human` or `agent` override for human-zone guards.

## Diagnostics

```bash
comments doctor
comments doctor --json
comments doctor --skip-mcp
```

`doctor` checks the binary/version, MCP handshake, installed plugin version,
and sidecar health. Failures exit `1`; warnings alone keep exit `0`.

For development and troubleshooting, see [CLAUDE.md](CLAUDE.md). For the
documentation status and retention policy, see [docs/README.md](docs/README.md).
