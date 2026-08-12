# Plan: landscape-driven improvements to the comments review loop

## Overview

Two improvements from the landscape research (docs/research-landscape-2026-08.md).
First: decompose the review skill under an instruction budget (F2).
Second: line-level "changed since your last signoff" marks in the TUI (F3).
Done means: SKILL.md is a thin router under ~120 lines with phase files loaded on demand,
and re-reviewing a doc shows which lines changed since the reviewer's last recorded pass.

## Current State

skills/review-comments/SKILL.md is one roughly 400-line file agents load whole (docs/research-landscape-2026-08.md:44-46).
Review-round memory is thread-level only: NEW badges via thread activity,
nothing marks changed document lines (docs/research-landscape-2026-08.md:52-56).
Both signoff writers record review passes through AddReviewRecord (pkg/comment/gate.go:112),
and the renderer already owns a per-line gutter (pkg/tui/rendering.go:287).

## Desired End State

An agent in any single phase reads only that phase's instructions.
Verify: each phase file stays under 40 directive bullets (counted by script), router under 120 lines.
A reviewer opening a previously signed-off doc sees changed lines marked in the gutter.
Verify: edit a signed-off doc, reopen the TUI, changed lines carry the mark; unchanged lines do not.

## What We're NOT Doing

No CRISPY-style human checkpoints between research and plan — that reverses the 2026-08-11 autonomous-chain decision.
No structure-outline artifact yet — carried as a review question, not planned work.
No spec-kit-style hooks/presets, no non-Claude packaging.
No browser surface.
No snapshot history — diffing keeps exactly one baseline per doc per reviewer (the latest), nothing more.

## Implementation Phases

### Phase 1 — Skill decomposition: router + phase files

SKILL.md becomes a router: mode detection, phase map, and universal conventions only.
Six phase files land in skills/review-comments/phases/ (draft, seed, respond, review, compound, autonomous).
Each is self-contained and under 40 directive bullets.
The router tells the agent which ONE file to read for its current phase.

A directive bullet is any list line: grep -E '^\s*([-*]|[0-9]+\.)\s'.
Success criteria — automated: scripts/skill-budget.sh (wired into scripts/ci.sh) fails on any phase file over 40 bullets or router over 120 lines.
Manual: run one full RPI cycle; the agent never needs a phase file it wasn't routed to.

### Phase 2 — Record last-reviewed content at signoff

New pure helper comment.SaveReviewBaseline(docPath, author, content), called explicitly by BOTH writers
(pkg/tui/keys_verdict.go:64 and cmd/comments/gate.go:158) right after AddReviewRecord — which stays pure and path-free.
Baseline lives at .comments/baselines/<doc-relpath, slashes as __>.<author>.md, latest only; directory gitignored.
Decisions approved and changes_requested update the baseline; commented (reply-pass) does NOT — the mark stays "since your last verdict".

Success criteria — automated: unit test proves both writers store identical baselines and reply-pass leaves it untouched; scripts/ci.sh green.
Manual: none — invisible plumbing.

### Phase 3 — Changed-since-signoff marks in the TUI

On document load, diff current text against the viewer's stored baseline (line-level LCS).
Changed/added lines render their line NUMBER in a distinct diff accent — no new gutter cell, no width change,
no collision with the occupied 2-char marker cell (pkg/tui/rendering.go:289).
No baseline or no prior review → no marks, zero cost.

Success criteria — automated: renderer test asserts marks on changed lines only; byte-identity tests still pass (marks are gutter-only).
Manual: Eric reopens an edited signed-off doc and can tell at a glance what moved since his pass.

## Risks

Baseline files grow clutter — accepted: one per doc per reviewer under gitignored .comments/baselines/; revisit if noisy.
Phase-file split could drift from the router — mitigated: skill-budget script also asserts every routed file exists.
LCS on large docs could lag redraws — mitigated: diff once at load, cache per document hash.
