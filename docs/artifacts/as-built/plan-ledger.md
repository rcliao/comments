---
comments:
    template: as-built
description: Shipped behavior for lightweight plan status, approval freshness, and implementation context.
related:
    - path: ../plans/plan-ledger.md
      relation: informed_by
status: draft
title: 'As Built: plan-led implementation ledger'
type: AsBuilt
---

# As Built: plan-led implementation ledger

## What This Describes

This describes the `main` feature commit that contains this document, based on `cfc5577`.
It covers the lightweight Markdown phase ledger, approval freshness, and implementation context shipped from the approved plan.
It deliberately excludes execution orchestration, semantic evidence verification, and dedicated TUI/browser progress UI.

## Data Flow

1. A human and agent write stable intent under each H3 phase, then append dated entries under its H4 `Status` heading.
   The parser recognizes `pending`, `active`, `blocked`, and `done`, requires Summary/Evidence/Next, and treats the final entry as current. pkg/comment/planstatus.go:20
2. Plan validation keeps the original Markdown structure but masks status-entry lines from document, section, and phase word counts. pkg/comment/template.go:305
   A second validator applies the independent 20-entry and 60-word-per-entry limits. pkg/comment/planstatus.go:299
3. Human signoff records both the complete document hash and a stable-intent hash whose input excludes status entries. pkg/comment/gate.go:112
   Status-only edits can therefore change the complete document without making the approved intent stale. pkg/comment/planstatus.go:69
4. `comments context <plan> --for implementation` loads the plan and its review sidecar, then builds one shared implementation view. pkg/comment/context.go:104
5. The view assigns high-priority or blocking threads to their containing phase and derives `unapproved`, `stale`, `attention-needed`, `aligned`, or `complete`. pkg/comment/planstatus.go:94
6. The CLI renders the same structure returned as JSON by the existing MCP context tool. cmd/comments/knowledge.go:47 pkg/mcp/knowledge.go:37

## Data Model

```text
PlanStatusEntry                    # identity: phase + line; lives in parsed memory
  updated: YYYY-MM-DD
  state: pending|active|blocked|done
  summary: string
  evidence: string
  next: string
  line: integer

PlanPhaseStatus                    # identity: ordered H3 section; lives in parsed memory
  title, section_path
  start_line, end_line
  state, history_count
  latest -> PlanStatusEntry
  entries[] -> PlanStatusEntry
  success_criteria
  warnings[]
  attention[] -> CommentView

ReviewRecord                       # identity: append order/timestamp; persists in sidecar
  author, timestamp, decision, note
  document_hash
  intent_hash

PlanImplementationContext          # computed per context request; not persisted
  overall_status
  approval -> PlanApproval
  phases[] -> PlanPhaseStatus
  warnings[]
```

The in-memory ledger types are colocated with approval and overall-status derivation. pkg/comment/planstatus.go:25

`ReviewRecord` remains backward-compatible because both new hashes omit empty JSON fields. pkg/comment/types.go:139

## What Persists Where

- Stable plan intent and the append-only status lists live together in the Markdown plan; normal edits and version control preserve them.
- Review threads and signoff records live in `<plan>.md.comments.json`; sidecar writes remain atomic and include the review array. pkg/comment/storage.go:213
- A new signoff persists a full document hash for audit and an intent hash for freshness; legacy records without the latter load as `unknown` rather than being guessed current. pkg/comment/planstatus.go:69
- Parsed phases, latest status, warnings, attention routing, and overall state are recomputed for every context request and are lost without consequence on process exit.

## Known Gaps

- **Unowned:** Status parsing uses the bold `**Success Criteria**` marker as the end of a preceding status log; alternate phase layouts can warn until the convention expands. Reproduce by placing arbitrary prose between Status and Success Criteria. pkg/comment/planstatus.go:201
- **Deliberate:** Evidence is recorded and surfaced but not semantically proven; a human or external verifier still judges whether it supports the claim.
- **Deliberate:** Historical signoffs cannot reconstruct approved intent and report `unknown` until the next explicit approval. pkg/comment/planstatus.go:81
- **Unowned:** The browser and TUI show the Markdown lists but have no specialized weekly dashboard; dogfood should determine whether one is worth adding.

## Open Questions

None for this slice. The unresolved product question is whether ordinary rendered Markdown plus implementation context is sufficient during a longer real project.
