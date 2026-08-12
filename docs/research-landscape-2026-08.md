# Research: comments vs the 2026 RPI/SDD and terminal-review landscape

## Research Question

Q1. What have the reference flows (HumanLayer, spec-kit) changed since our Aug 7 baseline, and what do those changes imply for our loop?
Q2. What do dedicated plan/doc review tools (Plannotator, revdiff) offer that our review surface lacks?
Q3. Where does comments stand in the terminal viewer/review niche, and which adoption-worthy gaps follow?

## Summary

HumanLayer replaced RPI with CRISPY: seven stages, two NEW intermediate human checkpoints (design discussion, structure outline), and a reversal — engineers read code, not long plans. [Q1]
Their sharpest transferable insight is an instruction budget: monolithic 85-instruction prompts failed; everything decomposed under 40 — and our own SKILL.md has grown into exactly the monolith they abandoned.

Plannotator's review surface adds one thing ours lacks outright: a revision diff between review rounds — what changed since I last looked. [Q2]
Everything else it offers (exact-text annotation, send-to-agent, approve) we match or exceed in-terminal with threads, gate, and watch.

No surveyed tool combines viewer, line-anchored threads, and a machine-readable gate in the terminal; that combination is ours alone. [Q3]
The compounding layer (thread citations, history-first drafting, veto persistence) has no equivalent anywhere surveyed.

## Findings

### F1 — HumanLayer's CRISPY supersedes RPI, adding checkpoints where we removed one [Q1]

CRISPY = Context, Research, Iterate (design discussion), Structure (outline), Plan, sYnthesize, Implement.
Two human checkpoints sit BETWEEN research and plan.
First a ~200-line design-discussion doc: current state, end state, patterns, decisions, open questions.
Then a ~2-page structure outline — "function signatures and types without implementation," a C-header for the plan.

Their motivation, quoted: agents "skip crucial interaction steps and immediately generate complete plans."
And reviewing 1,000-line plans gave poor leverage since "the implemented code would often differ from the plan."

Our autonomous chain went the OPPOSITE direction on the same evidence — we removed the research gate because the human added no value there (decided 2026-08-11).

The positions are compatible: their checkpoints are alignment artifacts BEFORE plan detail.
Our carried-questions + pause-on-shape policy covers the same risk with threads instead of documents.
What we lack from their split: a structure-level artifact smaller than the full plan — ours holds design and tactics in one document.

### F2 — The instruction budget: their hardest-won lesson indicts our skill file [Q1]

Frontier models reliably follow ~150-200 instructions TOTAL across system prompt, tools, and skill.
HumanLayer's 85-instruction planning prompt exceeded the budget in combination, so models "only partially attended."
CRISPY decomposes every prompt under 40 instructions.

Our skills/review-comments/SKILL.md carries the whole loop in one file agents load whole: drafting, callouts, reviewer passes, compounding, autonomous chain, conventions.
At 372 lines with 53 directive bullets, the skill alone consumes a large fraction of that whole-context budget — before the system prompt and tools take their share.
It is structurally the monolith they abandoned; no decomposition mechanism exists in our skill today.

### F3 — Plannotator: browser review with one feature we genuinely lack [Q2]

Plannotator opens plans in a LOCAL BROWSER: annotate exact text, mark for removal, edit, send-feedback or approve.
It hooks into Claude Code, Codex, OpenCode, VS Code and others.

Its differentiator against us: on resubmission it shows a change badge and an added/removed diff versus the last reviewed version.
That is review-round memory at line level.

Our equivalent is thread-level only: NEW badges mark threads with fresh activity.
The document pane has no "changed since your last signoff" signal.
A reviewer re-reads or trusts git.
revdiff (whose exit-code convention our gate borrowed) occupies the same browser-review space for code diffs.

### F4 — spec-kit: baseline gap-fill, not post-baseline news [Q1]

spec-kit's post-baseline delta is negligible: v0.16.1 (Aug 7) and v0.16.2 (Aug 10) are maintenance releases.
Its extensions/presets system (per-event hooks, priority ordering, template overrides) landed in v0.12.0 on June 28 — BEFORE our baseline, which did not cover it.
So this finding fills a baseline gap rather than reporting news.

Artifacts there are explicitly additive historical records — clarifications append, never destroy.
Our sembr + sidecar + review rounds already deliver that with finer grain.
Our template system (.comments/templates) matches their customization at the YAML level but has no hooks/presets concept.
Our MCP server is agent-agnostic in principle but documented and packaged only for Claude Code.

### F5 — The terminal niche is ours; nobody else combines the three layers [Q3]

Terminal markdown viewers (glow-class) render read-only.
glamour remains whole-document with no source-line mapping (research-diagram-render.md:52-54), so they cannot host line-anchored review.
Plan-review tools went to the browser (F3).

comments is the only surveyed tool holding viewer + line-anchored threads + machine gate in one terminal surface.
This week added fence highlighting, aligned tables, search, and walkthrough ordering (pkg/tui).
The compounding layer — thread citations, history-first drafting, veto-to-alternatives — appears in no surveyed tool in any form.

### F6 — What the comparison says we should NOT adopt [Q1] [Q2]

CRISPY's two extra human checkpoints would reverse the autonomous-chain decision the human just made for stated reasons.
Adopt the artifact granularity, not the gates.
Plannotator's browser surface trades away the terminal-native property that differentiates us (F5).
spec-kit's 30-agent integration breadth is a distribution play, not a review-quality lever.
Our depth-first bet — one agent ecosystem, richest contract — is the opposite strategy and currently our advantage.

## Code References

- skills/review-comments/SKILL.md — the monolith F2 indicts
- pkg/comment/gate.go:21-23 — the revdiff-convention exit code
- docs/research-rpi-loop-strength.md — Aug 7 baseline this extends
- docs/research-diagram-render.md:52-54 — glamour/source-line finding reused in F5

## Open Questions

- Structure-outline artifact: adopt as an optional chain stage (a 2-page skeleton the human can glance at mid-chain), or fold as a required plan section?
- Skill decomposition (F2): split SKILL.md by phase into loadable parts, or compress the one file under a budget?
- Round-over-round doc diff (F3): line-level changed-since-signoff in the TUI — worth its cost?

Sources: [CRISPY writeup](https://www.zenml.io/llmops-database/evolving-ai-coding-agent-workflows-from-research-plan-implement-to-crispy) · [Plannotator](https://plannotator.ai/) · [spec-kit releases](https://github.com/github/spec-kit/releases)
