# Documentation map

**Audit date:** 2026-08-26

The repository keeps three kinds of documentation: maintained references,
active proposals, and point-in-time design/research records. Only the first
group describes the current product end to end. In historical artifacts,
sections named “Current State” mean the state when that artifact was written.

## Maintained references

| Document | Purpose |
|---|---|
| [README](../README.md) | Product overview, install, and first review loop |
| [Usage guide](../USAGE.md) | Current CLI, TUI, template, gate, and anchor workflows |
| [OKF bundle guide](OKF.md) | OKF v0.2 boundary, Comments extensions, default taxonomy, context modes, and RPI workflow |
| [Architecture](ARCHITECTURE.md) | Current components, schema, invariants, and durable constraints |
| [Development guide](../CLAUDE.md) | Build gates and repository-wide implementation conventions |
| [TUI guide](../pkg/tui/CLAUDE.md) | Bubbletea modes, composition, rendering, and TUI invariants |
| [Review skill](../skills/review-comments/SKILL.md) | Current agent review and RPI workflow |
| [Template examples](examples/) | Maintained examples that validate against every built-in template |
| [Dogfood knowledge bundle](artifacts/) | Live OKF research and implementation brief used to validate bundle creation and context traversal |
| [Eval guide](../scripts/eval/README.md) | Current template-evaluation harness and recorded runs |

`comments help`, the embedded templates in `pkg/comment/templates/`, and the
MCP registrations in `pkg/mcp/server.go` are the executable sources of truth
for their respective surfaces.

## Proposed work

These documents describe work that is not fully implemented. They are design
inputs, not promises about current behavior.

| Document | Status |
|---|---|
| [Closed-loop feature delivery plan](plan-closed-loop-feature-delivery.md) | Proposed. Unifies artifact lineage, scoped approval, resumability, and implementation evidence behind one lightweight protocol. |
| [Closed-loop feature delivery research](research-closed-loop-feature-delivery.md) | Research basis for the closed-loop plan and its product boundary. |
| [Autonomous research convergence plan](plan-autonomous-research-convergence.md) | Phases 1–3 implemented: validation parity, `analyze`, convergence skill, and paired eval harness. Phase 4 dogfood awaits a promoting blinded pilot. |
| [Autonomous research convergence research](research-autonomous-research-convergence.md) | Research basis for the convergence plan. |
| [Artifact-quality eval plan](plan-artifact-quality-eval.md) | Partially implemented. The narrower `comments analyze` mechanical layer and autonomous paired-eval harness exist; post-signoff probe logging remains proposed. |
| [Landscape improvements plan](plan-landscape-improvements.md) | Partially implemented in the current worktree: changed-since-signoff baselines and marks landed; skill decomposition remains deferred. |
| [Artifact-quality research](research-artifact-quality-eval.md) | Research basis for the eval plan. |
| [2026 landscape research](research-landscape-2026-08.md) | Dated market/workflow snapshot supporting the landscape plan. |
| [Diagram-rendering research](research-diagram-render.md) | Partially adopted: fenced syntax highlighting shipped; injected ASCII diagram rendering remains deferred and its review has open blockers. |
| [Spec Kit handoff research](research-speckit-handoff.md) | Open exploration with unresolved human questions; no handoff integration exists. |

## Retained design records

These artifacts describe implemented decisions whose rationale is still useful
when changing anchors, the TUI, or the RPI workflow. Their implementation
checklists are historical; use the maintained architecture and code for current
behavior.

| Document | Why it remains |
|---|---|
| [Anchoring refactor](design-anchoring-refactor.md) | Explains the exact-text cascade, short IDs, and why orphaning preserves history. |
| [Rendering and thread priority](design-markdown-render.md) | Records why thread legibility preceded full markdown rendering and why source-line truth wins. |
| [Reference jump](design-reference-jump.md) | Records the peek plus editor-handoff choice over a multi-buffer TUI. |
| [Review-first TUI](design-tui-review-first.md) | Captures the review motions, verdict, queued decisions, and persistence goals. |
| [In-context TUI plan](plan-tui-in-context.md) | Preserves the prototype-driven choice of a side-panel takeover and composited dialogs. Implemented. |
| [RPI loop-strength plan](plan-rpi-loop-strength.md) | Preserves the interview, fresh-reviewer, attention-budget, and lightweight-scope decisions. Implemented in the skill. |
| [Compounding RPI plan](plan-compounding-rpi.md) | Preserves thread-citation syntax and veto-history rules; maintained examples cite its review threads. Implemented. |
| [Gate authority research](research-gate-authority.md) | Explains why the gate blocks on explicit thread/template state instead of semantic model judgment. |
| [RPI harness comparison](research-rpi-harness-comparison.md) | Records the deliberate scope fence at plan signoff and the evidence behind it. |
| [Skill quality surfacing](research-skill-quality-surfacing.md) | Connects reviewer history, eval telemetry, and the later compounding/eval proposals. |
| [Dogfood journal](dogfood-journal.md) | User-research log of failures and design lessons that produced current constraints. |

The following research documents are retained as the evidence trail for those
implemented decisions: [agent surface](research-agent-surface.md), [RPI loop
strength](research-rpi-loop-strength.md), [RPI templates](research-rpi-templates.md),
and [in-context TUI](research-tui-in-context.md).

## Retention rule

Keep a document when it is one of:

- a maintained reference with an identifiable current source of truth;
- an unimplemented proposal that still represents intended work;
- a decision or research record containing durable rationale, rejected
  alternatives, or cited review-thread history.

Delete completed execution checklists, scratch plans, redundant demos, and
runtime state after their durable decisions have landed in architecture or a
retained design record. Git history remains the archive; the working tree
should not make completed work look active.
