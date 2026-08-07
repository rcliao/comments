# Plan: strengthen the RPI loop (interview, fresh reviewer, attention budget)

## Overview

Three additions to the RPI skill, all prose in `skills/review-comments/SKILL.md`
— no new commands, no code, no template changes. When done: an agent
running RPI (1) interviews the human before drafting, (2) converges each
artifact against a fresh-context reviewer until the gate is green (or a 2-pass
cap, survivors left open for the human), and (3) keeps
the open-thread set a small attention budget, with its working trace parked in
self-resolved threads.

## Current State

- The loop iterates only after the artifact exists; the first human touch is a
  finished doc (research-rpi-loop-strength.md:14-21).
- Both references loop before prose: HumanLayer forbids one-shot plans
  (research-rpi-loop-strength.md:33) and requires understanding + questions
  first (research-rpi-loop-strength.md:35-38); spec-kit's clarify writes
  answers back into the spec pre-plan (research-rpi-loop-strength.md:71-85).
- Nothing checks plan-against-research faithfulness even though the plan
  template demands citations (research-rpi-loop-strength.md:99-101).
- We lack HumanLayer's verification rules: read-before-delegate and the
  wait-for-question gate (research-rpi-loop-strength.md:66-67) and
  verify-corrections-yourself (research-rpi-loop-strength.md:49-50).
- Our self-review callouts and zone:human are ahead of both references — keep
  (research-rpi-loop-strength.md:121-132).
- Binding constraint: lightweight / on-demand, opinionated not prescriptive
  (research-rpi-loop-strength.md:10-12).

## Desired End State

An RPI run on a real feature produces a research doc and plan doc where the
human's first contact is the agent's understanding (interview), every artifact
survived one fresh-context adversarial pass before human review, and the human
opens the TUI to ≤5 open threads. Verify: one dogfooded RPI cycle on a real
change in this repo, signoff notes confirm the human read everything open.

## What We're NOT Doing

- No new CLI/MCP commands: no `clarify`, no `analyze`. The coverage check rides
  in the reviewer subagent's prompt, not a command. If the prompt version
  proves insufficient after dogfooding, a command is a future plan.
- No spec-kit 9-category taxonomy sweep — a fixed interrogation protocol is
  the "prescribing workflow" the scope steer forbids.
- No changes to `zone: human`, self-review callouts, or the signoff/gate
  machinery — they already work.
- No template changes at all — attention budget and conventions live in
  SKILL.md prose only.

## Implementation Phases

### Phase 1: pre-draft interview + verification rules (SKILL.md)

Add to RPI mode, before drafting each phase doc: present understanding of the
question and the code, plus only the questions that cannot be answered from the
codebase; wait for answers before writing. The interview subsumes F2's
wait-for-question gate: presenting your understanding of the question and
having it confirmed IS the gate. Answers land in the first draft — the
interview precedes drafting, so there is no separate write-back mechanic
(settles research-rpi-loop-strength.md:146-148); questions that surface
mid-draft use the existing NEEDS CLARIFICATION markers. Port the two
HumanLayer verification rules as one short "Verify, don't trust" note: read
cited files before delegating searches, and when the human corrects you,
verify the correction in code before proceeding
(research-rpi-loop-strength.md:49-50,66). Keep it on-demand: a trivial doc may
state "no questions — drafting" and proceed.

**Success Criteria**
- automated: `grep` finds the interview step and both verification rules in
  SKILL.md; `comments validate` on both templates still passes
- manual: on the next RPI request, the agent's first message is understanding +
  questions (or an explicit "no questions"), not a document

### Phase 2: fresh-context reviewer pass, gate-terminated (SKILL.md)

After self-review callouts, before requesting human review: spawn a reviewer
with fresh context — only the doc path, its template criteria, and (for a plan)
the research doc path. Reviewer posts findings as comments (blocking for real
gaps), including the coverage question the author cannot ask itself: which
research findings does the plan silently drop, which claims cite nothing
(research-rpi-loop-strength.md:91-101). Author processes them through the
existing comment loop. Terminate on gate green; cap reviewer passes at 2 and
the attention budget at ~5 open threads — both provisional defaults, replaced
by measured values from the Phase 4 eval, not inherited. State plainly why fresh context: the
author's context finds its own prose convincing
(research-rpi-loop-strength.md:107-111). Environments that cannot spawn
subagents use the stated fallback: a fresh session given only the same
allowlist is an equally valid reviewer.

**Success Criteria**
- automated: SKILL.md describes reviewer input allowlist, blocking policy,
  gate-green termination, and the 2-pass cap
- manual: dogfood run shows reviewer comments materially changed the doc, and
  the loop stopped without human prompting

### Phase 3: attention budget + thought-trace convention (SKILL.md)

From the signoff note: open threads are the human's reading list. Add the
convention — agent working notes and iteration rationale go into threads it
resolves itself immediately (readable via the TUI's resolved toggle `R`,
pkg/tui/overlays.go:40);
threads left open for the human are capped at ~5, priority-ordered; if more
survive, consolidate before requesting review. The budget lives in SKILL.md
prose, not in the templates — template-enforcing it would be the prescription
the scope steer forbids (research-rpi-loop-strength.md:10-12).

**Success Criteria**
- automated: SKILL.md contains the attention-budget and thought-trace
  conventions; `git diff` shows pkg/comment/templates/ untouched
- manual: human opens the dogfood doc to ≤5 open threads and confirms the
  resolved trace was worth reading

### Phase 4: dogfood as eval + settle

Run the upgraded loop on the next 3-5 real feature docs. The sidecar is the
eval log — no new harness: each `.comments.json` already records every
reviewer finding, whether the author accepted (applied + resolved with a fix)
or rebutted it, passes until gate green, open-thread count at review time, and
the signoff decision + note with timestamps. Per run, read off: reviewer
signal ratio (findings that changed the doc / total), passes actually needed,
threads the human faced, and whether the human's note flags fatigue or missed
issues. Set the pass cap and attention budget from those traces; fold misfires
back into SKILL.md; update root CLAUDE.md's RPI paragraph to the final flow.

**Success Criteria**
- automated: `comments gate` green on each dogfood doc; the metrics above
  derivable from the sidecars with jq alone (no new tooling needed)
- manual: after 3-5 runs, the cap and budget re-set from evidence (or the
  provisional values confirmed); CLAUDE.md's RPI paragraph read against
  SKILL.md — no contradiction; your call per piece: keep, trim, revert

## Risks

- **Reviewer theater** — a reviewer that inherits drafting context rubber-stamps.
  Mitigated: SKILL.md states the input allowlist explicitly; anything beyond
  doc + criteria + research path invalidates the pass.
- **Ceremony creep** — the interview annoying on small docs. Mitigated: "no
  questions — drafting" fast path; scope steer cited at the top of the section.
- **Two sources of truth** — SKILL.md and CLAUDE.md drift. Mitigated: Phase 4
  criterion checks them against each other.
