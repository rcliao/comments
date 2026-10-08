---
comments:
    template: plan
related:
    - path: ../../research-notes/plan-mode-mods-2026-10.md
      relation: informed_by
status: draft
title: Decide Once
type: Plan
---

# Decide Once

## Overview

Cut human sign-offs to two: approve the plan once, review the finished work once.
In between, the agent files each decision it makes alone as a thread carrying its pick, and keeps going.
An amendment to an approved plan becomes such a pick instead of a new approval.
The human settles open picks at the end review.

## Current State

- One small plan took four approvals; the agent's amendments after approval caused two of them. docs/artifacts/plans/plan-contract-loop.md
- The field gates the decision and the outcome only; mid-run questions go to a queue with the agent's pick. docs/research-notes/plan-mode-mods-2026-10.md:205
- A reply pass already exists: `r` records `commented`. pkg/tui/keys_verdict.go:82
- Any intent change since the verdict marks approval stale. pkg/comment/planstatus.go:72
- Changes since a verdict are already computed per section. pkg/comment/baseline.go:104

## Desired End State

A normal feature needs two verdicts: the plan and the finished work.
The agent amends an approved plan without a new verdict when each changed section carries an open pick, and no human-zone section changed.
Open picks fail `comments gate --strict`, so the end review must settle them.

## What We're NOT Doing

- No evidence-anchored end review or plan-against-code check yet; next slice.
- No change to who records verdicts; picks never approve anything.
- No new CLI command or MCP tool; `pick` is a field on `add`.
- No amendment of `zone: human` sections without re-approval.

## Implementation Phases

### Phase 1 — Picks

`add` (pkg/comment/add.go:10) gains a `pick` field: the option the agent proceeds with unless the human objects.
A pick is never blocking, so the default gate ignores it, while `--strict` (pkg/comment/gate.go:62) fails on it.
Inbox and the TUI show the pick; resolving it accepts, a reply objects.

#### Status
- 2026-10-07 — **done**
  - Summary: `add --pick` stores a pick; a blocking pick is refused; inbox and the TUI show it.
  - Evidence: pick_test.go and TestThreadViewShowsOpenPick pass; CLI check: gate 0, strict 10.
  - Next: Phase 2.

**Success Criteria**
- automated: tests cover add with pick, refusal of pick plus blocking, the strict gate, and CLI/MCP parity.

### Phase 2 — Amended approval

Approval freshness gains `amended`.
It applies when each section changed since the verdict has an open pick posted after it.
No changed section may be in a human zone (pkg/comment/template.go:492).
The contract gate (mods/comments-review/hooks/register.tsx:410) treats `amended` as open; anything else stays `stale`.

#### Status
- 2026-10-07 — **done**
  - Summary: Freshness `amended` when every changed section has a post-verdict pick and no human zone changed; the mod opens edits on it.
  - Evidence: amend_test.go (covered, uncovered, resolved, human zone, pre-verdict pick) and plugin test 10/10 pass.
  - Next: Phase 3.
- 2026-10-07 — **blocked**
  - Summary: Withdrawn. A fresh review found five ways to fake an amended approval; the code is deleted, superseded by the brief's section-scoped hash.
  - Evidence: docs/artifacts/briefs/one-brief.md; pkg/comment/brief_test.go.
  - Next: None; picks (Phase 1) stay.

**Success Criteria**
- automated: tests cover covered, uncovered, and human-zone amendments, and the mod's lock for each.

### Phase 3 — Skill and mod wiring

The skill tells the agent to fold replies in before hand-off, file picks rather than ask, and not request re-review for amendments.
The mod's contract note shows the open pick count.

#### Status
- 2026-10-07 — **done**
  - Summary: Skill gains "Decide once"; contract note shows open picks and points at `add --pick`.
  - Evidence: plugin test asserts the pick count; `./scripts/ci.sh` all gates passed.
  - Next: Observe on the next real plan — manual criterion measured in practice.

- 2026-10-07 — **blocked**
  - Summary: Superseded. The skill was rewritten around the brief; the pick count in the contract note stays.
  - Evidence: skills/review-comments/SKILL.md; mods/comments-review/tests/review.test.ts.
  - Next: None.

**Success Criteria**
- automated: plugin test checks the pick count in the note.
- manual: the next real plan in this repo needs at most two verdicts.

## Risks

- An agent can amend widely and cover it with picks; mitigated by the human-zone rule and the strict end gate.
- Picks may pile up unread; the end review lists them all, and `gate --strict` will not pass until each is settled.
