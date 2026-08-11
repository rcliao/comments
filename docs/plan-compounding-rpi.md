# Plan: compounding RPI — rounds build on doc + thread history

## Overview

Make each RPI round inherit the last instead of restarting.
Agents read full thread history before drafting.
Vetoed designs move into the doc as rejected alternatives, their threads resolved into the readable trace.
Thread IDs become peekable citations, so decisions link the way code does.
One tool change (thread citations); the rest is skill convention, hardened later by measurement.

## Current State

- The skill has agents consume the human's signals at signoff, none of their own telemetry (research-skill-quality-surfacing.md:14-15); no step reads thread history before drafting.
- No rule addresses re-proposal of vetoed designs — this plan's observation, implied by the skill consuming no telemetry (research-skill-quality-surfacing.md:47-49 names the unread sidecar record).
- Citations resolve files only (pkg/markdown/refs.go:32); a thread ID in prose is dead text.
- Decisions (rcliao, 2026-08-11): join-first and reviewer-gets-threads are in the research thread record (c6mv7); veto-moves-to-alternatives was decided in chat — recorded as a blocking thread on THIS doc for confirmation, itself a demo of why chat decisions must land in threads.

## Desired End State

A round-2 draft demonstrably consumes round-1's threads.
Vetoed options sit under Options Considered citing their threads, `f` opens the debate, and no resolved veto resurfaces as a fresh proposal.
Verify: one real feature run where the human spot-checks inheritance at review.

## What We're NOT Doing

- No `vetoed` thread state or schema change — the doc is the veto's home (decision above).
- No cross-feature veto binding: compounding scopes to one feature's doc lineage (research → plan → as-built); sibling features read each other's docs, not threads.
- No automated re-proposal detection — the reviewer checks it, armed with history.
- No harness/metrics work here, including the lead-lag MEASUREMENT itself (research F4): this plan feeds history forward; measuring which signal predicts gaps is the harness plan's job.

## Implementation Phases

### Phase 1: thread citations — `thread:c1abc` peeks the thread

Syntax decided in review (thread cmvlt): an explicit `thread:` scheme — `thread:c1abc` for this doc, `thread:path.md#c1abc` cross-doc.
No heading-anchor collision exists, so no precedence rules are needed.
The peek box renders the thread with the existing renderer; `Enter` opens the doc at the thread's anchored line.
Unknown IDs report like missing files.

**Success Criteria**
- automated: parse tests for both forms (fences: comment trails only, as with file refs); peek renders a thread from another doc's sidecar; unknown ID errors in-box; suite green under -race
- manual: `f` on a rejected option's thread citation shows the veto debate beside the doc

### Phase 2: compounding conventions (skill)

Before ANY drafting round, read the doc's full thread history, open and resolved.
Plan drafting also reads the research doc's threads; the fresh reviewer's allowlist gains the sidecar(s).

On a veto: write the killed design under Options Considered (or NOT-doing) citing its thread, reply what was recorded, resolve the thread.
Never re-propose a recorded veto; if new evidence justifies revisiting, cite the vetoed thread and say what changed.

**Success Criteria**
- automated: `grep` finds history-first, veto-to-alternatives, no-silent-reproposal, and sidecar-in-allowlist in SKILL.md (the check is a command, per plan.yaml's own criterion)
- manual: dogfood round where a vetoed option lands in alternatives with a working thread citation

### Phase 3: dogfood on the next real feature

Run one feature through the compounded loop; the human spot-checks at each review: did the draft inherit, did vetoes stick, did citations peek.
Fold misfires back into SKILL.md; note the conventions in root CLAUDE.md.

**Success Criteria**
- automated: gate green on the feature's docs; thread-citation peek exercised in the run
- manual: your read at signoff — inheritance felt, or name where it broke

## Risks

- **History bloat** — full thread history in every round's context grows with rounds.
  Mitigated: resolved-trace reading is a skim with the walkthrough order (`P`) and ≤50-word comments; revisit if a lineage exceeds ~50 threads.
- **Citation rot** — thread IDs survive edits (sidecar-keyed), but a deleted sidecar kills them; accepted, same exposure as file citations.
