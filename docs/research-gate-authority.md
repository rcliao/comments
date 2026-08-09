# Research: What a review gate may safely block on

## Research Question

Q1. What do the surveyed SDD tools treat as authoritative, and what do they actually enforce?
Q2. What does our gate block on today, and who owns those signals?
Q3. What is currently absent that would let an agent gate itself?

## Summary

Spec Kit separates authority from enforcement on purpose: its constitution is declared non-negotiable and violations are automatically CRITICAL, yet `/speckit-analyze` is read-only and `/speckit-implement` never reads its findings. The plausible reason is that analyze is a language model's judgment, and blocking a developer on a model's opinion is a poor trade — enforcement is delegated to humans in pull-request review. Our gate blocks instead on thread state, which is not a model's opinion. But the ownership of that state is a convention rather than a mechanism: `EvaluateGate` never inspects authorship, and both write surfaces let a caller mark its own comment blocking and later resolve it. Nothing structural stops an agent from raising and clearing its own gate.

## Findings

### F1 — Spec Kit declares authority but delegates enforcement [Q1]

`/speckit-analyze` states "**Constitution Authority**: the project constitution is **non-negotiable** … Constitution conflicts are automatically CRITICAL and require adjustment of the spec, plan, or tasks — not dilution, reinterpretation, or silent ignoring." The same skill is "**STRICTLY READ-ONLY**", must "**NEVER modify files**", and asks before suggesting fixes: "(Do NOT apply them automatically.)" Its strongest action on a CRITICAL finding is to "Recommend resolving before `/speckit-implement`". The constitution template routes enforcement elsewhere: "All PRs/reviews must verify compliance" ([github/spec-kit](https://github.com/github/spec-kit), commit `684b3d8`).

### F2 — Progression gates on artifact existence, not artifact quality [Q1]

`/speckit-implement` runs `check-prerequisites.sh --require-tasks`, which resolves feature paths, prints them as JSON and exits 0; its only failure modes are a missing file or an unknown flag. It never reads the analysis report. A CRITICAL analyze run and a clean one therefore permit identical implementation. Existence is checked mechanically; quality is checked advisorily; the two are not connected.

### F3 — Our gate blocks on thread state and mechanical structure [Q2]

`EvaluateGate` (`pkg/comment/gate.go:39`) walks threads and branches on `Resolved`, `Blocking` and `IsPending` only. The CLI additionally fails the gate on template violations — word caps, sub-question coverage (`pkg/comment/coverage.go:75`), citation resolvability (`pkg/comment/citations.go:90`). None of these is a model's opinion: they are recorded state and deterministic checks over the document.

### F4 — Authorship is nowhere in the gate decision [Q2]

`EvaluateGate` never reads `Author`. A blocking thread raised by an agent and one raised by a human are indistinguishable to it. `Blocking` is settable by any writer on both surfaces (`pkg/mcp/types.go:44`, `cmd/comments/batch_add.go:23`), and `comments signoff` takes `--author` as a free string. The human ownership the gate appears to encode is carried by convention in the skill, not by the code.

### F5 — What is absent: any agent-versus-human distinction at write time [Q3]

The one place the distinction exists is thread resolution: `GuardZoneResolve` (`pkg/comment/actor.go:58`) refuses an agent resolving a thread in a `zone: human` section, using a TTY heuristic with an env override. That guard covers resolution only, and only inside declared zones. Nothing restricts who may create a blocking thread, resolve one outside a human zone, or record a signoff. An agent can therefore raise a blocking thread and clear it unaided.

## Code References

- `pkg/comment/gate.go:39` — the decision, and the fields it reads
- `pkg/comment/actor.go:58` — the only agent/human distinction in the codebase
- `pkg/mcp/types.go:44` — blocking is caller-settable
- `pkg/comment/citations.go:90`, `pkg/comment/coverage.go:75` — deterministic checks

## Open Questions

[NEEDS CLARIFICATION: Should agent-authored comments be structurally barred from being blocking, or is the convention in SKILL.md sufficient given a human reviews before signoff?]

Is signoff meant to be a human-only act? Today `comments signoff --author` accepts any string, so a loop could record its own approval.
