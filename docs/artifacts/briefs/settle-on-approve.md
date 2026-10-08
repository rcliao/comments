---
comments:
    template: brief
status: draft
title: Settle On Approve
type: Brief
---

# Settle On Approve

## Why

The first brief needed only two verdicts, but the end review still left seven threads open. Approving did not settle the agent's picks, so the strict gate never passed. The agent also left threads open that it could have closed. Quality came from fresh adversarial reviews, which the flow treats as optional.

## What

### Outcome

Approving the finished work settles it. Each open pick nobody objected to is accepted when the human approves. The agent closes every thread it fixed wherever it may. A fresh adversarial review of each invariant is a required check on every brief.

### Not doing

- No new command, and no agent path to settle a pick.
- No change to `changes_requested` or `commented`; only `approved` settles picks.
- No change to how picks are filed.

### Invariants

- Only a human verdict settles a pick.
- A pick with any reply from someone other than its author stays open.
- The TUI and the browser review record the same result.

### Stop and ask when

- Settling would need a second write path outside `AddReviewRecord`.
- The verdict dialog cannot show the open picks in its current box.

### Rejected

- A separate "accept all picks" key: one more step at the end, which is what this brief removes.

## Checks

- `go test ./pkg/comment -run SettlePicks` passes. Approve settles unreplied picks; a pick with a human reply stays open; `changes_requested` and `commented` settle nothing; `gate --strict` then passes.
- `go test ./pkg/webreview -run Verdict` passes, with a browser verdict that settles picks the same way.
- `go test ./pkg/tui -run Verdict` passes, with the dialog naming the open picks.
- `./scripts/ci.sh` prints `All CI gates passed.`
- A fresh-context reviewer tries to settle a pick without a human verdict and finds no way.
- Manual: the end review of this brief leaves no open threads.

## How

### Premises

- Both surfaces record through one core call. pkg/comment/gate.go:112 pkg/webreview/server.go:560
- Resolving a thread has one helper. pkg/comment/helpers.go:167
- The verdict dialog already shows gate counts. pkg/tui/keys_verdict.go:129

### Changes

- `AddReviewRecord`: on `approved`, resolve each open pick without a reply from another author, and add a reply saying the verdict accepted it.
- The verdict dialog lists open picks: "approving accepts N picks".
- Skill: close what you fixed wherever you may; a fresh adversarial review is a required check.
- Skill: a paraphrased Why gets a non-blocking thread, never a blocking one; approving the brief owns it.
- Brief template: the Checks criterion asks for that review.
- `ReplyToThreads` refuses an agent's resolve on an open pick (found by the adversarial review).
- The TUI refuses an approval when the open picks changed after the dialog rendered; the browser shows picks and the count.
- Do not touch suggestion handling or the review record's fields.

### Status

- 2026-10-08 — **done**
  - Summary: Approved at the end review. Every check passes except the manual one: one thread stays open.
  - Evidence: `gate --strict` exit 10 on `cd4ev` only, a note the agent left in a human zone, which neither the approval nor the agent can close.
  - Next: decide whether approval also closes the agent's unanswered notes in human zones, as it does picks.
- 2026-10-08 — **active**
  - Summary: Built. Approval settles unanswered picks in `AddReviewRecord`; the TUI dialog names the count; skill and template carry the required adversarial review.
  - Evidence: `go test ./pkg/comment -run SettlePicks`, `./pkg/webreview -run Verdict`, `./pkg/tui -run Verdict` pass; removing the settle call or the objection rule fails them; `./scripts/ci.sh` printed `All CI gates passed.`
  - Review: a fresh reviewer settled a pick with `reply --resolve`, then found the TUI could accept a pick filed after its dialog, and the browser accepted picks it never showed. All three are fixed and tested; each guard fails its test when removed.
  - Next: the end hand-off.
- 2026-10-07 — **pending**
  - Summary: Drafted; premises verified against the code.
  - Evidence: citations above resolve.
  - Next: Build after approval.
