# Review: CLI and MCP surface, end to end (2026-09-18)

Read-only review. No tool code was changed.

**Method.** One driver script ran the same 26-step review workflow through the
CLI and through a live `comments serve-mcp` process, on twin copies of one
`design-doc`. Agent steps used the surface under test (no TTY, no
`COMMENTS_ACTOR`). Human steps used the CLI with `COMMENTS_ACTOR=human` in both
workspaces, because humans never use MCP. Binary: branch
`feat/design-doc-pitch-tiers`, built from the working tree.

Driver and raw results (session scratchpad, not in the repo):
`scratchpad/e2e/drive.py`, `scratchpad/e2e-run/results.json`.

## Workflows, ranked by how much breaks if they fail

| Rank | Workflow | Commands | Verdict |
|---|---|---|---|
| P0 | Human authority: who may decide what | `resolve`, `accept`, `reject`, `batch-accept`, `signoff` | **Broken** for `accept` — finding 1. `reject` and `batch-accept` were not exercised; no guard call site exists for either. |
| P0 | Handoff and wait for signoff | `watch --until signoff`, `request_review`, `check-review` | Works; surfaces differ — finding 3 |
| P0 | Read the worklist | `gate`, `inbox` (the skill's path), then `list`, `get`, `status` | `gate` and `inbox` carry the same content on both. `list`, `get`, `status` differ — findings 2, 4 |
| P0 | Act on feedback | `reply`, `resolve`, `suggest`, `reanchor` | Works, identical on both |
| P1 | Draft | `new`, `template show`, `context`, `validate` | Works; brief differs — finding 5 |
| P1 | Annotate | `add`, `batch-add` | Works; section paths are a trap — finding 6 |
| P2 | RPI analysis, bundle index, doctor | `analyze`, `bundle index`, `doctor` | Works |

**What holds.** The final sidecar was identical across surfaces: same threads,
lines, anchor confidence, resolution state, suggestion state, and review record.
Both reached gate `approved`. Behaviour lives in `pkg/comment` as intended; the
differences below are all in the adapters.

## Findings

### 1. An agent can rewrite a human-owned section by accepting its own suggestion (P0)

`zone: human` is enforced at exactly one verb, `resolve` (`GuardZoneResolve`,
called from `cmd/comments/main.go:669`, `pkg/mcp/tools.go:326`, and
`pkg/webreview/server.go:538`). `accept` has no guard on either surface.

Reproduced as an agent (no TTY, no override), CLI:

```
comments suggest doc.md --anchor "Reads are slow and a cache is the smallest fix." \
  --author claude --original "..." --proposed "AGENT-WRITTEN PITCH LINE."
comments accept doc.md --suggestion cekf8
✓ Suggestion cekf8 accepted and applied      # line 13 of ## Pitch now reads AGENT-WRITTEN PITCH LINE.
```

Reproduced over MCP as well: `comments_suggest` into the Pitch, then
`comments_accept`, returned `"success": true` and line 13 of the Pitch became
`AGENT-WRITTEN PITCH LINE (via MCP).`

Why it matters: the skill says "do not accept your own suggestions", and the
`Pitch` design shipped on this branch says "the agent suggests, the human
accepts". Neither is enforced. The sanctioned path for AI help is also the
bypass. This is the same class as the resolve guard and has the same fix
shape: a `GuardZoneAccept` in `pkg/comment` called by both adapters, refusing an
agent `accept` (and `batch-accept`) when the suggestion's lines sit in a
`zone: human` section. A stricter option refuses any agent accepting a
suggestion it authored.

### 2. `list` has a different default on each surface (P1)

After one thread was resolved, the same call returned:

- CLI `list --format json`: 4 threads (resolved hidden by default), a bare array.
- MCP `comments_list`: 5 threads (resolved included), an object
  `{comments, filepath, total}`.

Each MCP item carries `resolved: true`, so the agent can tell; the defaults just
disagree. Ranked P1 because the skill's worklist path is `gate --json` and
`inbox`, not `list`. The resolved thread also still reports `status: active`.

### 3. The wait-for-review step is two different tools (P0)

| | CLI `watch --until signoff` | MCP `comments_request_review` |
|---|---|---|
| Payload | 463 chars: three NDJSON events ending in `signoff` with author, decision, note | 2,021 chars: the review record plus a full gate dump of every blocking thread with context |
| Intermediate events | yes (`comment_added`, `reply_added`) | no |

Both carry the reviewer's note, so neither is broken. But there is no CLI
`request-review` and no MCP `watch`, the skill teaches only `watch`, and
`check-review`/`comments_check_review` also differ: MCP adds the same `files`
gate dump, the CLI does not. This pair is the largest prose in the catalog
(2,136 chars of description) and the least aligned.

### 4. Same command, different JSON, in the older commands only

| Command | Difference |
|---|---|
| `status` | MCP lacks `blocking_threads` and `template`; CLI lacks `last_validated`. `blocking_threads` is the number an agent most needs. |
| `gate` | MCP lacks `summary`; MCP `file` is absolute, CLI relative; CLI items carry `anchor_confidence`, `reply_count`, `resolved`, `status`, `timestamp`, MCP items do not. |
| `gate` vs `get` text | MCP `gate` returns `[Q] Is four…`; MCP `get` and CLI `gate` return `❓ [Q] Is four…`. The emoji prefix is applied inconsistently. |
| `get` | CLI has no JSON output at all (`--format` is rejected); MCP returns JSON. |
| `validate` | CLI includes `file`; MCP does not. |
| `bundle index` | CLI text only; MCP JSON. |
| `check-review` | see finding 3. |

`new`, `context`, `inbox`, and `analyze` returned **identical** key sets. Those
are the newest commands, written after the "adapters only" rule, and each
marshals one result type from `pkg/comment` (`Analysis`, `ContextDocument`,
`InboxItem`). The older commands build their shape in the adapter: CLI `status`,
for example, is a map literal at `cmd/comments/parity.go:212`. That is the root cause, and it points at the
real trim: one result type per command in `pkg/comment`, marshalled by both
adapters, instead of shortening descriptions.

Argument names also differ for the same concept: `get --thread` vs `comment_id`;
`status --author` vs `reviewer`; `resolve --thread` vs `thread_id`.

### 5. The MCP template brief is bigger than the CLI's and says less (P1)

`comments_get_template` marshals the raw Go struct: 6,803 chars against the
CLI's 5,633. It has PascalCase keys, every zero-valued field on every section,
and none of the reading path, style guidance, or marker-budget notes the CLI
prints.

### 6. `--section` needs the full path including the document title (P1)

`--section "Proposed Design"` fails on both surfaces; only
`"Cache policy > Proposed Design"` works. Templates match headings by title or
path suffix, so the two section-matching rules disagree. The error lists the
valid paths, so an agent recovers in one retry. `batch-add` is atomic: one bad
item rejects the whole batch, which is correct, but the MCP error does not say
which item failed (the CLI says "Error in comment 1").

### 7. Smaller observations

- Exit code 10 from `check-review` means "changes requested", not failure. It
  matches `gate`, but a script using `set -e` will treat a completed review as an
  error.
- Bundle discovery walks up past a git repository boundary: `comments new` in a
  fresh nested repo wrote into a parent directory's `.comments/bundle.yaml`.
- The sidecar mixes three key casings: `documentHash`, `ID`/`Text`, and
  `selected_text`.
- Accepting a suggestion drops sibling threads anchored to the replaced line to
  `section-level`. Expected from the cascade, but the skill does not tell agents
  to `reanchor` after an accept.
- `CLAUDE.md` cites "Design Decision 8"; `docs/ARCHITECTURE.md` no longer has
  numbered decisions.

## Suggested order

1. Guard `accept`/`batch-accept` for agents in human zones (finding 1).
2. Align `list` defaults and add the missing `blocking_threads` to MCP `status`
   (findings 2, 4).
3. Move per-command result types into `pkg/comment`, starting with the template
   brief, `gate`, `status`, `list`, `get` (findings 4, 5). Add a parity test that
   runs this driver's comparison in Go, so shapes cannot drift again.
4. Decide the wait step: one mechanism on both surfaces (finding 3).
5. Only then trim tool descriptions or remove tools.
