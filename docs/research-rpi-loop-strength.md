# Research: is our RPI loop strong enough to produce a good research/plan artifact?

## Research Question

Our RPI skill asks an agent to produce a research doc and a plan doc, then hands
them to a human. Does that produce quality, or is it a one-shot draft with review
bolted on? What loops do HumanLayer (ACE/RPI) and GitHub spec-kit run to get a
good artifact instead of a plausible one?

Scope steer (rcliao, review 2026-08-07): comments is not becoming a heavy RPI
tool. Adoptions must stay lightweight / on-demand — opinionated about what makes
a quality artifact, never prescriptive about the workflow around it.

## Summary

The suspicion is correct, and the gap is specific: **our loop iterates only after
the artifact exists.** Both references run a loop *before* any prose is written —
HumanLayer with an interactive read → question → align → structure → draft
sequence that explicitly forbids one-shotting, spec-kit with a structured
clarification sweep whose answers are written back into the spec before planning
starts. Our first human touch is a finished document. Theirs is the agent's
understanding, when redirecting it is still cheap. A third source names the exact
failure mode we are worried about ("the plan-reading illusion"). Separately, both
references have a cross-artifact consistency check we have no equivalent of.

## Findings

### F1 — HumanLayer's plan prompt forbids one-shot drafting outright

`create_plan.md` prescribes a 4-5 round interactive process before a finished
plan exists:

> "Don't write the full plan in one shot"

Round shape: context gathering (spawn research) → **"Present informed
understanding and focused questions"** → research findings + design options →
**"Get feedback on structure before writing details"** → detailed writing, then
"Iterate based on feedback".

Two rules constrain the questions so the loop doesn't become an interview:

> "Only ask questions that you genuinely cannot answer through code investigation."

> "If you encounter open questions during planning, STOP" / "No Open Questions in
> Final Plan: Every decision must be made before finalizing"

And an anti-sycophancy rule with no analogue in our skill:

> "DO NOT just accept the correction... Only proceed once you've verified the
> facts yourself"

Source: https://raw.githubusercontent.com/humanlayer/humanlayer/main/.claude/commands/create_plan.md

### F2 — HumanLayer's research prompt is procedural about evidence, not just tone

- Read first, delegate second — marked CRITICAL: *"Use the Read tool WITHOUT
  limit/offset parameters to read entire files"*, *"Read these files yourself in
  the main context before spawning any sub-tasks."*
- Parallel subagents for search, synthesis only after all return.
- Documentarian rule, stated three times: *"DO NOT suggest improvements or
  changes unless the user explicitly asks"*, *"ONLY describe what exists"*,
  *"Document what IS, not what SHOULD BE."*
- Waits for the human's research question before starting.

We already encode the documentarian rule (`research.yaml` review_criteria) and
the file:line evidence requirement. We do **not** encode read-before-delegate or
the wait-for-question gate.

Source: https://raw.githubusercontent.com/humanlayer/humanlayer/main/.claude/commands/research_codebase.md

### F3 — spec-kit `/clarify` replaces "flag what you'd guess at" with a coverage scan

Our markers are emitted at the agent's discretion: the skill says use `[NEEDS CLARIFICATION]`
"where you would otherwise guess", capped at 3 (research) / 1 (plan). spec-kit
makes the same decision *systematic*:

- Scan a **9-category taxonomy** (functional scope, domain/data model,
  interaction/UX, non-functional, integration, edge cases, constraints/tradeoffs,
  terminology, completion signals), marking each **Clear / Partial / Missing**.
- Prioritize by **(Impact × Uncertainty)**, take the top 5. Hard max: 5 questions.
- **One question at a time**, each with a "Why it matters" line, offered as 2-5
  multiple-choice options with a recommended answer, or a ≤5-word short answer.
- After **each** accepted answer: append to a `## Clarifications` log AND
  immediately apply it to the relevant section, saving after every write.
- Unresolved categories past the quota are explicitly marked "Deferred".

Optional command, but positioned between `/specify` and `/plan`.

Source: https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/clarify.md

### F4 — spec-kit `/analyze` cross-checks the artifacts against each other

Read-only pass over spec ↔ plan ↔ tasks ↔ constitution, six detections
(duplication, ambiguity, underspecification, constitution alignment, **coverage
gaps**, inconsistency), four severities. Coverage gaps = "requirements with zero
tasks; tasks with no mapped requirement". CRITICAL findings are expected to be
resolved before implement. Explicitly: *"Do not modify any files."*

We have `validate` (structure of ONE doc) and `gate` (comment state). Nothing
checks that the plan is faithful to the research — even though our plan template
requires `research-foo.md:23` citations and the TUI can already peek them.

Source: https://raw.githubusercontent.com/github/spec-kit/main/templates/commands/analyze.md

### F5 — the failure mode has a name, from a practitioner's RPI critique

"From RPI to QRSPI" reports RPI producing "the plan-reading illusion":

> "Plans that read well don't necessarily build well. Architectural debt
> accumulated underneath because the plan's prose was convincing while its
> technical assumptions were wrong."

Their fix is a Design Discussion phase between research and plan — force the
agent to state its architectural understanding as an artifact the human can
redirect, *before* planning turns it into prose. Same diagnosis as F1's "don't
write the plan in one shot" — corroboration from practice, not an independent
replication: the post cites Horthy and is downstream of the HumanLayer school.

Source: https://alexlavaee.me/blog/from-rpi-to-qrspi/

### F6 — what our loop already does better

Not all gap. Our `## Drafting mode` step 3 requires the agent to self-review and
post **specific** callouts anchored at exact lines — weakest reasoning, unverified
assumptions, from-memory facts — with "a criterion your draft clearly satisfies
needs no comment". spec-kit's `/checklist` is the nearest analogue — generated
"unit tests for requirements writing" that probe completeness/clarity/coverage of
the artifact — but its probes are systematic questions about the doc, not the
author confessing its own least confident sentence at the exact line. That
confession move appears in neither reference. Our
`zone: human` rule (agents cannot resolve threads in human-decision sections) is
also stronger than anything in either reference.

## Code References

- `skills/review-comments/SKILL.md` — `## RPI mode`, `## Drafting mode` (the loop under review)
- `pkg/comment/templates/research.yaml` — documentarian criteria, marker cap 3
- `pkg/comment/templates/plan.yaml` — citation criteria, marker cap 1, automated/manual criteria split
- `pkg/tui/keys_refpeek.go`, `buildRefMap` in `pkg/tui/` — existing citation resolution, the machinery an `analyze`-style check would reuse

## Open Questions

- ~~"riptides"~~ resolved in review: riptides.io is runtime machine-IAM for
  agents (credential/egress control), not a workflow tool; no riptide-named RPI
  tool exists on the web or GitHub. Reference set stays HumanLayer + spec-kit.
- Does the interactive pre-draft round belong in the skill (prompt-only) or does
  it need tool support (a `clarify` command that writes answers back into the doc,
  the way spec-kit does)?
