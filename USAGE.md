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
All three choices record a review in the sidecar. Do not run `comments signoff`
after submitting a TUI verdict; `signoff` is the non-interactive alternative.

`comments serve` provides the same review loop in a browser. Open the one-time
URL printed by the command; it exchanges its random token for a local HttpOnly
session cookie. The server binds to loopback only, watches the markdown and
sidecar for changes, and rejects stale writes rather than overwriting a newer
review action. A directory target provides a document queue for every markdown
file under it that already has a comment sidecar.

## Command map

| Area | Commands |
|---|---|
| Human review | `view`, `serve` |
| Read threads | `list`, `get`, `status`, `inbox` |
| Write threads | `add`, `batch-add`, `reply`, `batch-reply`, `resolve` |
| Suggestions | `suggest`, `accept`, `batch-accept`, `reject` |
| Review coordination | `gate`, `signoff`, `check-review`, `watch` |
| Bundles and context | `new`, `context`, `bundle index` |
| Templates and artifact analysis | `template list`, `template show`, `validate`, `analyze` |
| Anchor maintenance | `reanchor` |
| Diagnostics and integration | `doctor`, `serve`, `serve-mcp` |

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
comments list doc.md
comments list doc.md --resolved --priority high --format table
comments list doc.md --section "Design" --with-context
comments list doc.md --status orphaned --format json

comments get doc.md --thread c7f3k
comments get 'thread:research.md#c7f3k' --from plan.md

comments reply doc.md --thread c7f3k --author claude --text "Applied in the draft"
comments resolve doc.md --thread c7f3k
```

`list` hides resolved roots by default. Its useful filters are `--type`,
`--author`, `--search`, `--line-range`, `--section`, `--status`, `--priority`,
and `--sort`; output can be `text`, `table`, or `json`.

`get` accepts either a document plus `--thread`, or a thread citation copied
directly from prose. `thread:c7f3k` means the citing document; use `--from` so
same-document and relative-path citations resolve correctly.

### Batch writes

`batch-add` validates the whole input before writing. Each item needs `author`,
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
    "section": "Risks",
    "author": "claude",
    "text": "Add the rollback risk",
    "type": "S"
  }
]
```

```bash
comments batch-add doc.md --json comments.json
comments batch-reply doc.md --json replies.json
```

Batch replies use objects shaped like
`{"thread":"c7f3k","author":"claude","text":"Applied"}`.
Pass `--json -` to either command to read from standard input.

## Edit suggestions

Suggestions target a line range, a whole section, or an anchor. With
`--anchor`, the number of lines in `--original` determines the range.

```bash
comments suggest doc.md --anchor "The old first line" \
  --author claude --text "Clarify the contract" \
  --original @old.txt --proposed @new.txt

comments accept doc.md --suggestion c91ab --preview
comments accept doc.md --suggestion c91ab
comments reject doc.md --suggestion c82de
```

`accept` is the content-writing path: it updates the markdown, marks the
suggestion accepted, shifts affected positions, and refreshes the sidecar.
`batch-accept` accepts pending suggestions by a JSON array of IDs, `--author`,
or `--type`.

In the TUI, `a` and `x` queue accept/reject decisions. The queue is applied
atomically when a verdict is submitted and is discarded by `Ctrl+C`.

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
own specific self-review callouts with `add` or `batch-add`; generic criterion
threads are intentionally not generated.

### OKF bundles and agent context

Projects opt in with `.comments/bundle.yaml`. The config maps templates to
review-friendly collections under one knowledge root. Concept files use
OKF-compatible frontmatter; `related` is a producer extension used for
deterministic navigation.

```bash
comments new cache-policy --template research-deep --description "Evidence for cache invalidation policy"
comments new cache-policy --template plan --from docs/artifacts/research/cache-policy.md
comments context docs/artifacts/plans/cache-policy.md --for drafting --include-threads
comments bundle index
```

`new` selects the folder from the template, emits required section headings,
creates an empty review sidecar, and refreshes root and collection indexes.
`context` returns explicit frontmatter relations, Markdown links, backlinks,
sources, review state, and up to five tag-based suggestions. Every edge names
why it was included. `coverage-scout` exposes only the Research Question as its
`focus` while forcibly excluding bodies, threads, and draft-derived links;
`evidence-verifier` and review modes do not broaden the working set with tag
suggestions.

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

## Signoff and waiting

```bash
# Non-interactive review record; decision derives from the gate.
comments signoff doc.md --author eric --note "Ready after the cache fix"

# Wait for either a TUI verdict or a signoff command.
comments watch doc.md --until signoff

# Durable non-blocking polling handle.
comments check-review doc.md --since 2026-08-12T18:30:00Z --json

# Agent attention view: new replies plus unresolved blockers.
comments inbox docs/ --since 2026-08-12T18:30:00Z --json
```

`watch` emits NDJSON events including `comment_added`, `reply_added`,
`thread_resolved`, `suggestion_accepted`, `signoff`, and `gate_changed`.
`--until` accepts a comma-separated event list.

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
