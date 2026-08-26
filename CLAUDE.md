# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`comments` is a CLI tool for collaborative document writing with inline comments and suggestions, designed for seamless LLM integration. It brings Google Doc-style commenting and track-changes functionality to terminal-based markdown editing, enabling better collaboration between humans and AI agents.

**Key Philosophy**: Instead of having LLMs rewrite entire documents, add contextual comments at specific lines to guide iteration and discussion. Suggestions allow proposing edits with preview and accept/reject workflow.

**Storage Model (v2.0)**: Comments and suggestions are stored in sidecar JSON files (`.md.comments.json`) separate from the markdown content, with nested thread structure and automatic staleness detection.

## Build and Development Commands

```bash
go build -o comments ./cmd/comments   # build root binary
go test ./...                         # run all tests
./scripts/ci.sh                       # every gate CI runs — required before commit/push
```

**Important**: After any code change, rebuild the root binary (`go build -o comments ./cmd/comments`) before testing CLI behavior — stale binaries have masqueraded as missing features.

### One-time setup per clone

```bash
git config core.hooksPath .githooks   # enables the pre-commit / pre-push gates
```

`.githooks/pre-commit` runs the fast gates (gofmt, vet — well under a second);
`.githooks/pre-push` runs the full `./scripts/ci.sh`. Both are plain shell with
no dependencies, and `--no-verify` bypasses them when you genuinely need to.
`./scripts/ci.sh` tells you if the hooks are not wired up, because a guard that
was never enabled looks exactly like a guard that passed.

### Run the CI gates before you commit or push (required)

`./scripts/ci.sh` runs, in CI's order: `gofmt -l`, `go build`, `go vet`,
`go test -race`, `golangci-lint` (version-pinned to the workflow), and the
review-flow smoke test. **A change is not ready to commit until this is green**,
and never push on the assumption CI will catch it.

- `go test ./...` alone is **not** enough. It misses three gates that have each
  caught real defects here: `-race`, `golangci-lint` (two unchecked `Close`
  errors), and the smoke test (a `zone: human` bypass, and a scan that died on
  an unreadable directory).
- `SKIP_LINT=1` exists for a fast inner loop only. It prints a loud skip notice;
  a run with lint skipped does not count as green.
- CI and the smoke test share one script (`scripts/smoke-test.sh`) on purpose.
  Do not inline a second copy into the workflow — duplicated definitions in this
  repo have gone stale every time (see the MCP tool banner in Design Decision 8).
- If a gate is failing for a reason you believe is unrelated to your change,
  find out why before pushing rather than assuming; both CI failures on this
  repo that looked environmental turned out to be genuine bugs.

Command surface: run `./comments` with no args for full usage. The core review-loop commands:

```bash
./comments view doc.md                                  # interactive TUI (the human review surface)
./comments add doc.md --line 10 --author eric --text "Fix this" --blocking
./comments add doc.md --anchor "quoted target line" --author claude --text "..."   # no grep for line numbers
./comments gate doc.md            # exit 0 = approved, 10 = changes requested
./comments signoff doc.md         # record a review pass (what agents wait on)
./comments watch specs/ --until signoff                 # block until a signoff (NDJSON events)
./comments validate draft.md --template design-doc
./comments new cache-policy --template design-doc       # when a bundle is configured
./comments context docs/artifacts/designs/cache-policy.md --for drafting
./comments doctor                 # install health: binary, MCP, plugin, sidecars
./comments serve-mcp              # MCP server over stdio
```

`--text @file.txt` reads text content from a file (also `--original`/`--proposed` on `suggest`) — useful for multi-line content.

### After merging a plugin-affecting change (skill, MCP, templates, version bump)

The installed Claude plugin does NOT track this repo — it is built from a
separate marketplace clone that never auto-pulls
(it once sat five days stale while every agent read an obsolete skill).
After any merge that touches `skills/`, `pkg/mcp/`, templates, or bumps
`.claude-plugin/plugin.json`, run:

```bash
claude plugin update comments@comments   # pulls the marketplace clone AND rebuilds the cache
```

