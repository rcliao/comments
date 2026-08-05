# Markdown Rendering & Thread Tracking Priority

## Problem

The approved TUI redesign chose render-first — but a later review comment (thread `cyidb`) reframed the premise:

- **Reading raw markdown is not the core pain.** "I often need to review the agent prompt/replies anyway" — raw text is Eric's daily medium.
- **The core pain is tracking conversation threads/branches within the doc** — which reply answered what, which decision landed where, what changed since my last pass.
- Rendering remains real polish (styled prose reads faster), but building it first would optimize the secondary problem.

This doc decides the order and shape of both workstreams before the big build starts.

## Goals / Non-Goals

**Goals:**

- **G1** — the review surface makes conversation state legible: what's open, what changed since my last pass, how a thread evolved
- **G2** — reading comfort improves without breaking anchor truthfulness
- **G3** — the chosen order ships value in small verified steps (no month-long rendering rewrite)

**Non-goals:** web UI; inline images/mermaid (ASCII diagrams are the template-encouraged medium); directory sessions (separate design).

## Proposed Design

The two workstreams, and the order this doc proposes:

```
            ┌─ Phase A: thread tracking (G1) ──────────────┐
            │ 1. round markers: what changed since my      │
            │    last signoff (new/edited badge per thread)│
            │ 2. thread timeline in sidebar: replies in    │
            │    order with round separators               │
            │ 3. `]r` jump to threads with new replies     │
            └──────────────────────────────────────────────┘
                              │ then
                              ▼
            ┌─ Phase B: rendering polish (G2) ─────────────┐
            │ 4. in-place span styling (bold/code/lists    │
            │    styled, syntax dimmed — no reflow)        │
            │ 5. full rendered mode + block↔line mapping   │
            │    (the approved-but-parked big build)       │
            └──────────────────────────────────────────────┘
```

**Phase A — thread tracking first (G1).** Uses data we already store: signoff timestamps partition thread activity into review rounds. (1) Each thread shows a `NEW` badge when it has replies newer than your last signoff; (2) the expanded sidebar thread gets round separators (`── round 2 ──`) so a conversation's evolution is scannable; (3) `]r`/`[r` jump between threads with unseen activity — the inbox motion, mirroring the agent-side `comments_inbox`.

**Phase B — rendering as polish, staged (G2, G3).** First the cheap 80%: style spans in place (bold/italic/code rendered, syntax glyphs dimmed, list bullets colored) with zero reflow — anchors untouched. Only then the full rendered mode with block↔line mapping from the approved TUI design, if in-place styling still feels insufficient in daily use. [NEEDS CLARIFICATION: does this order match your instinct — thread tracking (Phase A) before any rendering work, with full rendered mode contingent on in-place styling proving insufficient?]

## Options Considered

### Option 1: Thread tracking first, rendering staged behind it (recommended)

Matches the reframe: the unsolved problem (conversation legibility) ships first from existing data; rendering lands as two cheap-then-expensive stages with an off-ramp if the cheap stage suffices.

### Option 2: Full rendered mode first (the original approved order)

Honors the earlier render-first decision as written. Rejected: it front-loads the highest-risk build (block↔line mapping) against the need the review demoted.

### Option 3: Both in parallel via agent team

Fastest wall-clock; rejected as default because both touch pkg/tui rendering paths — merge risk where our parallel-agent success relied on disjoint files. Viable later for Phase B while Phase A stabilizes elsewhere.

## Risks

- **Round partitioning depends on signoff discipline** — reviews without signoff blur "since last pass". Contained: fall back to last-viewed timestamp from view-state; accepted.
- **In-place styling may prove insufficient**, making Phase B's big build mandatory after all. Accepted — that's the off-ramp's purpose: the decision gets made on lived experience, not speculation.

## Unresolved Questions

- [ ] Phase order confirmation (marker in Proposed Design)
- [ ] Should `NEW` badges clear on view (focused once) or only on reply/resolve?
