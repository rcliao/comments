---
comments:
    template: research-deep
description: Adjacent-product research for keeping Comments focused on reviewed implementation intent across long-running agent execution.
generated:
    by: agent:codex
    at: 2026-08-26T13:00:00-07:00
related:
    - path: okf-comments-workflow.md
      relation: informed_by
sources:
    - id: factory-missions
      resource: https://docs.factory.ai/missions/overview
      title: Factory Missions overview
    - id: factory-planning
      resource: https://docs.factory.ai/missions/planning
      title: Factory Missions planning and validation
    - id: linear-agents
      resource: https://linear.app/docs/agents-in-linear
      title: AI agents in Linear
    - id: linear-sessions
      resource: https://linear.app/developers/agent-interaction
      title: Linear agent sessions and activities
    - id: github-planning
      resource: https://docs.github.com/en/copilot/tutorials/plan-a-project
      title: Planning a project with GitHub Copilot
    - id: github-management
      resource: https://docs.github.com/en/copilot/concepts/agents/cloud-agent/agent-management
      title: GitHub agent management
    - id: openspec
      resource: https://github.com/Fission-AI/OpenSpec/blob/main/docs/overview.md
      title: OpenSpec overview
    - id: plannotator
      resource: https://github.com/backnotprop/plannotator
      title: Plannotator
    - id: cursor-background
      resource: https://docs.cursor.com/background-agent
      title: Cursor background agents
status: draft
tags: [agents, alignment, implementation, planning, status]
title: Week-Long Human-Agent Implementation Alignment
type: Research
---

# Week-Long Human-Agent Implementation Alignment

## Research Question

Q1. How do adjacent products divide reviewed intent, work tracking, and live agent execution?

Q2. Which mechanics keep a week-long implementation aligned without requiring constant human supervision?

Q3. What durable state can Comments own after adopting OKF without becoming a project manager or agent runtime?

Q4. What is the smallest integration seam that can create an always-on human experience?

## Summary

The market has separated into three layers: agreement artifacts, work graphs, and execution runtimes. [Q1]
OpenSpec and Plannotator improve the agreement boundary.
Linear and GitHub own delegation and shared work state.
Factory, Cursor, and coding agents own sessions, workers, validation, and resume.

Comments is strongest as the reviewed agreement layer. [Q2] [Q3]
OKF now supplies portable lineage and bounded context, while threads supply decisions and signoff.
Its missing contribution is a current, evidence-backed projection of implementation against the approved plan.

An always-on experience does not require Comments to execute work. [Q4]
It requires runtime events to promote milestone outcomes, deviations, evidence, and human asks into the reviewed artifact.
Routine logs, ownership, retries, and worker health remain outside Comments.

## Findings

### F1 — adjacent products occupy three distinct state layers [Q1]

