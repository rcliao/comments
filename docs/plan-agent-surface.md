# Plan: agent-surface fixes (anchor text, list shape, validate feedback, seed)

## Overview

Four fixes to the CLI/MCP surface, one per research finding: comments anchor
by text instead of grepped line numbers; list returns roots with nested
replies and the write path accepts any ID; validate reports per-section
counts and names offenders; seed states that the template was recorded.
Ordered by the research's own frequency/severity ranking (Summary).

## Current State

- add/suggest take only `line`/`section`; no text-to-line path — a grep per
  anchor, stale lines mis-anchor silently (research-agent-surface.md:24-30).
- MCP list flattens roots + replies; reply IDs fail the write path while
  nested reply text goes unfound (research-agent-surface.md:32-43).
- validate names over-cap sections but shows no counts when under cap; the
  external miss is unreproduced (research-agent-surface.md:45-53). CLI JSON
  list is already roots-nested (cmd/comments/list_filters.go:155).
- Seed's template-recording is implicit on MCP (research-agent-surface.md:55-62).

## Desired End State

An agent adds a comment by quoting the target line, reads a thread's replies
from one list call, replies to any ID it was shown, sees per-section word
counts on every validate, and knows seed recorded the template. Verify: the
next RPI session runs zero anchor greps and zero raw sidecar reads, and its
first over-budget doc trims to green in one validate pass.

## What We're NOT Doing

- No fuzzy anchor matching in v1 — exact and whitespace-normalized only;
  ambiguity is an error listing candidate lines, never a guess.
- No change to sidecar storage; TUI untouched. (CLI validate/seed OUTPUT
  changes in Phases 3-4; comment listing text output does not.)
- No new `template set` command — seed's response gets explicit semantics;
  a separate command waits for evidence it is needed.
- MCP resources untouched (no friction evidence either way,
  research-agent-surface.md:64-71).

## Implementation Phases

### Phase 1: anchor-by-text comment creation

Add `anchor` (exact or whitespace/case-normalized line match) as an
alternative to `line` on add, suggest, batch_add and their MCP tools —
resolution reuses the anchor-cascade matching that re-anchoring already
trusts (pkg/comment/positions.go / anchoring code). Zero matches or >1 match:
error listing candidates with line numbers, so the agent picks precisely.

**Success Criteria**
- automated: unit tests — unique match anchors correctly; ambiguous and
  missing anchors error with candidates; suite green under -race
- manual: one RPI callout round posted with `anchor` only, no grep

### Phase 2: list shape + tolerant write path

MCP list returns thread ROOTS with replies nested (aligning to CLI JSON —
consistency fix, not a new shape); filters match roots OR their replies but
always return the root. reply/resolve/reanchor resolve ANY known ID to its
thread: a reply ID addresses its parent thread instead of "thread not found"
(research-agent-surface.md:32-43). `parent_thread_id` appears on nested
replies so flattened consumers can migrate.

**Success Criteria**
- automated: list output contains no top-level reply items; reply, resolve
  AND reanchor each accept a reply ID (landing on the parent thread);
  filter-matching-a-reply returns its root
- manual: re-run the external agent's failing sequence (list → reply to a
  reply ID) — succeeds

### Phase 3: validate reports counts, always

validate (CLI + MCP) appends a per-section word report — count and cap for
every capped section, count alone for uncapped, doc total — on success AND
failure, so trimming is informed, not blind (research-agent-surface.md:45-53).
First: reproduce the external miss against the design-doc template; if the
heading matcher is at fault, fix it here.

**Success Criteria**
- automated: over-cap doc reports the offending section AND the full count
  table; repro test for the external design-doc case (or documented
  non-repro); suite green
- manual: one over-budget doc trimmed to green in a single pass using the
  table

### Phase 4: seed semantics

Seed responses (CLI + MCP) state `template_recorded: <name>` explicitly and
that the gate now enforces it; the MCP response leads with that field so an
empty `seeded` list no longer reads as the whole answer — recording is a
distinct act from thread-seeding (research-agent-surface.md:55-62).

**Success Criteria**
- automated: MCP seed response asserts template_recorded field; CLI wording
  test
- manual: none — wording change

## Risks

- **Phase 2 changes MCP list output** — consumers expecting flattened
  replies break. Accepted: the only known consumers are agents reading tool
  results per-call; `parent_thread_id` eases any migration.
- **Anchor ambiguity annoyance** — common lines (headings, blanks) collide.
  Mitigated: error lists candidates; agents fall back to `line` precisely.
