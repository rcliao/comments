# Research: autonomous research convergence before plan review

## Research Question

Q1. What can comments already verify while an agent researches autonomously, and where does that verification stop?
Q2. How can agents validate claims, discover missing coverage, and add research questions without turning model judgment into gate authority?
Q3. What evidence should cross into a plan so one human review can safely authorize implementation?

## Summary

comments already enforces the questions an author remembered to ask, citation resolvability, template shape, and explicit review state. [Q1]
It cannot discover an omitted question or decide that a citation supports a claim.

The missing unit is a convergent research protocol.
An evidence-first coverage scout proposes candidate questions, a separate verifier challenges claims, and the author records accepted gaps as new numbered questions before repeating deterministic checks. [Q2]
Threads preserve rejected questions and disagreements without granting those model judgments gate authority.

The plan remains the only human gate.
It should cite every accepted research finding, carry only scope or priority uncertainty as high-priority threads, and pass a two-way research-to-plan coverage check before implementation. [Q3]

## Findings

### F1 — the CLI verifies declared coverage, not problem coverage [Q1]

Research templates require numbered questions and tagged findings.
`validateQuestionCoverage` reports a declared question with no finding and a finding tag with no declared question (pkg/comment/coverage.go:73-121).
This catches omission between two sections of the same draft, but only after the author chose the question set.
It cannot identify a concern that appears in neither section.

CLI citation validation similarly checks that a token resolves and its line exists; it does not judge whether the referenced text supports the sentence (pkg/comment/citations.go:13-18,90-148).
The MCP validator currently omits even that check: CLI `validate` appends `ValidateCitations`, while `comments_validate` returns after `ValidateTemplate` (cmd/comments/template.go:136-146, pkg/mcp/template.go:24-40).
Autonomous agents therefore receive a weaker result on their primary surface.
The deterministic floor is useful precisely because its limits are explicit.

### F2 — autonomous research currently has review, but no research-specific convergence contract [Q1] [Q2]

The skill already defaults to research without a human gate and requires a fresh-context reviewer (skills/review-comments/SKILL.md:282-304).
The reviewer protocol asks which research findings a plan drops and which claims are uncited (skills/review-comments/SKILL.md:349-369).
It does not assign an independent pass to derive missing research questions from the repository, distinguish evidence disagreement from coverage gaps, or define when a new question restarts investigation.

The loop therefore converges on thread state, but its semantic research coverage depends on whatever the author and generic reviewer happened to notice.

### F3 — semantic findings belong in threads, not deterministic gate rules [Q2]

The gate is intentionally authoritative only over recorded thread state and mechanical template checks (docs/research-gate-authority.md:23-33).
A model-generated claim such as “the research missed the retry boundary” is contestable; making it an intrinsic gate violation would convert model opinion into tool authority.

An agent reviewer can instead post the claim as a blocking thread with evidence.
The author must apply it, rebut it, or leave it for the human.
The existing sidecar then preserves the challenge, response, and resolution as the next round's memory (skills/review-comments/SKILL.md:253-274).

### F4 — question generation needs a source-derived pass [Q2]

The earlier artifact-eval plan already separates source-derived coverage probes from document-derived faithfulness probes (docs/plan-artifact-quality-eval.md:68-82).
Its coverage generator sees the question and repository but not the draft, preventing the draft's framing from defining what “complete” means.
Its faithfulness pass starts from claims in the draft and verifies them against code.

Those directions answer different failures: missing questions versus unsupported answers.
Moving both before plan drafting makes them a convergence mechanism rather than post-signoff telemetry.

### F5 — accepted gaps must expand the original question set [Q2]

Appending an untracked finding would recreate the omission problem in reverse.
When a coverage scout finds a real gap, the author should append the next `Qn` to Research Question, investigate it, and tag one or more findings with that ID.
The existing coverage validator then makes the newly accepted scope mechanically visible (pkg/comment/coverage.go:73-121).

Rejected candidates should become resolved threads that state why they are irrelevant or duplicate.
This preserves negative coverage decisions and prevents a later round from silently reopening them.

### F6 — the plan is the human authorization surface [Q3]

The current autonomous-chain decision puts the human's value at plan review and pauses early only when uncertainty changes scope or direction (skills/review-comments/SKILL.md:282-304).
That boundary remains sound if the handoff exposes research coverage instead of hiding it.

Every accepted finding should be cited by the plan or explicitly fenced out.
Surviving scope and priority calls become high-priority plan threads; implementation details stay agent-owned.
Only a green, signed-off plan authorizes phase-by-phase implementation (skills/review-comments/SKILL.md:338-347).

## Code References

- pkg/comment/coverage.go:73-121 — deterministic declared-question coverage
- pkg/comment/citations.go:90-148 — citation resolution boundary
- cmd/comments/template.go:136-146, pkg/mcp/template.go:24-40 — CLI/MCP validation mismatch
- skills/review-comments/SKILL.md:282-369 — autonomous chain and current reviewer pass
- docs/research-gate-authority.md:23-33 — why semantic judgment stays outside the gate
- docs/plan-artifact-quality-eval.md:68-82 — two-direction probe design

## Open Questions

- Disposition: proceed to a plan; the findings identify a bounded CLI report plus skill protocol without changing gate authority.
- Human decision belongs on the plan: should the first release stop after a fixed pass cap with visible survivors, or require semantic convergence with no cap?
