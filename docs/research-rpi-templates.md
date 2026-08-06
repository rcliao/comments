# Research: RPI Phase Templates for comments

- date: 2026-08-06
- researcher: claude
- topic: what research/plan phase docs should contain, and what our template engine already supports
- status: complete

## Research Question

The RPI flow (Research → Plan → Implement, HumanLayer's ACE-FCA) reviews a plan doc that cites a research doc. `comments` has doc templates and, as of today, citation peek (`f`). What template shapes would make `comments` natively support RPI phase docs — without importing the verbosity the ecosystem is criticized for?

## Summary

HumanLayer's plan template is the strongest prior art: ~200-line plans, per-phase success criteria split automated/manual, an explicit scope fence, and a "No Open Questions" rule at plan time. Their research docs carry frontmatter plus fixed sections ending in Code References and Open Questions. Our template engine already supports every constraint these shapes need (required sections, word caps, forced subsections, human zones, review criteria, marker caps) — the gap is purely two missing built-in templates and documented flow guidance.

## Findings

### F1 — HumanLayer plan shape (the review unit)

Their `create_plan.md` command template produces plans with this structure: Overview, Current State Analysis, Desired End State (spec plus how to verify), Key Discoveries as file:line references, "What We're NOT Doing", Implementation Approach, then numbered Phases — each phase carrying Success Criteria split into **Automated Verification** (runnable commands as checkboxes) and **Manual Verification** (human-executed checks).

### F2 — Plan discipline rules

Three hard rules travel with that template: plans target ~200 lines reviewable in 10-15 minutes ("I can't read 2000 lines of golang daily, but I can read 200 lines of a well-written implementation plan"); prohibited content is actual code, exploration logs, and rejected alternatives; and "No Open Questions — every decision must be made before finalizing the plan." Open questions belong to the research phase, not the plan.

### F3 — HumanLayer research shape

Research docs live in a shared directory with timestamped names and YAML frontmatter (date, researcher, git_commit, topic, status). Fixed sections: Research Question, Summary, Detailed Findings, Code References, Open Questions. They are "documentarian" artifacts — current state described without critique, findings carrying file:line evidence.

### F4 — What the ecosystem gets wrong

Spec Kit enforces no length caps anywhere and drew measured criticism: 2,577 lines of markdown to review 689 lines of code (~3.5h of human review), "repetitive... I'd rather review code than all these markdown files." Its later remedy was capping [NEEDS CLARIFICATION] markers at 3 per spec. Kiro's phase gates (requirements → design → tasks approval checkpoints) are the approval pattern; its miss is the same — no brevity control, a small bug becoming "4 user stories with 16 acceptance criteria."

### F5 — What our engine already supports

Everything the shapes above need exists in pkg/comment/template.go: doc-level word caps (pkg/comment/template.go:65), per-section caps (pkg/comment/template.go:71), forced subsections (pkg/comment/template.go:72), human-owned zones (pkg/comment/template.go:73), per-section review criteria seeded as threads (pkg/comment/template.go:74), and marker caps (pkg/comment/template.go:84). The design-doc built-in (pkg/comment/templates/design-doc.yaml) is 53 lines and exercises all of them.

### F6 — The citation side is already built

Plan-cites-research review is what reference peek shipped for: ParseReferences (pkg/markdown/refs.go:36) detects `path.ext:line` tokens and md links, ResolveReference (pkg/markdown/refs.go:118) resolves doc-relative then walking up, and `f` in the TUI peeks the cited line. A plan and its research doc sitting in the same directory need no configuration at all.

### F7 — Phase linkage is convention, not mechanism

Kiro gates phases inside its IDE; HumanLayer gates them by human review between commands. Neither uses a machine-readable per-phase contract. Our `gate`/`signoff` already provides one per document — so RPI phase progression (research signed off before plan drafted, plan gate green before implementing) can be pure convention over existing commands, needing no new engine features.

## Code References

- pkg/comment/template.go:65 — doc max_words
- pkg/comment/template.go:71-74 — section caps, min_subsections, zone, review_criteria
- pkg/comment/template.go:84 — markers.max
- pkg/comment/templates/design-doc.yaml — the pattern to follow (53 lines)
- pkg/markdown/refs.go:36 — ParseReferences
- pkg/markdown/refs.go:118 — ResolveReference
- pkg/tui/keys_refpeek.go — peek + $EDITOR handoff

## Open Questions

- Should the plan template hard-forbid an "Options Considered" section (F2: rejected alternatives are prohibited in plans — they belong in the design doc), or merely omit it?
- Do we want a `docs/research/` + `docs/plans/` directory convention, or leave placement free?
