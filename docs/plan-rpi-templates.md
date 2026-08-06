# Plan: RPI Phase Templates (research + plan built-ins)

## Overview

Add two built-in templates — `research` and `plan` — plus flow documentation, so `comments` natively supports the RPI loop: agent researches → human signs off research → agent plans, citing the research → human reviews the plan with citation peek → gate green → implement. Grounded in research-rpi-templates.md (reviewed separately).

## Current State

- The template engine already supports every needed constraint — word caps, forced subsections, human zones, review criteria, marker caps (research-rpi-templates.md:34) — and design-doc/adr/rfc/mini are the only built-ins.
- Citation review shipped today: `f` peeks `path.ext:line` references, resolution is automatic for same-directory docs (research-rpi-templates.md:38).
- Phase progression needs no new mechanism — `gate`/`signoff` per doc already provides the contract (research-rpi-templates.md:42).

## Desired End State

`comments template list` shows `research` and `plan`. An agent can be pointed at either as a writing brief; `validate` enforces the phase shape; a plan doc citing its research doc reviews end-to-end in the TUI with peek. Verify: the Success Criteria below, plus one real RPI pass on a future feature.

## What We're NOT Doing

- No new engine features, commands, or MCP tools — templates + docs only (research-rpi-templates.md:42).
- No directory convention enforcement — placement stays free; same-directory is recommended, not required (open question in research-rpi-templates.md:56).
- No Options Considered in plans — alternatives live in design docs; plans record decisions already made (research-rpi-templates.md:22).

## Phase 1: `research` built-in template

Mirror the HumanLayer research shape (research-rpi-templates.md:26) with our guardrails, following the design-doc YAML pattern (pkg/comment/templates/design-doc.yaml):

- Sections: Research Question (required, ~100 words), Summary (required, ~150 words), Findings (required, min_subsections 2 — forces discrete findings over a wall of prose), Code References (required — file:line evidence, peek-able), Open Questions (required, zone: human — this is where questions BELONG in RPI, unlike plans).
- doc max_words 1500; markers.max 3.
- Review criteria: findings must carry file:line evidence; documentarian tone (describe, don't prescribe — research-rpi-templates.md:26); Summary answers the Research Question.

**Success Criteria**
- automated: `TestLoadBuiltinTemplates` covers `research`; a conforming fixture validates clean and a findings-as-prose fixture fails min_subsections; suite green under -race
- manual: this plan's own research doc, retrofitted, validates against the template

## Phase 2: `plan` built-in template

Mirror the HumanLayer plan shape (research-rpi-templates.md:18) with the discipline rules as enforced structure (research-rpi-templates.md:22):

- Sections: Overview (required, ~100 words), Current State (required), Desired End State (required, zone: human), What We're NOT Doing (required, zone: human — the scope fence), Implementation Phases (required, min_subsections 2), Risks (optional, ~150 words).
- doc max_words 2000 (~200 lines — the review-unit budget from research-rpi-templates.md:22); markers.max 0 set explicitly to 1: plans should have NO open questions, but one escape valve beats an agent guessing silently.
- Review criteria: every phase carries Success Criteria split automated/manual; no code blocks longer than a few lines (plans prohibit code — research-rpi-templates.md:22); Current State claims carry file:line citations the reviewer can peek.

**Success Criteria**
- automated: template loads + conforming/violating fixtures as in Phase 1; a fixture with 2+ markers fails the cap
- manual: this plan doc itself validates against the new template

## Phase 3: Flow documentation

- CLAUDE.md: add the RPI recipe to the review-flow section — research (template `research`) → signoff → plan (template `plan`, citing research file:line) → review with `f`-peek → gate → implement.
- skills/review-comments/SKILL.md: drafting-mode note — when writing a plan, cite the research doc by `file:line` so the reviewer can peek every claim; open questions go to the research doc's Open Questions, never the plan.

**Success Criteria**
- automated: none (docs)
- manual: one full RPI pass on the next real feature using both templates

## Risks

- **Template proliferation** — six built-ins start to be a catalog to learn. Contained: `template list` descriptions say when to use which; mini/design-doc/research/plan map to distinct moments.
- **Plans citing stale line numbers** after the research doc is edited. Accepted for v1: peek shows the drift immediately (the excerpt won't match the claim), which is itself the review signal.
