# Plan: autonomous research convergence before implementation

## Overview

Give agents a repeatable research loop that can discover new questions, verify claims, and prove plan coverage before involving the human.
The CLI exposes deterministic analysis; independent agent roles supply semantic judgment through ordinary threads.
One human reviews and signs off the plan before implementation begins.

## Current State

The validator maps declared `Qn` questions to tagged findings but cannot discover a concern absent from both sections (research-autonomous-research-convergence.md:23-33).
MCP validation is weaker than CLI validation because it omits filesystem citation checks (research-autonomous-research-convergence.md:30-32).

The autonomous skill has a generic reviewer, but no source-derived question pass or rule that restarts research when a real gap appears (research-autonomous-research-convergence.md:35-41).
The earlier probe design separates source-derived coverage from draft-derived faithfulness, but runs after signoff rather than before planning (research-autonomous-research-convergence.md:52-59).

## Desired End State

An agent can show that its latest research round added no accepted questions, every material claim survived evidence review, and every accepted finding is cited or excluded by the plan.
Verify with one dogfood run whose plan opens with no hidden research blockers and receives human signoff before code changes.

## What We're NOT Doing

- No model score inside `comments gate`; semantic findings remain contestable threads (research-autonomous-research-convergence.md:43-50).
- No built-in LLM provider, orchestration runtime, or dependency on one agent product.
- No automatic implementation after a plan merely validates; human plan signoff remains required.
- No permanent pass cap chosen from intuition; the eval measures convergence cost before dogfood.
- No change to the unresolved Phase 1 skill-router proposal in `thread:plan-landscape-improvements.md#cb5gr`.

## Implementation Phases

### Phase 1 — make validation identical on CLI and MCP

Add one path-aware `pkg/comment` helper combining template and optional citation checks.
Use it from CLI `validate`, MCP `comments_validate`, and both gates without changing existing JSON fields.

**Success Criteria**

- automated: parity fixture proves both surfaces return identical structural, coverage, marker, and citation rules.
  `go test -race ./...` passes.
- manual: an MCP agent sees and corrects the broken citation without using the CLI.

### Phase 2 — add deterministic `comments analyze`

Add `comments analyze <doc.md> [--against <research.md>] [--json]` and MCP parity.
Report questions, finding headings, and citation violations even when clean.
With `--against`, classify research findings as cited, explicitly excluded, or uncovered from resolved line references.
Return advisory findings and `ready`; never change gate state.

**Success Criteria**

- automated: parser tests cover multi-question findings, ranges, fences, exclusions, and an uncovered finding.
  CLI/MCP JSON is equivalent.
- manual: analyze this plan against its research and inspect every coverage edge.

### Phase 3 — build and evaluate the convergence loop

Add two pre-plan roles.
A draft-blind coverage scout proposes missing questions with answers and evidence; an evidence verifier checks draft claims against sources.
Accepted gaps add the next `Qn`; rejected candidates become resolved rationale (research-autonomous-research-convergence.md:52-68).
Repeat until validation passes, no question is added, and no verifier blocker remains; expose any capped survivors.

Before Phase 4, run 3-5 fixed tasks through current and new workflows with the same model and inputs.
Blind-score source-derived question coverage, unsupported claims, plan-to-research coverage, cold-agent execution questions, review-thread count, and passes to convergence.
Promote only if most tasks improve coverage without faithfulness or review-burden regression.
Store fixtures and JSON results under `scripts/eval/`.

**Success Criteria**

- automated: contract tests pin the protocol; the eval emits comparable baseline/treatment JSON.
- manual: a blinded human scores artifacts without variant labels and makes the Phase 4 go/no-go call.

### Phase 4 — dogfood after the eval passes

Begin only after Phase 3 meets its promotion rule.
Require `comments analyze plan.md --against research.md` before human review.
Every finding is cited or explicitly fenced out with rationale.
Carry only scope and priority uncertainty as high-priority plan threads.
The TUI keeps research citations and thread history available, preserving the plan as the human authorization surface (research-autonomous-research-convergence.md:70-77).
Update README, USAGE, architecture, and plugin version after the dogfood run.

**Success Criteria**

- automated: analyze reports `ready: true`; plan validation and gate are green.
  `./scripts/ci.sh` passes.
- manual: human confirms research coverage and convergence in `comments view docs/plan-autonomous-research-convergence.md`, then signs off before implementation.

## Risks

- **Coverage scout anchoring** — repository access may still bias it toward obvious code paths.
  Mitigate with a draft-free allowlist and evidence required for every proposed question.
- **False completeness** — a paired improvement does not prove truth.
  Keep the trace reviewable and label cap exhaustion explicitly.
- **Analyze precision** — a finding may be intentionally irrelevant.
  Count an explicit exclusion with a research citation as covered, never infer intent from absence.
