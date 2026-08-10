# Research: evaluating the quality of RPI research/plan artifacts

## Research Question

Phase-4 metrics measure the RPI *loop* (reviewer signal, passes, threads), not
the *artifact*. What should the comments repo measure to score the quality of a
research or plan doc itself — for tuning the skill (offline/batch, per-doc
probes acceptable), living in this repo?

Decisions (rcliao, 2026-08-07): tune the skill, don't gate docs inline; lives
in comments repo; per-doc probes fine; scores go to a log, not the sidecar
(tuning signal, not review material); eval feedback drives REWRITING within the
template word budgets — a failing probe makes the doc denser, never longer.

## Summary

Quality can be measured on three layers, two of which need no new judgment
machinery. (1) **Mechanical**: citation validity — the repo already parses and
resolves `file:line` references (`ParseReferences`, `buildRefMap`); what's
missing is only the check that the cited line supports the claim and a CLI to
run it outside the TUI. (2) **Predictive**: these docs are compressed context,
so score them by what a fresh agent can do with *only* the doc — answer
code-grounded probes (research) or start phase 1 without questions (plan).
LLM-judge rubrics are the design to avoid as the primary signal: judges read
convincing prose the same way authors do. (3) **Lagging**: the sidecar of a plan
doc keeps accruing evidence during implementation for free — comments added
mid-implement are plan gaps; orphaned/re-anchored threads track drift.

## Findings

### F1 — the sidecar records process, not artifact quality

`ReviewRecord` carries author/decision/note (pkg/comment/types.go:140-145);
threads carry status, blocking, resolution and full reply history. Run-1
extraction from today's two docs produced loop metrics (reviewer signal 7/7,
1 pass, 1-2 open threads faced — docs/plan-rpi-loop-strength.md.comments.json,
threads/reviews arrays) with plain JSON reading — but nothing in the
sidecar says whether a *finding-free* doc is good or merely unexamined. Artifact
quality needs a signal source outside the review conversation.

### F2 — citation machinery exists; only the judgment and CLI are missing

`ParseReferences` extracts every `path:line` and markdown-link reference with
its document position (pkg/markdown/refs.go:15-23,36), skipping code fences;
`buildRefMap` resolves them against disk once per load (pkg/tui/keys_refpeek.go:28-40).
Resolution today answers "does the target exist", not "does the target support
the claim", and nothing outside the TUI runs this resolution at all. The
mechanically checkable gaps are: unresolved citations, claims with no citation,
and research findings a plan never cites (coverage both directions). One check
resists mechanization — "does research-foo.md:23 actually support the sentence
citing it" — but it decomposes into one quote-vs-claim pair at a time rather
than whole-doc judgment.

### F3 — predictive probes score the doc by its function, not its prose

RPI's own framing: each phase produces "a compacted artifact that serves as
the input for the next phase" (ACE, ace-fca.md — source below). The eval that
matches that function: give a fresh agent ONLY the artifact, measure
reconstruction. This is established practice, not invention — QAGS (ACL 2020)
scores generated summaries by generating questions and comparing answers
against the source, and correlates with human factuality judgments better than
prior metrics. The QA-eval literature also names the axis that matters here:

- **Faithfulness** (doc-derived probes): questions generated from the doc,
  answered against the code — catches convincing-but-wrong prose. QAGS's
  direction.
- **Coverage** (source-derived probes): questions generated from the research
  QUESTION plus the repo, *before* reading the doc — a doc that quietly
  narrowed its scope fails these. Doc-derived probes alone cannot detect
  omission; a thin doc would score perfectly.

For a plan doc the probe is behavioral: a sandboxed cold agent attempts phase 1
from the plan alone; score = questions it is forced to ask + where it
improvises off-plan. Operationalizes plan.yaml's "could a reviewer tell what
will exist" as behavior instead of opinion. All probes are subagent-runnable
with strict allowlists — the isolation discipline the reviewer pass already
uses (skills/review-comments/SKILL.md, "Fresh-context reviewer pass").

Sources: https://github.com/humanlayer/advanced-context-engineering-for-coding-agents/blob/main/ace-fca.md ·
https://aclanthology.org/2020.acl-main.450/ (QAGS)

### F4 — pure LLM-judge scoring is the known failure mode

The plan-reading illusion — prose convincing while assumptions are wrong — is
documented for human readers (research-rpi-loop-strength.md:105-113); that it
applies to LLM judges too is this doc's inference, consistent with QAGS
existing precisely because direct model judgment of summaries proved
unreliable. Judges belong at the narrow points
where judgment is unavoidable (quote-supports-claim in F2; probe-answer grading
in F3), never as a holistic 1-10 doc score.

### F5 — lagging outcome signals accrue in the sidecar for free

During implementation the plan doc's sidecar keeps living: comments added
mid-implement are, by definition, things the plan failed to settle; threads
going orphaned mark sections that churned (pkg/comment/types.go:12-15 lifecycle
states; re-anchor cascade on document change). Correlating "probe score at
signoff" with "gaps surfaced during implement" is how we learn which leading
metrics actually predict quality — the tuning purpose from the interview.

## Code References

- pkg/markdown/refs.go:15-23,36 — Reference struct + ParseReferences (the analyze parser)
- pkg/tui/keys_refpeek.go:28-40 — buildRefMap resolution pattern to lift out of the TUI
- pkg/comment/types.go:140-145 — ReviewRecord (what signoff already stores)
- pkg/comment/types.go:12-15 — thread lifecycle states (lagging-signal source)
- skills/review-comments/SKILL.md — reviewer-pass allowlist discipline probes reuse
- docs/research-rpi-loop-strength.md:105-113 — the plan-reading illusion (judge risk)

## Open Questions

- ~~Where do scores live~~ — decided in review: a log outside the sidecar
  (see Decisions above).
- Probe generation source for research docs: hand-written per doc, or generated
  by an agent reading the code fresh? (Generated is scalable but the generator
  becomes its own quality question.)
- The plan-executability probe needs a crisp protocol (when does the cold agent
  stop? what counts as improvising?) — if the plan cannot make it crisp, it
  starts human-observed rather than automated.
