# Research: comments as a closed-loop feature-delivery protocol

## Research Question

Q1. Which controls let an agent expand research coverage without making its own judgment authoritative?

Q2. Which document and comment mechanics reduce the human cost of approving a large implementation plan?

Q3. What durable state must survive from approved plan through implementation so an agent can resume without drifting?

Q4. Where should comments stop so it remains a lightweight collaboration protocol rather than an agent orchestrator?

Q5. How do current harnesses divide work and context across agents?

Q6. Which coordination state belongs to runtimes during implementation?

Q7. What reviewed context survives agent handoffs, and what role remains for comments?

## Summary

comments already has the right research primitive: independent agents express semantic challenges as evidence-backed threads, while deterministic analysis checks declared coverage and citations. [Q1]
Its templates, compact thread walkthrough, change marks, and machine-readable gate provide a stronger plan-review contract than artifact generators alone. [Q2]

After approval, comments preserves no implementation context for a later session. [Q3]
Current harnesses converge on isolated workers plus root or lead synthesis; shared write-heavy state remains a coordination hazard. [Q5]
They keep dependencies, agent identity, messaging, retries, and resume in runtime state, not reviewed documents. [Q6]

Shared artifacts preserve intent, feedback, findings, validation, and later-session context.
No surveyed system uses a commented plan as the executor. [Q7]
The unfilled role for comments is narrower: a living decision record for material implementation findings while the harness owns execution. [Q4] [Q7]

## Findings

### F1 — research quality needs independent challenges, not a semantic gate [Q1]

`comments analyze` can prove that declared questions have findings, citations resolve, and a plan covers every research finding (pkg/comment/analyze.go:60-118,198-243).
It cannot prove that the initial question set is complete or that cited text supports a claim.

The repository's convergence protocol handles that boundary correctly.
A draft-blind coverage scout proposes missing questions, while an evidence verifier challenges claims using ordinary threads (skills/review-comments/SKILL.md:304-340).
Accepted gaps expand the numbered question set; rejected gaps remain resolved rationale.
Model judgment stays contestable, while the thread record compounds across rounds.

