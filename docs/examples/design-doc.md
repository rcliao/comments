---
comments:
  template: design-doc
description: A bounded design for measuring the quality of research and plan artifacts.
status: stable
title: "Design: probe-based quality eval for RPI artifacts"
type: Design
---

# Design: probe-based quality eval for RPI artifacts

## Problem

The RPI loop measures its *process* — reviewer signal, passes to gate green — but nothing measures the *artifact*.
A research doc can be gate-clean and still be thin, wrong, or quietly narrower than its question.

Today the only detector is the human reviewer: exactly the expensive attention the loop exists to protect.
The failure mode is documented — prose that reads convincingly while its claims are unverified.
Affected: every RPI doc in this repo, and skill tuning with no quality signal to tune against.

## Goals / Non-Goals

**Goals**

- Score research docs on faithfulness (claims match code) and coverage (the question is fully answered), mechanically where possible.
- Feed scores to skill tuning; a failing probe drives a rewrite within word budgets, never expansion.

**Non-Goals**

- No inline gating: probes never block `gate` or `signoff`.
- No holistic LLM judge ("rate this doc 1–10") — judges only grade single probe answers.
- No new probe-runner infrastructure in Go; probes are skill-directed subagent work.

## Proposed Design

Three probe types, run post-signoff by the skill, results appended to a log the review surface never shows.

**Coverage probes**: a fresh-context subagent reads only the research QUESTION plus the repo, never the doc.
It writes five questions an ideal doc would answer, each with an expected answer and file:line evidence.
A second fresh subagent answers those questions from the doc alone.
A grader marks each answer against the key.
A doc that quietly narrowed its scope fails here; doc-derived probes cannot detect omission.

**Faithfulness probes**: a subagent given only the doc extracts five checkable claims.
The grader verifies each against the cited files, one claim at a time.
Convincing-but-wrong prose fails here.

**Executability probe** (plan docs): a cold subagent receives the plan alone and writes an execution brief.
The brief names files it would touch, their order, and every question it cannot answer.
It does not implement.
Score = forced-question count plus the human's read of brief-vs-intent divergence, observed rather than automated in v1.

Each run appends one JSON line to `.comments/evals.jsonl`, keyed by doc path and content hash.
A revised doc re-scores as a new entry, making rewrite-not-expand checkable: same path, fewer words, higher score.

## Data Flow

1. The **skill** (NEW) fires after a signoff lands: it reads the doc's research question and hands it to a generator.
2. A **generator** subagent (NEW) reads the question + repo — not the doc — and emits five coverage probes with an answer key, each key carrying file:line evidence.
3. An **answerer** subagent (NEW) receives only the doc and the five questions, and returns five answers. Runs concurrently with step 4.
4. An **extractor** subagent (NEW) receives only the doc and returns five checkable claims. (3 and 4 are the concurrent pair; everything else is sequential.)
5. A **grader** subagent (NEW) marks answers against the key and claims against the cited files, one item at a time, and emits pass/fail per probe.
6. The **skill** appends one eval record to the log and reports the failing probes to the drafting agent, which rewrites within word budgets.

Role names are not greppable in code — the actors are prompts in `skills/review-comments/SKILL.md`, not Go types.

## Data Model

```dbml
// NEW — one line per eval run, append-only
Table eval_record {
  doc_path string [pk]         // NEW · .comments/evals.jsonl · written step 6
  doc_hash string [pk]         // NEW · sha256 of content at scoring time
  probe_type string            // NEW · coverage | faithfulness | executability
  results json                 // NEW · pass/fail per probe id
  scored_at timestamp          // NEW
}

// NEW — held in the run only, never persisted; the log keeps outcomes
Table probe {
  id string [pk]               // NEW · p1..p5 within a run
  question string              // NEW · generator, step 2
  expected string              // NEW · answer key (coverage only)
  evidence string              // NEW · file:line the key rests on
}

// unchanged — adjacent: eval joins against it for lagging signals
Table review_record {
  author string                // pkg/comment/types.go:141
  timestamp timestamp [pk]     // pkg/comment/types.go:142
  decision string              // approved | changes_requested | commented
  note string
}

// unchanged — adjacent: mid-implement threads are the lagging quality signal
Table thread {
  id string [pk]               // pkg/comment/types.go:190 lookup
  line int
  resolved bool
  blocking bool
}

Ref: eval_record.doc_path > thread.id        // by doc file only; convention, nothing joins them
Ref: probe.id > eval_record.results          // by probe id inside the json; convention

// Table probe_history { }                   // should exist: probe reuse across
//                                           // runs is unmeasured without it
```

## Interfaces

NEW `comments eval log <doc.md>` — used by the human after Data Flow step 6; reads `eval_record`.

```
in:  doc.md path (must exist; no sidecar required)
out: the doc's eval records as JSON lines, newest first; exit 0 even when empty
err: exit 1 only when the path is unreadable — an unscored doc is empty, not an error
```

NEW skill contract "Artifact probes (post-signoff)" — fires Data Flow step 1; writes `eval_record`, creates `probe`.

```
in:  a signed-off doc + its research question (from the doc's Research Question section)
out: one appended line in .comments/evals.jsonl {doc_path, doc_hash, probe_type, results, scored_at}
constraints: never blocks gate/signoff; five probes per type; subagents are
fresh-context with input allowlists (generator never sees the doc)
```

CHANGED none — `gate`, `signoff`, `validate` contracts untouched; the eval reads the same sidecar they write (`review_record`, `thread`).

## Options Considered

### Option A — probes as skill prose + JSONL log (chosen)

- Pros: no new Go surface; fresh-context subagents already exist in the loop; log is greppable and diffable.
- Cons: probe quality depends on prompt quality; nothing enforces the skill actually runs probes.

### Option B — a `comments probe` Go command orchestrating subagents

- Pros: enforceable, versioned, testable in CI.
- Cons: the tool would have to spawn and manage LLM agents — a runtime dependency this repo has deliberately avoided; simpler prose version unproven first.

### Option C — holistic LLM-judge rubric

- Pros: cheapest to implement — one prompt.
- Cons: judges read convincing prose the way authors write it; a 6/10 gives no rewrite direction. Rejected on the failure mode this design exists to catch.

## Risks

- **Probe quality becomes the bottleneck** — bad generated probes measure nothing. Mitigated: generators are fresh-context and direction-split; unpredictive probe types get dropped after correlation against lagging signals. Accepted for v1.
- **Log divergence across machines** — `.comments/evals.jsonl` is per-checkout. Accepted: tuning telemetry, not shared truth.

## Definition of Done

- automated: one eval record lands in `.comments/evals.jsonl` for a real research doc, keyed by content hash; `jq` produces per-doc scores.
- automated: a doc revised after a failing probe re-scores as a new entry at the same path with a different hash.
- manual: one coverage-probe run on a real doc where the human agrees the five generated questions are fair; one executability brief judged against its plan.
- Out of this handoff: automation of the executability score; probe count tuning (starts at five, revisited after three scored docs).

## Unresolved Questions

- Should the eval log be committed like sidecars or remain untracked telemetry? The choice determines whether scores are shared across machines.
- Probe count of five is inherited from spec-kit's clarify quota, not evidence — revisit after correlation. (non-blocking)
