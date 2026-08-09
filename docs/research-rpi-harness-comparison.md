# Research: How our RPI harness compares to the state of the art

## Research Question

Q1. What do the leading spec-driven harnesses enforce at the research/plan review boundary?
Q2. What does our harness enforce that they do not?
Q3. What do they provide that we lack?
Q4. What about our harness is still unverified?

## Summary

The published SDD tools converge on the same artifact chain — spec, plan, tasks — and on the claim that a human reviews the artifact before code exists. They differ from us in where that review has teeth. Spec Kit ships `/speckit.analyze` and `/speckit.checklist` as *analytical* commands that report and do not block; Kiro states approval gates between phases but reviews the markdown through pull requests. Neither exposes a machine-readable verdict an agent can wait on, nor threaded comments anchored inside the artifact. Our harness inverts that: review state is structured data, the gate is an exit code, and structure is checked mechanically. Their breadth is greater — task and implementation phases — but ours is a deliberate fence around the step that actually costs human attention. The guards themselves are only partly validated: one paired A/B, n=3, and no adoption outside this repo.

## Findings

### F1 — Spec Kit's cross-artifact checks report, they do not block [Q1]

Installed at commit `684b3d8`. `/speckit-analyze` declares itself "STRICTLY READ-ONLY: Do not modify any files", and its Next Actions say only "If CRITICAL issues exist: Recommend resolving before `/speckit-implement`"; remediation is offered, never applied. `/speckit-implement` gates on `check-prerequisites.sh --require-tasks`, which prints paths as JSON and exits 0 — it verifies plan.md and tasks.md exist, and never reads analyze's findings. Nothing in the scaffold mentions threads, comments or signoff ([github/spec-kit](https://github.com/github/spec-kit)).

### F2 — Kiro gates on phases but reviews through pull requests [Q1]

Kiro produces `requirements.md` (EARS notation), `design.md`, and a dependency-ordered `tasks.md`, and positions design as "the document your team reviews before a single line of production code gets written". Review happens on specs that are "committed, versioned, reviewable in pull requests"; the source does not describe inline commenting or threaded discussion on the artifact itself ([doit.com](https://www.doit.com/blog/spec-driven-development-with-kiro-ai-code-ownership)). Approval is a phase boundary, not a queryable state.

### F3 — Our review state is structured data with an exit code [Q2]

Review lives in a sidecar as nested threads (`pkg/comment/types.go:64`), not prose. `EvaluateGate` (`pkg/comment/gate.go:39`) reduces that state to a decision surfaced as exit code 10 (`pkg/comment/gate.go:23`); a signoff is an appended record (`pkg/comment/gate.go:112`). It reduces thread state and signoff records, not prose or a model's report — though nothing in `EvaluateGate` inspects authorship.

### F4 — Structure is checked mechanically, including omission [Q2]

Templates cap words per subsection (`pkg/comment/template.go:84`), cross-check that every enumerated sub-question is answered (`pkg/comment/coverage.go:75`), and verify that cited evidence resolves (`pkg/comment/citations.go:90`). Sections may be marked human-owned, and an agent caller is refused there (`pkg/comment/actor.go:58`). No surveyed tool checks omission or citation resolvability.

### F5 — They carry the phases after planning; we stop by design [Q3]

Spec Kit and Kiro both continue past planning into task decomposition and an implementation driver, and both ship broader surfaces (Kiro an IDE, Spec Kit a constitution phase). Our templates cover research and plan only. Per the project owner that fence is the bet, not a gap: the expensive step is a human reading heavy agent-written documentation, so the author distills before spending another engineer's attention. Tasks after an agreed plan are typically agent-managed or tracked as tickets.

### F6 — The guards are measured, not proven [Q4]

One paired A/B, three questions, two arms. Subsection caps caught real sprawl in 1 of 3 controls; question coverage helped in 1 of 3, and the single trial that motivated it did not replicate. Citation ambiguity was the strongest signal — 15 of 25 references unpeekable in one document. Nothing yet measures whether a reviewed artifact yields better implementation, which is the actual claim.

## Code References

- `pkg/comment/gate.go:23,39,112` — decision, exit code, signoff record
- `pkg/comment/coverage.go:75` — sub-question coverage
- `pkg/comment/citations.go:90` — citation resolvability
- `pkg/comment/actor.go:58` — human-zone enforcement

## Open Questions

[NEEDS CLARIFICATION: Is stopping at plan signoff a deliberate scope fence, or should the harness grow a task phase to match Spec Kit and Kiro?]

Does the comparison need a hands-on trial of Spec Kit and Kiro before we rely on it? Every claim above is from published documentation, not from running either tool.
