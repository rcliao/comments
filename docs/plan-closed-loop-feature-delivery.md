# Plan: improve research and plan review without expanding workflow scope

## Overview

Improve comments at the work it owns: evidence-backed Markdown and precise human review.
After signoff, plan threads become a durable decision record across agent handoffs while the harness continues to own execution.

## Current State

`research-deep`, question coverage, citation validation, and `comments analyze` give agents a deterministic research floor (research-closed-loop-feature-delivery.md:34-45).
Coverage scouts and evidence verifiers add semantic challenges as contestable threads rather than model-authored gate rules.

Templates cap prose and prioritize reviewer attention, while the sidecar preserves discussion and decisions (research-closed-loop-feature-delivery.md:47-56).
The browser has a sticky review rail and point-and-click targeting, but comments preserves no cross-session implementation context (pkg/webreview/static/app.css:108-150, research-closed-loop-feature-delivery.md:155-165).

## Desired End State

An agent recursively expands and verifies research, then distills it into a short, cited plan.
A human reviews it in one sitting; implementation agents return material findings and deviations to anchored plan threads without duplicating runtime status.

## What We're NOT Doing

- No artifact graph, checkpoints, resume protocol, implementation ledger, or as-built workflow (research-closed-loop-feature-delivery.md:58-129, thread:c7pgm).
- No new research, outline, task, or protocol-spec artifact between the existing research and plan documents.
- No agent status, ownership, dependencies, messaging, retries, worktrees, or resume state; the harness owns them.
- No raw agent transcripts, routine progress logs, or new commands (thread:c50of).

## Implementation Phases

### Phase 1 — prove and tune autonomous research coverage

Run five fixed repository questions through baseline and treatment with paired LLM judges: draft-blind coverage and citation-bound evidence.
Keep model, context, and budgets identical.

Accept each real gap as the next Qn and repeat until no gap survives or the three-pass cap is reached.
Judges write contestable threads; `analyze` remains structural and human signoff authoritative.

Success criteria — automated: each trace records questions, judgments, cited evidence, and its termination reason for all five tasks.
Manual: blinded scoring shows treatment improves coverage on at least four tasks without increasing unsupported claims.

### Phase 2 — make the plan and its threads the review walkthrough

Keep the existing plan sections and budget; add no artifact.
Require every Current State and design claim to cite research or code, while exclusions cite their rationale and thread.

Each revision removes repetition and merges duplicate threads without dropping decisions or evidence.
Retain two to four high-priority threads naming the decision, consequence, and human ask.

Success criteria — automated: template fixtures enforce citations and budgets; no new CLI command exists.
Manual: a reviewer can narrate the shorter plan from its high-priority threads and peek evidence instead of reading research linearly.

### Phase 3 — finish in-context browser review

Retain the shipped sticky rail and point-and-click passage or section targeting.
Add click and keyboard citation peek for repository file-line and thread references, using the TUI's resolution semantics.

Render the cited excerpt in the rail, preserve the active thread, and show changed-since-last-review sections for the selected reviewer.
All writes continue through existing comment, reply, resolve, and verdict actions.

Success criteria — automated: browser tests cover safe citation resolution, changed-section state, and click targeting; `./scripts/ci.sh` passes.
Manual: review a revised plan without leaving the browser or manually entering a line number.

### Phase 4 — make plan threads the durable implementation handoff

Add a worker-to-root handoff convention to the skill, not the schema (research-closed-loop-feature-delivery.md:131-165, thread:cauxs).
The root delegates a cited phase and success criteria; each worker returns evidence, material discoveries or deviations, and human asks.

The root reconciles and anchors only durable evidence or questions; routine status remains in the harness.
Edit plan text only when a discovery changes an approved decision; a blocking thread preserves evidence and impact.

Success criteria — automated: a multi-agent fixture attributes each promoted finding to worker evidence and a plan phase; existing gates pass.
Manual: a fresh reviewer reconstructs changes and decisions without reading agent chats or runtime logs.

## Risks

- Better-looking documents can hide weak reasoning; coverage and faithfulness remain separate measures.
- Thread counts can be gamed by consolidation; reviewer time and missed decisions remain the outcome measures.
- The current dirty worktree overlaps browser, template, and skill files; settle it before implementing this plan.