HumanLayer now separates research questions from research for the same contamination concern.
Its current workflow also separates objective research from later design decisions ([HumanLayer workflow reference](https://docs.humanlayer.com/reference/skills-workflows)).

### F2 — templates constrain production; comments should constrain attention [Q2]

Spec Kit's current pipeline uses Markdown artifacts, templates, checklists, and cross-artifact analysis to carry intent from specification through implementation ([GitHub Spec Kit](https://github.github.com/spec-kit/)).
Those mechanisms improve what an agent produces, but they do not create a precise conversation inside the artifact.

comments adds the missing attention layer.
Templates cap document and subsection size, require evidence, reserve human-owned sections, and force explicit alternatives (pkg/comment/templates/plan.yaml:1-51).
Priority threads become the human walkthrough, while the gate reduces recorded blockers to a stable exit code (pkg/comment/gate.go:21-70).

This is the product distinction: templates budget prose; comments budget reviewer attention.

### F3 — the market has converged on annotation, but not durable authority [Q4]

Plannotator, r3, redline, MarkupMarkdown, and Markdown Reader now offer browser or editor annotation for agent plans and Markdown.
Several support threaded replies, revisions, MCP, or sending feedback back to an agent ([Plannotator](https://github.com/backnotprop/plannotator), [r3](https://github.com/hyperlogue/r3), [redline](https://github.com/alevi/redline), [MarkupMarkdown](https://github.com/jonradoff/markupmarkdown)).

The opportunity is no longer "Google Docs comments for Markdown."
It is a local, agent-readable decision protocol: human-owned zones, durable resolved rationale, deterministic gates, review waiting, and cross-document evidence.

Spec Kit has expanded in the other direction with a workflow DSL for commands, shell steps, gates, branches, loops, and fan-out ([Spec Kit workflows](https://github.com/github/spec-kit/blob/main/docs/reference/workflows.md)).
That orchestration surface defines the boundary comments should integrate with rather than duplicate.

Competing on editor richness would erase the lightweight advantage.
The browser surface should remain a review client over the same sidecar semantics, not the product's source of truth.

### F4 — plan approval currently authorizes nothing durable [Q3]

`ReviewRecord` contains author, timestamp, decision, and note (pkg/comment/types.go:139-145).
It does not name a phase or store the reviewed document hash.
`EvaluateGate` reports the latest review but does not require one, so a green document and an approved document are mechanically equivalent (pkg/comment/gate.go:25-70).

That ambiguity becomes dangerous after a plan changes.
The next agent cannot tell whether approval covers the current content, the whole plan, or one implementation phase.
Repository evidence confirms recent plans were implemented without any review record (docs/research-long-horizon-alignment.md:35-88).

Approval must become a content-bound capability that covers a named scope, while remaining a local record rather than identity infrastructure.

### F5 — cold-start context is the largest missing agent surface [Q3]

`comments inbox` reports blocking threads and new replies (pkg/comment/inbox.go:23-71).
It cannot report which research produced a plan, which phase is active, what was verified, or what comes next.
The sidecar stores no artifact relationship beyond its template name (pkg/comment/storage.go:16-24).

HumanLayer's current task model keeps separate research, design, implementation, and review sessions under one durable task.
Its implementation commands record completed phases and validation items in the outline or plan ([HumanLayer task model](https://docs.humanlayer.com/explanation/tasks), [HumanLayer workflow reference](https://docs.humanlayer.com/reference/skills-workflows)).

comments does not need a task service to gain that benefit.
A persisted document graph and phase checkpoint records are sufficient for a `resume` view to reconstruct the working set.

### F6 — autonomous implementation needs drift detection and evidence checkpoints [Q3]

The plan template requires automated and manual success criteria for every phase (pkg/comment/templates/plan.yaml:35-48).
After plan review, no command records whether those criteria ran or links them to a commit.
`watch` observes sidecar changes only and cannot wake on a changed plan (pkg/comment/watch.go:49-93).

An agent can therefore continue from stale authority, skip a phase check, or report completion without a durable evidence trail.
The lightweight remedy is not an execution engine.
It is a checkpoint record containing phase, approved plan hash, outcome, evidence references, and implementation commit.

A changed plan invalidates later checkpoints.
A failed or manual criterion creates a precise thread instead of a free-form status paragraph.

### F7 — the human gate should stay at plan shape, with exceptions for drift [Q2] [Q3]

HumanLayer's current guided flow uses design discussion and a structure outline before implementation.
The outline groups work into vertical slices with validation checks, and detailed plans are now an optional older path ([HumanLayer workflow phases](https://docs.humanlayer.com/explanation/workflow-phases)).

comments previously found that a separate research signoff added ceremony without leverage (skills/review-comments/SKILL.md:282-303).
The compatible lesson is artifact granularity, not more mandatory meetings.

Keep one default human sitting on the plan.
Pause early only for scope-changing research questions, and pause during implementation only when the approved plan changes, a manual check is required, or evidence fails.

### F8 — success must be measured at delivery, not document conformance [Q1] [Q3]

Current evaluation measures research coverage, unsupported claims, handoff coverage, review burden, and convergence passes (docs/plan-autonomous-research-convergence.md:60-79).
Those are useful leading indicators, but the product promise is accurate implementation.

The missing evaluation follows a fixed feature from question through code.
It scores requirement coverage, deviations from approved decisions, success-criterion completion, human interventions, rework after review, and cold-session recovery.

Without that trace, a cleaner plan can be mistaken for a better delivered feature.
The current product therefore cannot claim that document improvements increase delivery autonomy.

### F9 — parallel agents isolate context and centralize synthesis [Q5]

OpenAI Multi-agent gives bounded workers separate context and makes the root responsible for synthesis.
Its guidance recommends parallel agents for independent work and warns against shared mutable resources ([OpenAI Multi-agent](https://developers.openai.com/api/docs/guides/responses-multi-agent)).
Codex similarly returns distilled worker summaries to the main thread and cautions against parallel write-heavy workflows ([OpenAI Codex subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents)).

GitHub Copilot Fleet takes an implementation plan, decomposes independent subtasks, and leaves dependency management to the main agent ([GitHub Copilot Fleet](https://docs.github.com/en/copilot/concepts/agents/copilot-cli/fleet)).
Claude Code distinguishes result-only subagents from teams that need direct discussion and a shared task list ([Claude Code agent teams](https://code.claude.com/docs/en/agent-teams)).

Across these systems, the worker contract is a bounded assignment plus a compact result.
The root or lead remains responsible for reconciliation, conflicts, and the final account.

### F10 — task state and durable execution belong to the harness [Q6]

Claude Code stores teammate identity, mailboxes, task dependencies, and task status as generated runtime state.
Its documentation explicitly warns that team configuration is overwritten and should not be hand-authored ([Claude Code agent teams](https://code.claude.com/docs/en/agent-teams)).

Spec Kit persists resumable workflow runs and coordinates gates, loops, and fan-out/fan-in ([Spec Kit workflows](https://github.com/github/spec-kit/blob/main/docs/reference/workflows.md)).
LangGraph separates thread-scoped checkpoints from cross-thread application data, with the agent server owning persistence infrastructure ([LangGraph persistence](https://docs.langchain.com/oss/python/langgraph/persistence)).
Codex long-running work similarly keeps each goal and parallel chat in its own context and recommends isolated worktrees for concurrent writes ([OpenAI Codex long-running work](https://learn.chatgpt.com/docs/long-running-work)).

These are rapidly changing operational facts: who is running, what is blocked, where work resumes, and which checkout may be written.
Duplicating them in document comments creates two sources of truth.

### F11 — reviewed artifacts carry decisions and discoveries, not scheduling [Q7]

HumanLayer groups separate sessions with shared task files, comments, and history.
Reviewers comment on synced documents, and later agents can read that feedback ([HumanLayer task model](https://docs.humanlayer.com/explanation/tasks)).
Its implementation workflow records completed phases and validation in the approved outline or plan ([HumanLayer workflow reference](https://docs.humanlayer.com/reference/skills-workflows)).

GitHub's asynchronous coding agents use issues or prompts as inputs, isolated sessions for work, and pull-request comments for human iteration ([GitHub coding agents](https://docs.github.com/en/copilot/concepts/agents/about-third-party-coding-agents)).
No surveyed system uses annotation as the live task scheduler.

comments already preserves anchored discussion and resolved rationale, but only within one document review (pkg/comment/storage.go:16-24, pkg/comment/inbox.go:23-71).
The unfilled role is a concise, reviewable projection of approved decisions, material findings, deviations, evidence, and human asks across agent sessions.

## Code References

- pkg/comment/analyze.go:60-118,198-243 — deterministic research and handoff checks
- pkg/comment/templates/plan.yaml:1-51 — reviewability and phase-success constraints
- pkg/comment/types.go:139-145 — unscoped review record
- pkg/comment/inbox.go:23-71 — warm attention view, not cold-start state
- pkg/comment/watch.go:49-93 — review-state snapshot omits document changes
- skills/review-comments/SKILL.md:282-340 — autonomous chain and semantic convergence
- pkg/comment/storage.go:16-24, pkg/comment/inbox.go:23-71 — document-local review state without implementation context

## Open Questions

- Proceed: revise the plan now around a living decision record, without adding implementation runtime state.
- Sequence: finish or park current uncommitted work before changing the sidecar schema; overlapping edits make parallel implementation unsafe.
- Product fence: keep status, ownership, dependencies, messaging, retries, worktrees, and resume in the harness.
