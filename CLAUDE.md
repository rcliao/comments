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
```

**Important**: After any code change, rebuild the root binary (`go build -o comments ./cmd/comments`) before testing CLI behavior — stale binaries have masqueraded as missing features.

Command surface: run `./comments` with no args for full usage. The core review-loop commands:

```bash
./comments view doc.md                                  # interactive TUI (the human review surface)
./comments add doc.md --line 10 --author eric --text "Fix this" --blocking
./comments add doc.md --anchor "quoted target line" --author claude --text "..."   # no grep for line numbers
./comments gate doc.md            # exit 0 = approved, 10 = changes requested
./comments signoff doc.md         # record a review pass (what agents wait on)
./comments watch specs/ --until signoff                 # block until a signoff (NDJSON events)
./comments validate draft.md --template design-doc
./comments seed draft.md --template design-doc
./comments serve-mcp              # MCP server over stdio
```

`--text @file.txt` reads text content from a file (also `--original`/`--proposed` on `suggest`) — useful for multi-line content.

### Doc Templates (Review Guardrails)

Templates constrain what an agent writes so humans can review it well. A template (YAML, built-in or in `.comments/templates/`) declares:

- **Sections**: required headings (matched by title or path suffix), order, `max_words` caps (attacks LLM padding), `min_subsections` (e.g. "Options Considered" needs >= 2 alternatives).
- **`zone: human`**: human-decision sections. Threads anchored there cannot be resolved by agents over MCP — the agent gets an error telling it to reply instead; only the human resolves (CLI/TUI).
- **`review_criteria`**: per-section self-review prompts for the *agent* — the skill requires the agent to judge its draft against each criterion and post doc-specific callouts (weakest reasoning, assumptions, invented facts) at exact lines, instead of forwarding generic questions. `comments seed` without flags still materializes criteria as generic blocking threads (useful for human-only workflows); agent flows use `seed --markers-only`.
- **Markers**: every `[NEEDS CLARIFICATION: ...]` occurrence is a validation violation and seeds a blocking Q thread at that line — agents must flag ambiguity instead of guessing (Spec Kit convention).

Workflow: agent reads the template (`comments_get_template`) as its writing brief → drafts → `comments_validate` and self-corrects structure → `comments_seed` → human review = resolving seeded/own threads → `comments gate` (validates structure + comment state; `seed` records the template in the sidecar so the gate picks it up automatically) → `signoff`.

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
- **Agent loop**: agent drafts → calls `comments_request_review` (MCP, blocks) or waits on `comments watch --until signoff` → human reviews and signs off (`comments view`, verdict on exit) → agent runs the inbox FIRST (replies are the payload, the decision is the envelope), then acts on the decision (see `skills/review-comments/SKILL.md`) → repeat until gate passes.

### Model Context Protocol (MCP) Integration

`./comments serve-mcp` runs an MCP server over stdio: 2 subscribable resources (`comments://doc/{filepath}`, `comments://thread/{filepath}/{thread_id}`) and 20 tools mirroring the CLI (list/get/status, add/reply/resolve, suggest/accept/reject, batch ops, gate/request_review/check_review, inbox, template get/validate/seed, reanchor). The tool catalog with schemas lives in `pkg/mcp/server.go`. Notable semantics:

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

For feature-sized work, use the phase templates: agent drafts research under the `research` template (findings with file:line evidence; open questions live there) → human signs off → agent drafts the plan under the `plan` template, citing the research by `file:line` → human reviews the plan in the TUI, peeking citations with `f` → gate green → implement phase by phase. Plans carry no open questions (marker cap 1) and split every phase's success criteria into automated vs manual. See `skills/review-comments/SKILL.md` (RPI mode).

## Recommended Review Flow (the tool's core loop)

1. **Agent produces doc** under a template: read the brief (`comments_get_template`), draft, `comments validate` until structure is clean.
2. **Seed** review threads: `comments seed` (template criteria + NEEDS CLARIFICATION markers become blocking threads; template recorded in sidecar).
3. **Human reviews** in the TUI: `comments view <doc>` — walk threads, reply/resolve, add comments (`--blocking` for must-fix), then `comments signoff`.
4. **Agent processes feedback** one comment at a time (see `skills/review-comments/SKILL.md`): reply/resolve/suggest, `comments_reanchor` after edits, re-request review.
5. **Iterate until the gate unblocks**: `comments gate <doc>` exit 0 → implement.

## Adding a New CLI Command

1. Add case to switch in `cmd/comments/main.go`, implement a handler following the pattern of `addCommand`/`replyCommand`, update `printUsage()`, test with an example document.

## Environment Variables

- `USER`: Used as default author name for comments (falls back to "user")
