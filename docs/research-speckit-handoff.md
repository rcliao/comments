# Research: Handing a reviewed plan to Spec Kit

## Research Question

Q1. What does Spec Kit require as input before it will generate tasks?
Q2. Which of our reviewed artifacts map onto those inputs as they stand?
Q3. What is currently absent that prevents a direct handoff?

## Summary

Spec Kit consumes a feature directory, not loose documents. Task generation reads two required files — `plan.md` for technical context and `spec.md` for user stories with priorities — and phases the generated work by those priorities.

Our research doc lands in that structure unchanged: Spec Kit already looks for a `research.md` and detected ours immediately. Our plan doc does not map as cleanly, and we have nothing at all shaped like `spec.md`.

The blocker is therefore not format but content. Spec Kit derives task order from prioritised user stories, and no artifact in our flow records one. Everything else is a directory layout we can satisfy by copying files.

## Findings

### F1 — Task generation requires two files, and reads them for different things [Q1]

`/speckit-tasks` states its inputs plainly:

- **Required**: `plan.md` — "tech stack, libraries, structure"
- **Required**: `spec.md` — "user stories with their priorities (P1, P2, P3, etc.)"

It then builds "one phase per user story (in priority order from spec.md)". Priority ordering is the organising principle of the output, and it comes from `spec.md` alone ([github/spec-kit](https://github.com/github/spec-kit), commit `684b3d8`).

### F2 — The gate before implementation is file existence, checked with an exit code [Q1]

`check-prerequisites.sh --require-tasks` reports `ERROR: tasks.md not found` and exits 1 when the file is absent. Run without that flag it prints paths as JSON and exits 0.

`AVAILABLE_DOCS` is a separate list, populated only from supplementary files: `research.md`, `data-model.md`, `contracts/`, `quickstart.md`. Absence of `plan.md` from that list is not a rejection — it is tracked as its own required path.

### F3 — Our research doc drops in unchanged [Q2]

Copied `docs/research-agent-surface.md` into a feature directory as `research.md`. The prerequisite script picked it up on the next run, reporting `AVAILABLE_DOCS: ["research.md"]`.

No adaptation was needed. Spec Kit already reserves that filename for exactly this purpose, so a reviewed research doc is consumable as supplementary context today.

### F4 — Our plan template answers a different question than theirs [Q2]

Our plan sections are decision-shaped (`pkg/comment/templates/plan.yaml:11-44`):

- Overview, Current State, Desired End State
- What We're NOT Doing, Implementation Phases, Risks

Spec Kit's `plan-template.md` is context-shaped: Summary, Technical Context, Constitution Check, Project Structure, Complexity Tracking. The overlap is partial, and the field `/speckit-tasks` actually reads — tech stack and libraries — has no home in our template.

### F5 — Nothing in our flow records prioritised user stories [Q3]

Spec Kit's `spec-template.md` is built around `## User Scenarios & Testing (mandatory)`, with `### User Story N (Priority: PN)`, plus Functional Requirements and Measurable Outcomes.

We have no equivalent artifact and no template that produces one. A handoff therefore requires authoring `spec.md` separately, or running `/speckit-specify` first — at which point Spec Kit, not our review loop, owns the input that orders the work.

### F6 — The two flows produce the same artifact twice [Q3]

Our Implementation Phases and their `tasks.md` both decompose work into ordered units. Ours is human-reviewed prose with success criteria split automated versus manual; theirs is generated from user stories and dependency-ordered.

Carrying our plan across and then running `/speckit-tasks` re-derives phases a human already reviewed.

## Code References

- `pkg/comment/templates/plan.yaml:11-44` — our plan's section shape
- Spec Kit `.specify/scripts/bash/check-prerequisites.sh` — existence gate, exit 1
- Spec Kit `.specify/templates/spec-template.md` — user stories and priorities

## Open Questions

[NEEDS CLARIFICATION: Is adopting Spec Kit's task phase compatible with the scope fence, given tasks are meant to be agent-managed or become tickets?]

Would a `spec` template that emits their `spec.md` shape be worth building, or does authoring one by hand at handoff time cost less than maintaining a template?