Agreement tools preserve what humans and agents decided.
OpenSpec keeps proposed changes beside current specifications, then merges accepted deltas into project truth after implementation.
Plannotator preserves plan versions, annotations, and approval history.
Neither surface is the live worker scheduler. ([OpenSpec](https://github.com/Fission-AI/OpenSpec/blob/main/docs/overview.md), [Plannotator](https://github.com/backnotprop/plannotator))

Work systems preserve ownership, hierarchy, and team-visible progress.
GitHub Copilot can turn a project description into an epic and issue tree.
Linear keeps the human as issue owner when work is delegated to an agent. ([GitHub planning](https://docs.github.com/en/copilot/tutorials/plan-a-project), [Linear agents](https://linear.app/docs/agents-in-linear))

Execution systems preserve sessions, workers, validation, and runtime health.
Factory Missions and Cursor Background Agents expose ongoing remote work, steering, and resume. ([Factory Missions](https://docs.factory.ai/missions/overview), [Cursor Background Agents](https://docs.cursor.com/background-agent))

### F2 — Comments already owns a differentiated agreement record [Q1] [Q3]

The OKF bundle makes artifact placement and lineage deterministic. pkg/comment/bundle.go:137

Its collections cover research, plans, decisions, and as-built records.
Role-scoped context returns every related concept with an explicit inclusion reason.
It does not delegate discovery to an opaque relevance score. pkg/comment/context.go:60

The portable Markdown stores intent and evidence.
Frontmatter stores lifecycle, provenance, and relationships.
The sidecar stores anchored discussion and verdicts.
That separation is narrower than a project system and richer than a transient plan-mode response. docs/OKF.md:19

The review surface budgets human attention through section caps and human-owned zones. pkg/comment/templates/plan.yaml:10

Priority threads and a deterministic gate narrow the remaining decisions.
That is the product center to preserve.

### F3 — Plannotator is closest at review time, not execution time [Q1] [Q3]

Plannotator visually reviews plans and code, returns annotations to an agent, and keeps plan-version history.
Its archive preserves approved and denied decisions.
Those capabilities make it the closest direct review competitor. ([Plannotator architecture](https://github.com/backnotprop/plannotator/blob/main/AGENTS.md))

Comments carries a different durable contract.
Threads remain anchored beside repository Markdown, citations resolve back to evidence, and a machine-readable gate survives the originating agent session.
The new OKF bundle also relates that plan to other typed knowledge.

Plannotator's growing code-review and workspace surface shows one expansion path.
Comments has chosen the other: repository-native knowledge plus review authority.
Neither product currently describes the annotated plan as the executor's live task database.

### F4 — OpenSpec is adjacent at the artifact lifecycle boundary [Q1] [Q3]

OpenSpec frames itself as an agreement layer.
Its default loop creates proposal, specification, design, and task artifacts; `/opsx:apply` checks off tasks; archive merges the delta into current truth. ([OpenSpec overview](https://github.com/Fission-AI/OpenSpec/blob/main/docs/overview.md))

This overlaps Comments in durable planning and artifact lineage.
It differs at review granularity.
OpenSpec's documented loop asks the human to read and adjust artifacts, while Comments stores anchored debate, resolved rationale, suggestions, and verdicts.

The comparison exposes a lifecycle gap in Comments.
The bundle contains an `as-built` collection. .comments/bundle.yaml:20

Current review state does not connect an approved plan to implementation outcomes.
The gap is durable closure, not another specification generator.

### F5 — work graphs make agents visible while humans retain responsibility [Q1] [Q2]

Linear models delegation separately from assignment.
The human assignee remains responsible while the agent acts on the issue.
Agent sessions expose working, waiting, error, and completed lifecycle states. ([Linear delegation](https://linear.app/docs/agents-in-linear), [Linear sessions](https://linear.app/developers/agent-interaction))

GitHub similarly centralizes agent sessions, live logs, steering, pull requests, and scheduled or event-triggered automations.
Its planning surface produces issue hierarchies before sessions begin. ([GitHub agent management](https://docs.github.com/en/copilot/concepts/agents/cloud-agent/agent-management))

These systems solve portfolio visibility and routing.
Their unit is an issue or session, not a line inside a reviewed implementation argument.
Comments can complement that work graph, but duplicating assignment, status transitions, or notifications would introduce two authorities.

### F6 — week-long execution requires a runtime, checkpoints, and steering [Q2]

Factory Missions most closely matches the ideal experience.
Human and agent refine features, milestones, skills, and success criteria before execution.
Mission Control then coordinates workers, tracks progress, validates milestones, and supports pause, redirect, and resume. ([Factory Missions](https://docs.factory.ai/missions/overview))

Factory also states that long-running plans accumulate errors.
Its mitigation is validation at milestone boundaries, with more frequent milestones for longer work. ([Factory validation](https://docs.factory.ai/missions/planning))

Cursor and GitHub expose the same operational category through background sessions, follow-ups, live status, and pull-request handoff.
This state changes too quickly for a reviewed Markdown sidecar.
The execution harness remains the authoritative source for workers, queues, retries, and health.

### F7 — Comments cannot currently show whether execution still matches approval [Q2] [Q3]

The sidecar stores document hash, threads, reviews, and template only.
It has no phase, active implementation, checkpoint, or external session reference. pkg/comment/storage.go:16

Each review records author, decision, note, and time without the reviewed content hash or scope. pkg/comment/types.go:139

`watch` observes sidecar changes only.
Its snapshot omits the current Markdown hash. pkg/comment/watch.go:49

Plan edits therefore do not produce an event.
`context` exposes review counts and relationships, not implementation progress. pkg/comment/context.go:49

Therefore a fresh agent can reconstruct approved reasoning but not execution position.
A human can see unresolved decisions but not whether the latest milestone passed.
The OKF adoption improves navigation; it does not close this temporal gap.

### F8 — the smallest seam is a reviewed projection of runtime events [Q2] [Q4]

Linear's agent API separates immutable session activities from editable comments.
Its guidance warns that comments may change and recommends activities as runtime snapshots. ([Linear best practices](https://linear.app/developers/agent-best-practices))
Comments can apply the inverse boundary: runtime events stay authoritative, while selected outcomes become durable review knowledge.

Only four event classes have lasting planning value: milestone evidence, discovered deviation, blocking human decision, and completion summary.
Routine progress, worker identity, retries, token usage, and logs remain runtime facts.

That projection can use existing plan threads and the existing as-built artifact type.
The plan changes only when approved intent changes.
The as-built record captures what shipped and validation evidence.
This preserves one execution authority while giving later humans and agents a concise, reviewed memory.

### F9 — “always on” is a notification promise, not document autonomy [Q4]

Background products feel continuous because they run remotely, retain sessions, and notify humans when input or review is needed.
Cursor exposes status and follow-up messages.
GitHub exposes active sessions, steering, automations, and review handoff. ([Cursor](https://docs.cursor.com/background-agent), [GitHub](https://docs.github.com/en/copilot/concepts/agents/cloud-agent/agent-management))

Comments already offers a local event bus for review changes. pkg/comment/watch.go:9

It covers comments, replies, verdicts, and gate transitions.
It does not host remote work or deliver cross-device notifications.

The adjacent evidence supports an integration posture.
An external harness or work system remains always on.
Comments supplies the approved context at launch and receives durable milestone outcomes.
The human returns only for a blocking deviation, failed manual criterion, or final review.

## Code References

- `pkg/comment/bundle.go:137` — default OKF taxonomy and artifact placement.
- `pkg/comment/context.go:49` — role-scoped bundle neighborhood without execution state.
- `pkg/comment/storage.go:16` — document-local collaboration schema.
- `pkg/comment/types.go:139` — unscoped review record.
- `pkg/comment/watch.go:49` — review-only event snapshot.
- `pkg/comment/templates/plan.yaml:10` — reviewable plan contract and milestone success criteria.
- `.comments/bundle.yaml:20` — existing as-built collection for durable closure.

## Open Questions

- Proceed: turn this research into a narrow plan for runtime-to-artifact milestone projection, or keep integration as a documented convention only?
- Product fence: confirm that Comments never owns assignment, worker health, retries, scheduling, or remote execution.
- First adapter: dogfood against Codex tasks, GitHub agent sessions, or a generic NDJSON event input?
- Human surface: prefer status as anchored milestone threads, a generated read-only digest, or both?
