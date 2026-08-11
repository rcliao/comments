# Research: surfacing artifact quality through the skill loop

## Research Question

Q1. Which quality signals about an artifact does the loop already produce, and where do they surface (or fail to)?
Q2. What does the skill instruct agents to do with those signals today?
Q3. Which designed-but-unbuilt or unused signals would raise what the human actually sees at review time?

## Summary

Structure and evidence are now measured mechanically: caps, style, question coverage, citation resolvability. [Q1]
Content truth still has one instrument — the fresh-context reviewer — and its catches are recorded, then thrown away.

The skill has agents consume the human's signals — gate state, inbox replies, signoff decisions — but none of their own telemetry. [Q2]
Reviewer yield, verdict notes, and mid-implement gap threads all land in sidecars nothing reads back.

Three levers stand out, in leverage order. [Q3]
The unbuilt probe layer tests a doc by what a fresh reader can do with it.
The existing eval harness could be wired into the loop today.
The sidecars already permit a lead-lag join that would tell us which leading signal predicts a bad artifact.

## Findings

### F1 — Mechanical quality is now enforced; content truth is not [Q1]

`validate` checks caps (pkg/comment/template.go:296), prose shape (pkg/comment/style.go:119), question coverage (pkg/comment/coverage.go:11-21), and citation resolvability (pkg/comment/citations.go:90).
Structure and review state also surface through `gate --json` and `watch` NDJSON events — machine channels that carry doc state, not content quality.
`scripts/eval/check-doc.py` emits similar metrics as JSON for tuning.

Nothing checks whether a resolvable citation *supports* its sentence, or whether claims are true — the mechanical floor stops at the semantic wall.

### F2 — The reviewer is the only content instrument, and its yield is discarded [Q1] [Q2]

Fresh-context reviewer passes caught real defects in every RPI round this week — fabricated citation, half-adopted source, false negative-claim — recorded as `reviewer`-authored threads in docs/plan-rpi-loop-strength.md.comments.json, docs/research-agent-surface.md.comments.json and docs/plan-agent-surface.md.comments.json.
No prompt, allowlist, or pass-count learns from that record; each reviewer starts from the same skill prose regardless of what past reviewers caught or missed.

### F3 — The probe layer is designed, exampled, and unbuilt [Q3]

Faithfulness/coverage probes and the executability brief exist as design history (docs/plan-artifact-quality-eval.md) and as the worked example (examples/design-probe-eval.md).
They are the only proposed signal that tests an artifact by its function — what a fresh reader can reconstruct — rather than by its shape.

### F4 — Lagging truth accrues free and unjoined [Q1] [Q3]

Threads added to a plan during implementation are plan gaps by definition; orphan churn marks unstable sections; both persist in sidecars with timestamps.
Verdict notes and reply-pass decisions record the human's own read (ReviewRecord, pkg/comment/types.go:140).
No join exists between these lagging signals and any leading metric, so "which leading signal predicts a bad artifact" stays unanswerable.

### F5 — The skill consumes structure, not judgment [Q2]

The skill mandates producing judgment signals — anchored callouts, walkthrough priorities, reviewer passes, inbox-first reply processing — and validating structure before review.
It never instructs an agent to run the eval harness, read past reviewer yield, or check lagging signals from the last cycle before drafting the next artifact.

## Code References

- pkg/comment/style.go:119, coverage.go:11-21, citations.go:90 — the mechanical floor
- scripts/eval/check-doc.py — harness the skill never invokes
- docs/plan-artifact-quality-eval.md — probe-layer design (unbuilt)
- skills/review-comments/SKILL.md — the loop under study

## Open Questions

- ~~Build order~~ — decided in review (c6mv7): lead-lag join first, harness wiring second; probes wait until their value is explained and felt.
- ~~Reviewer statelessness~~ — decided with it: agents, including the reviewer, get the doc AND its thread history before the next round starts; fresh context applies to the draft, not to the review record.
- Do lagging signals stay per-repo telemetry, or aggregate across repos where the plugin runs?
