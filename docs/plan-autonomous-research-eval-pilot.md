# Plan: credible pilot for autonomous research convergence

## Overview

Turn the existing paired-eval skeleton into a reproducible decision artifact, run its three fixed cases, and make a reviewable Phase 4 recommendation.
The round separates research discovery from plan handoff quality, preserves blinded judgments, and records enough provenance to reproduce every score.
Dogfood begins only after the human reviews the completed result in the terminal.

## Current State

The harness defines three cases with four golden source questions each, paired baseline and treatment conditions, and strict source allowlists (../scripts/eval/autonomous-research/cases.json:3-36).
The scorer checks paired shape, coverage improvement, three no-regression guardrails, and blinded preference (../scripts/eval/autonomous-research/score.py:15-22,50-126).

The example result is not executable as written because plan coverage has a zero total, which the scorer rejects (../scripts/eval/autonomous-research/result-example.json:14-20,27-33; ../scripts/eval/autonomous-research/score.py:25-32).
The protocol requests randomized A/B labels but does not persist the assignment or evaluator provenance (../scripts/eval/autonomous-research/README.md:7-24).

## Desired End State

One immutable pilot bundle contains the run manifest, six artifact pairs, concealed label mapping, completed judgments, raw results, and scorer output.
Automated checks can reproduce every derived metric.
The human can inspect the bundle in `comments view`, decide whether the evidence earns dogfood, and leave a durable signoff.

## What We're NOT Doing

- No claim of statistical significance from three cases.
- No model provider or orchestration runtime added to the CLI.
- No tuning cases, golden questions, or promotion rules after seeing treatment output.
- No unblinded author scores accepted as human preference.
- No Phase 4 rollout when a guardrail fails, even if the treatment prose looks better.

## Implementation Phases

### Phase 1 — freeze an executable evaluation contract

Correct the example envelope and add a validator for case IDs, paired variants, artifact paths, positive coverage totals, exact model settings, repository revision, budget, and evaluator identities.
Split promotion semantics: source-question coverage must improve on a majority of cases; plan-to-research coverage becomes a no-regression guardrail.
Keep unsupported claims, execution questions, and human open threads as aggregate no-regression guardrails.

**Success Criteria**

- automated: fixture tests reject zero totals, missing provenance, duplicate variants, altered case definitions, and absent artifacts.
  Existing scorer and CI tests pass.
- manual: reviewer confirms the revised rule measures discovery separately from handoff completeness.

### Phase 2 — generate and blind the six paired runs

Freeze one manifest before generation.
Run baseline and treatment for all three cases with identical model, reasoning, revision, source allowlist, and budget.
Store research, plan, sidecars, semantic-pass logs, and failure status per run.
Randomize labels only after generation; keep the mapping outside the judge bundle until scoring finishes.

**Success Criteria**

- automated: manifest validation proves six complete runs and equal paired settings; artifact hashes remain stable after blinding.
- manual: spot-check one pair for source-boundary compliance without revealing its condition.

### Phase 3 — judge independently and publish the decision bundle

Use a source-derived coverage judge for golden questions and unsupported claims.
Give a cold implementation judge only each plan to count execution-blocking questions.
Have the human compare randomized pairs after mechanical metrics are frozen.
Unblind once, run the scorer, and preserve raw judgments plus its recommendation in a commented pilot report.

**Success Criteria**

- automated: recomputing the bundle produces identical metrics, guardrails, and recommendation.
  `./scripts/ci.sh` passes.
- manual: human reviews the report in the terminal and records `approved` only for `promote_to_dogfood`; any other result starts a new improvement plan.

### Phase 4 — perform one bounded dogfood run

If approved, use the treatment workflow on one real repository task.
Record accepted and rejected coverage questions, verifier findings, semantic passes, plan coverage, open human threads, and implementation follow-up.
Treat this as operational validation, not another chance to change the pilot result.

**Success Criteria**

- automated: research and plan analyses report `ready: true`; plan validation, review gate, and CI pass.
- manual: human confirms the workflow exposed uncertainty early enough and was easier to authorize than the prior process.

## Risks

- **Judge leakage** — filenames or thread authors may reveal variants.
  Mitigate with copied blinded bundles and a checked forbidden-token scan.
- **Metric gaming** — extra low-value questions can inflate coverage.
  Score only frozen golden questions and count review burden separately.
- **Incomplete runs** — failures can disappear from averages.
  Require every manifest entry and preserve failures as treatment outcomes.