Then restart running Claude sessions to pick it up. `comments doctor` compares
the installed plugin version against the one this binary ships
(`PluginVersion` in pkg/comment/doctor.go — bumped together with
`.claude-plugin/plugin.json`, a test enforces the pair) and warns with the
update command when they drift.

### Doc Templates (Review Guardrails)

Templates constrain what an agent writes so humans can review it well. A template (YAML, built-in or in `.comments/templates/`) declares:

- **Sections**: required headings (matched by title or path suffix), order, `max_words` caps (attacks LLM padding), `min_subsections` (e.g. "Options Considered" needs >= 2 alternatives). **Citation tokens (`file.go:12`, `thread:c1abc`) do not count toward any word cap** — one counter, `countWords`, strips them via `markdown.StripCitations`, so evidence never competes with content (measured cost before the exemption: ~12% of a section's budget, `scripts/eval/`).
- **`zone: human`**: human-decision sections. Threads anchored there cannot be resolved by agents over MCP — the agent gets an error telling it to reply instead; only the human resolves (CLI/TUI).
- **`review_criteria`**: per-section self-review prompts for the *agent* — the skill requires the agent to judge its draft against each criterion and post doc-specific callouts (weakest reasoning, assumptions, invented facts) at exact lines, instead of forwarding generic questions.
- **Markers**: every `[NEEDS CLARIFICATION: ...]` occurrence is a validation violation; agents add a specific blocking Q comment at that line instead of guessing (Spec Kit convention).

Workflow: agent creates a bundle concept or reads the template (`comments_get_template`) → loads `comments_context` → drafts → validates and self-corrects → adds specific inline annotations → human review → `comments gate` → agent listens for `signoff`.

### Review Gate and Signoff

The gate turns review state into a machine-readable contract for agent loops and SDD phase boundaries:

- **Blocking comments**: `--blocking` on `add` (or `"blocking": true` in batch/MCP) marks a thread as gate-failing until resolved. Non-blocking comments are reported but don't fail the gate.
- **`comments gate <file-or-dir>`**: exit 0 = approved, exit 10 = changes requested (revdiff/Plannotator convention). `--json` emits `{"decision", "files", "summary"}` with blocking/non-blocking/pending-suggestion lists and document context. `--strict` fails on any unresolved thread or pending suggestion.
- **Signoff** — a review pass recorded in the sidecar's `reviews` array (author, decision, optional note). There are **two equivalent writers**, and every consumer (`request_review`, `check_review`, `watch --until signoff`) keys on the record, not on who wrote it:
  - `comments view <file>` → `q` → `a`/`c`/`r`, with `n` for the note. `r`
    records decision `commented` — a reply-pass: the human answered threads
    without judging the doc; agents process the replies and keep iterating
    (never treat it as approval). Also applies the queued suggestion decisions and exits 0/10, so the TUI doubles as the interactive gate. **A human who reviewed in the TUI has already signed off — do not also ask them to run `comments signoff`** (it would append a second record).
  - `comments signoff <file>` for everything non-interactive: CI, scripts, `--decision`/`--note`/`--strict` overrides, or signing off a doc reviewed elsewhere. Decision derives from the gate unless overridden.
- **Waiting for a review** (no MCP): `comments watch <file-or-dir> --until signoff` blocks and exits 0 on the first signoff, emitting `{"event":"signoff","author","decision","note"}` — the decision and the reviewer's message in one event. The sidecar is the shared event bus, so it fires for either writer above.
- **Agent loop**: agent drafts → tells the human to review and listens with `comments watch --until signoff` → human reviews and signs off (`comments view`, verdict on exit) → agent runs the inbox FIRST (replies are the payload, the decision is the envelope), then acts on the decision (see `skills/review-comments/SKILL.md`) → repeat until gate passes.

### Model Context Protocol (MCP) Integration

`./comments serve-mcp` runs an MCP server over stdio: 2 subscribable resources (`comments://doc/{filepath}`, `comments://thread/{filepath}/{thread_id}`) and 23 tools mirroring the CLI (list/get/status/analyze, add/reply/resolve, suggest/accept/reject, batch ops, gate/request_review/check_review, inbox, template get/validate, new/context/bundle index, reanchor). The tool catalog with schemas lives in `pkg/mcp/server.go`. Notable semantics:

- **comments_request_review** — default blocks until a human signoff, then returns the decision + remaining comments. With `blocking: false` returns `{status: "requested", since: <RFC3339>}` — a durable handle polled via **comments_check_review** (survives agent restarts).
- **comments_inbox** — one-call attention view: unresolved threads with replies newer than `since`, plus all unresolved blocking threads.
- **comments_reanchor** — after editing a commented document, agents must migrate the anchors their edits displaced (batch comment_id → new line/section).

### Content Anchoring (v2.1)

Comments carry a content `Anchor` (target line text + one line of context each side), captured automatically at creation. On document change, each comment re-anchors via a cascade: exact position → exact text search → normalized (whitespace/case-insensitive) search, labeled `anchor_confidence: fuzzy` → section-path fallback (`section-level`) → orphan. `add --line N` stores N exactly — comments no longer snap to section headings. Old sidecars get anchors backfilled lazily on load when the document hash matches. Comment IDs are short random base36 (`c7f3k`); existing long IDs stay valid, and `SaveToSidecar` guarantees uniqueness.

### Section-Based Operations

Comments can target a markdown section instead of a line: `--section "Implementation > Architecture"` (hierarchical path, " > " separator). `--section` and `--line` are mutually exclusive; section filters on `list` include all descendant sections (tree behavior). Invalid paths error with the list of available sections.

## Architecture Notes

- `pkg/comment/` is pure logic (storage, validation, anchoring, templates, gate, watch) with no UI dependencies; `pkg/tui/` is the Bubbletea front end (see `pkg/tui/CLAUDE.md` for TUI patterns and gotchas); `pkg/mcp/` wraps `pkg/comment` for MCP; `pkg/markdown/` parses ATX headings for section addressing.
- **Threading (v2.0)**: replies nest in each Comment's `Replies` array — there is no flat thread/parent-ID model. Always use the helpers in `pkg/comment/helpers.go` (`AddReplyToThread`, `ResolveThread`, `AcceptSuggestion`, ...) instead of manipulating `Replies` directly; use `doc.GetAllComments()` to flatten for searching.
- **Suggestions** are multi-line only: `StartLine`/`EndLine` + `OriginalText`/`ProposedText`, acceptance state is `Accepted *bool` (nil=pending, true=accepted, false=rejected).
- **Staleness**: sidecars store a SHA-256 `documentHash`; a hash mismatch on load marks the sidecar stale and triggers the re-anchoring cascade.

## RPI Flow (Research → Plan → Implement)

For feature-sized work the AUTONOMOUS CHAIN is the default: interview once, then question → research (`research-deep`) → draft-blind coverage scout + evidence verifier until convergence → plan → `comments analyze plan --against research` → ONE human sitting on the plan. Accepted coverage gaps become new `Qn` questions; rejected candidates remain resolved rationale threads; shape-changing survivors pause the chain. The paired eval under `scripts/eval/autonomous-research/` measures the provisional pass cap before dogfood. Say "gate the research" for the two-gate flow. Plans carry no open questions, cite or explicitly exclude every research finding, and split success criteria into automated/manual. See `skills/review-comments/SKILL.md`.

## Recommended Review Flow (the tool's core loop)

1. **Agent produces doc** under a template: use `comments new` in a configured bundle, read the brief and `comments context`, then draft and validate until structure is clean.
2. **Annotate and self-review**: post specific anchored callouts from the template criteria with `add`/`batch-add`; each ambiguity marker gets a blocking Q thread. Template identity lives in frontmatter, not in review threads.
3. **Human reviews** in the TUI: `comments view <doc>` — walk threads, reply/resolve, add comments (`--blocking` for must-fix), then submit the TUI verdict, which records the signoff.
4. **Agent processes feedback** one comment at a time (see `skills/review-comments/SKILL.md`): reply/resolve/suggest, `comments_reanchor` after edits, re-request review.
5. **Iterate until the gate unblocks**: `comments gate <doc>` exit 0 → implement.

## Adding a New CLI Command

1. Add case to switch in `cmd/comments/main.go`, implement a handler following the pattern of `addCommand`/`replyCommand`, update `printUsage()`, test with an example document.

## Environment Variables

- `USER`: Used as default author name for comments (falls back to "user")
