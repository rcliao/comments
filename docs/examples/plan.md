---
comments:
  template: plan
description: A phased implementation plan for making uncertain anchor recovery visible.
related:
  - path: research.md
    relation: informed_by
status: stable
title: "Plan: surface fuzzy re-anchors at review time"
type: Plan
---

# Plan: surface fuzzy re-anchors at review time

## Overview

Fuzzy re-anchors (a comment that moved by normalized text match) currently succeed silently; only orphans announce themselves.
Surface them: a `≈` sigil in the sidebar and a count in doctor, so a reviewer knows which threads drifted before trusting their anchors.
Follows docs/examples/research.md (the anchor cascade), which left this as its open question.

## Current State

- The cascade labels normalized matches `fuzzy` in `AnchorConfidence` (docs/examples/research.md:33-35).
- Sidebar rows show sigils for blocking/priority/suggestions but nothing for confidence.
- `doctor` counts orphans per sidecar; fuzzy counts are invisible everywhere.

## Desired End State

A reviewer opening a heavily-edited doc sees `≈` on drifted threads and can jump to each to confirm the anchor still points where the author meant.
Verify: one doc edited enough to force fuzzy matches, reviewed with the sigil visible.

## What We're NOT Doing

- No auto-resolution or re-prompting of fuzzy threads — the human confirms; the tool only discloses.
- No new thread state: `AnchorConfidence` already stores the fact.

## Implementation Phases

### Phase 1: sidebar sigil

`threadMarkers` gains `≈` when `AnchorConfidence` is fuzzy and the thread is open.
The walkthrough order (`P`) is unaffected; the sigil rides existing machinery.

#### Status

- 2026-01-15 — **active**
  - Summary: Adding the review-time confidence marker.
  - Evidence: thread:example
  - Next: Exercise the fuzzy-anchor fixture.

**Success Criteria**
- automated: fuzzy fixture renders `≈`; resolved stays quiet
- manual: sigil legible in both densities

### Phase 2: doctor count

The sidecars check reports fuzzy counts beside orphans ("2 fuzzy re-anchors").

#### Status

- 2026-01-15 — **pending**
  - Summary: Waiting for the sidebar phase.
  - Evidence: —
  - Next: Implement after Phase 1.

**Success Criteria**
- automated: doctor fixture asserts the count
- manual: none — reporting change

## Risks

- **Sigil noise on old docs** whose lazy backfill marked many threads fuzzy.
  Mitigated: only OPEN fuzzy threads get the sigil; the resolved trace stays quiet.
