---
comments:
    template: brief
status: draft
title: One Brief
type: Brief
---

# One Brief

## Why

comments is becoming a dedicated planning tool, and planning with it keeps getting heavier for little benefit. An hour of planning for an hour of work is worse than letting the agent build and iterate. Explore a lightweight brief to align on, the way a team plans and syncs, and then build autonomously.

## What

### Outcome

One template, `brief`, carries a piece of work from idea to finish. Each section says who owns it: Why, What, Shape and Checks are the human's; How is the agent's.
There are two verdicts, on the brief and on the finished work. Between them the agent keeps How current and files picks. The human steers with threads, never a verdict.

### Not doing

- No chain of documents; research becomes the agent's notes, cited from How.
- No approval per phase and no reviewed step lists.
- No hosted surface; the brief stays a file in the repo.
- No deleting old templates or commands yet; they stop being recommended first.

### Invariants

- Only a human records a verdict. pkg/comment/gate.go:139
- An approval stays current exactly while Why, What, Shape and Checks are unchanged.
- Edits to How never stale an approval.
- Existing docs and sidecars keep loading and gating as before.

### Stop and ask when

- An invariant would have to bend.
- A premise in How turns out false.
- A check cannot be made into a command.
- The work needs anything outside the repo.

### Rejected

- Bigger plans so the agent can run longer: they produced four approvals for one plan. docs/artifacts/plans/plan-contract-loop.md
- Diffing an approved plan to allow amendments: a fresh review found five ways to fake approval. pkg/comment/planstatus.go:314

## Shape

- Data model: a verdict also records the template it was approved under; nothing else in the sidecar changes.
- Data flow: the human approves, the verdict stores the hash of Why, What, Shape and Checks. Each contract check recomputes that hash under the recorded template.
- Interfaces: `add --pick` stays; `amended` freshness goes; no new command.

## Checks

- `./scripts/ci.sh` prints `All CI gates passed.`
- `go test ./pkg/comment -run TestBriefApproval` passes. It edits each human section the ways an agent could: in place, deletion, heading rename, frontmatter, and text that only looks like status. Every case must read stale; a How edit reads current.
- `go test ./cmd/comments -run Parity` passes, still with no signoff or accept command. cmd/comments/parity_test.go:31
- A script gates every doc under `docs/artifacts` before and after; the decisions match.
- `claude plugin test mods/comments-review` passes, with a brief hand-off that pauses and wakes.
- Manual: the next real piece of work here uses one brief and at most two verdicts.

## How

### Premises

- Tiers already exist on template sections, so the hash can be scoped by tier. pkg/comment/template.go:100
- Project templates load from `.comments/templates`; this brief is the first one.
- The mod recognises only `template: plan` as a hand-off. mods/comments-review/hooks/register.tsx:368
- Zones are looked up by heading in current text, so a rename can hide one. pkg/comment/template.go:492

### Changes

- The approval hash covers the frontmatter, every heading, and the human sections' bodies, from the approved template, not the current one. pkg/comment/brief_contract.go:20
- Remove `amended` and its baseline diff; keep picks.
- The mod treats `brief` like `plan`.
- Rewrite the skill to about 100 lines around the brief.
- Mark research, research-deep, plan, design-doc, rfc, adr and mini as legacy in their descriptions.
- Do not touch verdict recording, the TUI verdict dialog or the sidecar format.

### Steps

1. Write the TestBriefApproval cases, failing; then the tier-scoped hash until they pass.
2. Mod hand-off for briefs; status line shows tiers and open picks.
3. Skill rewrite; legacy labels; docs.
4. The gate comparison script, then the full checks.

### Status

- 2026-10-07 — **pending**
  - Summary: Drafted as its own first example; awaiting the one sitting.
  - Evidence: `comments validate` passes against the brief template.
  - Next: Step 1 after approval.
- 2026-10-07 — **active**
  - Summary: Steps 1-4 built. Brief hash covers frontmatter, outline and human sections; `amended` deleted; mod hands off briefs; skill rewritten, old one kept as legacy.md.
  - Evidence: TestBriefApproval 11 stale and 3 current cases pass, and a mutation fails 8; `./scripts/ci.sh` All CI gates passed; plugin test 11/11; gate-compare 13 docs, 0 differ, 1 new.
  - Next: Fresh adversarial review of the hash, then the end hand-off.
- 2026-10-07 — **active**
  - Summary: A second adversarial review found four holes (project template, unparsed headings, blank lines, preamble); the hash now covers everything but How's body, under the embedded template.
  - Evidence: TestBriefApproval 15 stale, 5 current and a planted-template case pass, all red first; `./scripts/ci.sh` and gate-compare below.
  - Next: End hand-off.
- 2026-10-07 — **done**
  - Summary: Approved at the end review, the second verdict on this brief; the manual check (one brief, at most two verdicts) holds.
  - Evidence: the sidecar's two verdicts; freshness `current` under `template: brief`.
  - Next: Settle the four open picks; then the lessons in the end-of-run report.
