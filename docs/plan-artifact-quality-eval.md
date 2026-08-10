# Plan: artifact-quality eval for RPI docs (analyze command + probe protocol)

## Overview

Two deliverables: `comments analyze` — the mechanical layer (citation
resolution and two-way coverage between plan and research; quote-supports-claim
stays in the probe layer, since it resists mechanization) — and a probe
protocol the RPI skill runs per doc (faithfulness + coverage probes for
research docs, an execution-brief probe for plans), with scores appended to a
log outside the human review surface. Output exists to drive rewriting within
template word budgets, not to grade.

## Current State

- Citation parsing/resolution exists but is TUI-locked: `ParseReferences`
  extracts refs with positions (research-artifact-quality-eval.md:43-44),
  `buildRefMap` resolves them at load (research-artifact-quality-eval.md:45);
  nothing runs this outside the TUI (research-artifact-quality-eval.md:47).
- The sidecar records process, not artifact quality
  (research-artifact-quality-eval.md:31-39).
- Probe-based doc eval has direct precedent (QAGS) and a named axis split:
  doc-derived = faithfulness, source-derived = coverage; doc-derived alone
  rewards thin docs (research-artifact-quality-eval.md:64-70).
- Lagging signals accrue free in the plan sidecar during implementation
  (research-artifact-quality-eval.md:93-99).
- Decisions already made: tune-the-skill purpose, comments-repo home, per-doc
  probes, scores to a log not the sidecar, rewrite-not-expand
  (research-artifact-quality-eval.md:10-13).

## Desired End State

After any RPI doc is signed off, one command reports its mechanical health and
one skill-directed probe pass appends a scores line to `.comments/evals.jsonl`;
a doc revised after a failing probe gets denser, not longer (word budgets
unchanged). Verify: both new RPI docs from this feature are themselves scored,
and the log holds runs for at least three docs.

## What We're NOT Doing

- No inline gating: `analyze` and probes never block `gate`/`signoff`; scores
  are tuning signal (research-artifact-quality-eval.md:10-13).
- No holistic LLM-judge score — judges only at quote-vs-claim and probe-answer
  grading (research-artifact-quality-eval.md:86-90).
- No sidecar schema change: scores live outside it, so review UX is untouched.
- No probe-runner infrastructure in Go: probes are skill-directed subagent
  work; only `analyze` and the log format are code.
- No automated executability scoring in v1 (protocol below starts observed).

## Implementation Phases

### Phase 1: `comments analyze` — mechanical layer

New command: `comments analyze <doc.md> [--against <research.md>] [--json]`.
Lift resolution out of the TUI: move the `buildRefMap` pattern into
`pkg/markdown` or a new `pkg/analyze` so CLI and TUI share it
(research-artifact-quality-eval.md:102-103). Report: unresolved citations
(target missing/stale line), Current-State/design claims with no citation, and
with `--against`: research findings never cited by the plan + plan citations
pointing at nothing load-bearing (two-way coverage,
research-artifact-quality-eval.md:47-49). Exit 0 always (never a gate).

**Success Criteria**
- automated: unit tests per detection category; `analyze` on
  docs/plan-rpi-loop-strength.md --against docs/research-rpi-loop-strength.md
  runs clean or reports true positives only; suite green under -race
- manual: run on this plan/research pair; every reported finding is real

### Phase 2: probe protocol + eval log (skill prose)

Add an "Artifact probes (post-signoff)" section to SKILL.md. Probe generation
is agent-generated, two passes with fresh contexts: (a) coverage probes — a
subagent given ONLY the research question + repo (not the doc) writes 5
questions an ideal doc would answer, EACH with its expected answer and
file:line evidence (the answer key); (b) faithfulness probes — a subagent
given ONLY the doc writes 5 claims to verify against code. A third fresh
subagent answers (a) from the doc alone and a grader marks each against the
answer key; the same grader checks (b) against the cited files one claim at a
time (research-artifact-quality-eval.md:64-70). Scores append
one JSON line per run to `.comments/evals.jsonl`: doc path, doc hash, probe
type, pass/fail per probe, timestamp. The skill states the response rule:
failing probes drive a REWRITE within word budgets — never appended content
(research-artifact-quality-eval.md:10-13).

**Success Criteria**
- automated: `jq` over `.comments/evals.jsonl` yields per-doc scores; log
  lines carry doc hash so revised docs re-score as new entries
- manual: run the protocol on docs/research-artifact-quality-eval.md itself;
  probe failures (if any) produce a denser revision, not a longer one

### Phase 3: executability probe — human-observed v1

Protocol (crisp, settling research-artifact-quality-eval.md:117-118): a cold
subagent receives ONLY the plan doc and phase-1 scope, and must produce a
written execution brief — files it would touch, in what order, and every
question it cannot answer from the plan. It does NOT implement. Score =
question count + human's read of whether the brief matches plan intent
(observed, not automated — automation is a later plan if the observed version
proves predictive). The log line uses the Phase 2 envelope (doc path, doc
hash, probe type "executability", timestamp) with a type-specific payload:
question count, brief path, and the human's divergence read — so Phase 4's jq
join works across probe types.

**Success Criteria**
- automated: log line present for at least one plan doc
- manual: you read one execution brief against its plan and judge the
  divergence score fair

### Phase 4: correlate + settle

After ≥3 docs have log entries and one implementation completes: compare probe
scores at signoff against lagging signals (mid-implement comments, orphaned
threads — research-artifact-quality-eval.md:93-99). Keep the probes that
predicted gaps; drop or rewrite the ones that didn't. Fold protocol fixes into
SKILL.md; note the eval in root CLAUDE.md.

**Success Criteria**
- automated: a jq join of evals.jsonl and sidecar data produces the
  correlation table
- manual: your call per probe type: keep, adjust, drop

## Risks

- **Probe quality is the new bottleneck** — bad generated probes measure
  nothing. Mitigated: generators are fresh-context and split by direction;
  Phase 4 exists to kill unpredictive probes; accepted for v1.
- **Log divergence across machines** — evals.jsonl is per-checkout state.
  Accepted: it's tuning telemetry, not shared truth; commit it like the
  sidecars if history proves useful.
- **Analyze false positives on prose citations** — not every uncited sentence
  is a claim. Mitigated: v1 scopes the no-citation check to Current State
  sections only; widen later if precision holds.
