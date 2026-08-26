# Plan: resume — the plan as the durable state of long-running work

## Overview

Make a plan document the thing an agent resumes from and a human re-enters by.
Three additions: signoffs scoped to a phase, a `document_changed` wake-up event, and one command — `comments resume` — that answers "where are we" for a document or a spec folder.

When done, a cold session learns the current phase, which signoff covers each phase, what moved since, and what is still open.
Implementation of a phase waits on a signoff record that names it.

## Current State

Every plan since 2026-08-11 shipped code with no `reviews[]` entry; approvals live in replies saying "accepted in chat" (research-long-horizon-alignment.md:35-55, research-long-horizon-alignment.md:57-73).
No plan records phase progress; the one phase-gated plan did it with three manual signoffs (research-long-horizon-alignment.md:75-88).
`ReviewRecord` holds author, timestamp, decision, note and nothing about scope (pkg/comment/types.go:140-145).

Nothing checks a signoff exists before implementation; `request_review` only stats the file (research-long-horizon-alignment.md:90-107).
At session start `inbox` answers what is blocking, never where the work stands (research-long-horizon-alignment.md:129-137).
Every `since` is carried by hand; the plan→research link `analyze` computes is never persisted (research-long-horizon-alignment.md:139-148).
`watch` fires on review state only, never on a document edit (research-long-horizon-alignment.md:173-181).
Git already records "Implements docs/plan-X.md" in commit bodies, unread (research-long-horizon-alignment.md:150-160).

## Desired End State

An agent opening a new session runs `comments resume docs/` and gets, per plan:

- its research link
- each phase with the signoff that covers it
- lines and sections changed since that signoff
- open high-priority threads
- commits that claim to implement it

A human re-entering sees the current phase and its covering signoff in the TUI rail.
Verify: resume this plan after Phase 1 ships — it names Phase 1 signed off and Phase 2 next, with no hand-carried timestamps.

## What We're NOT Doing

- No reviewer identity or signoff authentication: the boundary is "a record exists", not "who wrote it" (research-long-horizon-alignment.md:90-107).
- No enforcement of the agent's other discipline rules — reanchor, inbox-first, thread budget stay prose (research-long-horizon-alignment.md:109-127).
- No commit, PR, or URL citation syntax; `resume` reads git log, docs do not cite commits.
- No MCP subscriptions.
- No as-built automation and no template schema for phases; phases remain the H3s under Implementation Phases (research-long-horizon-alignment.md:162-171).
- No research-convergence changes (owned by docs/plan-autonomous-research-convergence.md).
- No skill split (`thread:plan-landscape-improvements.md#cb5gr` stays open).
- No adoption of the two uncommitted, unsigned implementations in the tree; they are committed or parked before Phase 1 (decided on this plan).

## Implementation Phases

### Phase 1 — phase-scoped signoffs and a document_changed event

`ReviewRecord` gains two optional fields: `phase` — an H3 under Implementation Phases, per `markdown.ParseDocument` — and `document_hash` at signoff.
`comments signoff --phase` writes both; the TUI verdict offers the phase list when such H3s exist; `watch --until signoff` carries them.

`watch` gains `document_changed {file, hash, changed_sections}`, diffing poll-to-poll from the content it last read with `DiffLines` (pkg/comment/watch.go:49-57, pkg/comment/baseline.go:127); no baseline involved, first poll silent.
`analyze --against` persists `links.against` in the sidecar (pkg/comment/storage.go:17-24); `seed` is unchanged.

**Success Criteria**

- automated: sidecar round-trip tests for `phase` and `links`; `watch` test emits `document_changed` once per edit and never for sidecar-only writes; both signoff writers produce identical records; `./scripts/ci.sh` green.
- manual: in the TUI, `q` offers the phase list and the record lands with it. This plan's Phase 1 is approved with a plain signoff (legacy, covers the whole doc).

### Phase 2 — `comments resume` (CLI and MCP)

New command `comments resume <file-or-dir> [--author <reviewer>] [--json]` (author defaults to `$USER`, like `status`) and MCP `comments_resume`.
Per document it reports:

- template and `links.against`
- phases in order, each `signed_off`, `in_review` (unresolved threads anchored in its section), or `untouched`
- the covering record: latest naming the phase, or phase-less (covers the whole doc); a later `changes_requested` un-covers
- `changed` (record `document_hash` ≠ current) and, when the reviewer's baseline matches that hash, the changed sections
- open high-priority threads
- commits whose subject or body cites the document path (`git log --fixed-strings --grep`, no rename following, absent outside a repo)

`inbox` is unchanged: `resume` is the cold-start surface, `inbox` the warm one.

**Success Criteria**

- automated: fixture with three phases, two signoffs, one edit, one citing commit → expected JSON; CLI/MCP parity test; works without git.
- manual: run on docs/ — every plan's status matches what you remember doing.

### Phase 3 — the implement boundary, in gate and skill

`gate --json` gains `signoff_state` per file: the latest record, its phase, and `stale` when its `document_hash` differs from the current one.
`--require-signoff` makes the gate exit 10 when no approved record covers the requested phase; opt-in, so existing callers are untouched.

SKILL.md gains two rules.
Implementation of phase N begins only after `comments gate --require-signoff --phase N` passes.
An approval given in chat is recorded by running `comments signoff --phase N` at a terminal, never by an agent reply.
The rail shows the current phase and its covering signoff beside the gate decision.

**Success Criteria**

- automated: gate tests for covered / stale / missing signoff; `grep` finds both skill rules.
- manual: dogfood on this plan — Phase 1 is implemented only after its signoff record exists.

## Risks

- **Phase titles drift after signoff** — a renamed H3 orphans its record. Mitigated: `resume` matches by normalized title and reports `unmatched_phase` rather than dropping it.
- **Git grep false positives** — a commit citing the plan for another reason. Accepted: commits are listed as claims, never as status.
- **`--require-signoff` ignored** — it is opt-in and the agent decides to call it. Accepted: who signs stays out of scope; a record existing is the improvement over "accepted in chat".
