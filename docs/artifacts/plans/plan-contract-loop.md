---
comments:
    template: plan
status: draft
title: Plan Contract Loop
type: Plan
---

# Plan Contract Loop

## Overview

Make an approved comments plan the contract for implementation, enforced by the `comments-review` mod instead of plan mode.
When Claude hands a doc off for review, review opens beside the session and Claude's turn ends.
Code edits stay blocked until the human's approval matches the plan on disk.
After compaction, Claude gets back the contract's state, not its prose.

This is the first slice: hand-off, gate, and state; phase checks and the end review come later.

## Current State

- Plan mode is a prompt and no longer the default; practitioners use short plan files reviewed once early and once at the end. docs/research-notes/plan-mode-mods-2026-10.md:117 docs/research-notes/plan-mode-mods-2026-10.md:121
- Strong peers bind approval to a content hash, refuse approve verbs to the model, and fail closed. docs/research-notes/plan-mode-mods-2026-10.md:140
- Each verdict already records the document hash it approved. pkg/comment/types.go:145 pkg/comment/gate.go:139
- The review skill hands off with a blocking `comments watch --until signoff`. skills/review-comments/SKILL.md:105
- The mod lives only in the session's dev-mods folder, so its code is not in this repo or its review.

## Desired End State

Claude drafts a plan, review opens on its hand-off, and Claude stops.
Until an approved verdict's hash equals the plan's current hash, Claude's code edits are refused; editing the plan after approval locks them again.
Verify: the mod's tests pass, and one dogfood run in this repo shows each step.

## What We're NOT Doing

- No per-phase human checkpoints; phase checks become automated in a later slice.
- No drift detection, end-of-implementation review, or `prompt.compose` injection yet.
- No new CLI command or MCP tool; the parity rule stands. ./CLAUDE.md:116
- No change to how verdicts are recorded; the TUI stays the only approve surface.
- Not shipping the mod in the published plugin yet.

## Implementation Phases

### Phase 1 — Bring the mod into the repo and catch the hand-off

Move the mod from dev-mods into `mods/comments-review/` so changes are reviewed here, and load it with `--plugin-dir`.
Only a `plan` doc is handed to the human; research and design docs are supporting material the plan cites.

The mod catches Claude's `comments watch <plan> --until signoff` and opens the review beside the session.
It answers at once with "review is open, end your turn."
Until the verdict arrives it refuses Claude's non-read-only tools, so ending the turn is the only move.
The existing watcher then wakes Claude with the verdict.

#### Status
- 2026-10-06 — **active**
  - Summary: Mod moved to `mods/comments-review/`; plan hand-off intercepted; only reads run until the verdict.
  - Evidence: `claude plugin test mods/comments-review` 8/8 pass; `claude plugin validate` passes; tsc clean.
  - Next: Manual hand-off check in a dogfood run, then Phase 2.
- 2026-10-06 — **done**
  - Summary: Manual check passed: hand-off opened review in herdr, ended the turn; the verdict woke Claude and lifted the pause.
  - Evidence: this plan's own hand-off and approval; `go version` ran after the verdict.
  - Next: Phase 2.

**Success Criteria**
- automated: `claude plugin test mods/comments-review` covers the intercepted call, the tool lock, and the wake-up.
- manual: a hand-off in this repo opens the review and ends Claude's turn.

### Phase 2 — Gate code edits on the approved hash

A plan becomes active when Claude hands it off.
While it is active, Edit and Write outside the plan doc are refused until three things hold.
The latest verdict is `approved`, the gate passes, and its intent hash is current. pkg/comment/planstatus.go:72

Errors deny, with `/review-doc --unlock` as the escape.
The unlock runs only from an engine-stamped `composer` origin, the person's own Enter; commands have no model origin.
A status line shows plan, gate, and lock state.

#### Status
- 2026-10-06 — **active**
  - Summary: Gate built on `context --for implementation` freshness plus `gate`; fails closed; `--unlock` requires a composer origin; status line.
  - Evidence: `claude plugin test mods/comments-review` 9/9 pass, including lock, unlock, stale re-lock, error-deny, refused unlock.
  - Next: Manual check after re-approval: an edit is refused before approval and runs after.
- 2026-10-06 — **done**
  - Summary: Live run: after approval the gate opened and a test write ran; the before-approval refusal was not observed live because the approval came first.
  - Evidence: approval current, gate exit 0 at 21:20Z; before-approval refusal covered by the phase 2 plugin test.
  - Next: Phase 3.

**Success Criteria**
- automated: tests cover lock, unlock, re-lock on plan edit, error-deny, and a refused non-composer `--unlock`.
- manual: a pre-approval edit is refused; a post-approval edit succeeds.

### Phase 3 — Restore contract state after compaction

On compaction or resume, the mod re-adds a few lines: plan path, gate decision, lock state, open blocking count.
It never re-adds thread text or plan prose.

#### Status
- 2026-10-06 — **active**
  - Summary: `session.compact` keeps one fresh contract note; the active plan is saved per directory and restored on start, with a one-time reminder.
  - Evidence: `claude plugin test mods/comments-review` 10/10 pass, including the compaction and restore test.
  - Next: Manual check: run /compact and confirm Claude still names the plan and lock state.
- 2026-10-06 — **done**
  - Summary: After /compact Claude named the plan, gate and lock. Fixed two bugs it exposed: status edits staled approval; reloads repeated the note.
  - Evidence: `go test ./pkg/comment` passes with appended-status and legacy-hash tests; plugin test 10/10; `./scripts/ci.sh` green.
  - Next: Re-approve once (stored hash predates the fix); then plan phase checks.

**Success Criteria**
- automated: a test raises compaction and checks the re-added lines.
- manual: after `/compact`, Claude still names the active plan and lock state.

## Risks

- A fail-closed gate can strand a session if the mod breaks; mitigated by the user-only unlock.
- Intercepting `comments watch` changes the skill's blocking contract for other harnesses; mitigated by doing it only in this mod, leaving the CLI unchanged.
- Hash binding may lock after harmless edits such as reanchoring; accepted, since re-review is cheap.
