---
comments:
    template: plan
description: Track implementation alignment, phase status, verification, and human attention without owning agent execution.
related:
    - path: ../research/week-long-agent-alignment.md
      relation: informed_by
status: draft
title: Lightweight Plan Ledger
type: Plan
---

# Lightweight Plan Ledger

## Overview

Make the reviewed plan a lightweight execution ledger without making Comments an executor.
Plan headings remain the work structure.
Each phase keeps an append-only Markdown status list containing state, summary, evidence, and next step.
A related as-built artifact closes the lightweight agreement lifecycle. docs/artifacts/research/week-long-agent-alignment.md:120-133

Comments validates and summarizes that visible plan state while threads remain the guardrail for deviations and human decisions. docs/artifacts/research/week-long-agent-alignment.md:178-190

## Current State

Comments already separates portable OKF knowledge, Markdown intent, and anchored review authority. docs/artifacts/research/week-long-agent-alignment.md:74-105

The plan template requires ordered phases and automated plus manual success criteria. pkg/comment/templates/plan.yaml:33-43

The sidecar stores threads, signoffs, hashes, and template identity, while the plan contains no implementation status convention.
Signoffs do not capture the reviewed document hash. pkg/comment/storage.go:16-24, pkg/comment/types.go:139-145

Fresh sessions can recover decisions, not the active phase or validation state. docs/artifacts/research/week-long-agent-alignment.md:161-176

## Desired End State

Each plan phase contains a human-readable, append-only status list that agents update through normal Markdown editing.
`comments context <plan> --for implementation` returns approval freshness, structured phase status, evidence, next steps, and unresolved human attention.
Verify by completing this implementation through its own status lists, then reconstructing the work without chat history.

## What We're NOT Doing

- No arbitrary tasks, assignees, dependencies, estimates, deadlines, projects, or portfolio views. docs/artifacts/research/week-long-agent-alignment.md:135-146
- No workers, queues, retries, scheduling, logs, remote execution, or runtime adapters. docs/artifacts/research/week-long-agent-alignment.md:148-159
- No checkpoint or progress write command and no execution state in the sidecar; agents edit Markdown directly.
- No TUI or browser progress UI in the first dogfood; `context` establishes the read contract.
- No semantic judgment that evidence proves a claim; progress reports recorded state and references.
- No expansion toward Plannotator-style code review or workspace sharing. docs/artifacts/research/week-long-agent-alignment.md:106-118
- No notification service or claim that Comments itself is always on. docs/artifacts/research/week-long-agent-alignment.md:192-206

## Implementation Phases

### Phase 1 — define status as a plan-template convention

Parse plan H3 phases and optional H4 `Status` sections.
Each dated entry nests Summary, Evidence, and Next; latest wins.

Guide new phases through the template and example.
Warn on missing or malformed status so historical plans stay valid.
Exclude Status from normal word caps; apply separate entry and length caps.

#### Status

- 2026-08-26 — **pending**
  - Summary: Revised after review.
  - Evidence: thread:cwuik, thread:c4un6
  - Next: Implement after approval.
- 2026-08-26 — **active**
  - Summary: Implementing the approved status-list contract.
  - Evidence: thread:cwuik, thread:c4un6
  - Next: Add parsing, template guidance, and separate status caps.
- 2026-08-26 — **done**
  - Summary: Status lists parse deterministically and use an independent budget.
  - Evidence: plan status and template unit tests pass.
  - Next: Preserve approval across status-only edits.

**Success Criteria**

- Automated: parser tests cover valid, missing, malformed, fenced, and duplicate entries; existing plan fixtures remain valid.
- Manual: phase discovery returns these four phases and the latest entry from each list.

### Phase 2 — preserve approval while detecting plan drift

Human approval covers phase definitions, success criteria, scope, and design—not progress updates.
Record full document and plan-intent hashes on new signoffs.
The intent hash excludes Status sections, so routine updates preserve approval.
Changes to approved intent make approval stale and ask for human attention.
Legacy signoffs remain readable but report freshness as unknown.

#### Status

- 2026-08-26 — **pending**
  - Summary: Approval boundary clarified.
  - Evidence: thread:ct2pb
  - Next: Implement after Phase 1.
- 2026-08-26 — **done**
  - Summary: New signoffs store document and stable-intent hashes.
  - Evidence: approval freshness unit tests pass.
  - Next: Expose alignment through implementation context.

**Success Criteria**

- Automated: review round trips cover full and intent hashes; status-only edits stay current; intent edits become stale; legacy records remain compatible.
- Manual: append Phase 1 status after signoff and confirm approval remains current.

### Phase 3 — guide implementation through context

Add `implementation` to the existing context modes.
Return approval freshness plus ordered phases, latest status, history count, success criteria, structural warnings, and high-priority or blocking threads in each phase.
Reuse CLI and MCP context surfaces; do not add a second status command.

Overall status is deterministic: unapproved, stale, attention-needed, aligned, or complete.
Blocked state, failed evidence, or an unresolved guardrail thread needs attention.

#### Status

- 2026-08-26 — **pending**
  - Summary: Context contract revised.
  - Evidence: thread:cwuik
  - Next: Implement after Phase 2.
- 2026-08-26 — **done**
  - Summary: CLI and MCP now share structured implementation context.
  - Evidence: context and MCP package tests pass.
  - Next: Dogfood the ledger and run full verification.

**Success Criteria**

- Automated: context tests cover each approval and overall state, missing status lists, phase-scoped attention, and CLI/MCP parity.
- Manual: one context call identifies the active phase, evidence, next step, and human ask without reading the sidecar.

### Phase 4 — dogfood the ledger and close the loop

Use this plan as the first live ledger.
Append status entries after every implemented phase, run full CI, and create a related as-built concept containing behavior, evidence, deviations, and remaining gaps.
Keep routine execution chatter out of both artifacts.

#### Status

- 2026-08-26 — **pending**
  - Summary: Awaiting implementation.
  - Evidence: —
  - Next: Dogfood after Phase 3.
- 2026-08-26 — **active**
  - Summary: Verifying the live ledger and documenting the shipped behavior.
  - Evidence: implementation context reads all four phase lists.
  - Next: Run full CI and write the related as-built concept.
- 2026-08-26 — **done**
  - Summary: The live ledger, documentation, and as-built handoff are complete.
  - Evidence: full CI passed; as-built plan-ledger concept validates.
  - Next: Monitor the next multi-day project for workflow friction.

**Success Criteria**

- Automated: `./scripts/ci.sh` passes; implementation context reports every phase done with current approval and evidence.
- Manual: a fresh agent reconstructs outcome and next action using OKF context alone, without this conversation.

## Risks

- Status lists can become execution logs; cap entries and words while preserving milestone evidence.
- Agents can overstate progress; Comments reports recorded claims and lets evidence plus threads challenge them.
- First-version CLI output may not be the best human surface; dogfood before adding UI.
