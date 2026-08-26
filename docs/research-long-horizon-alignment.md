# Research: where comments loses authority and memory across sessions, documents, and the implement boundary

## Research Question

Inside one document and one round, comments enforces well.
Eric's four pains — drift, review cost, skipped steps, weak traceability — sit outside that unit.

Q1. Do recent plans still pass the human gate, and is phase progress recorded anywhere?
Q2. Which skill directives are mechanically enforced, which are prose, and where does authority end at the implement boundary?
Q3. What can an agent learn at the start of a new session about where long work stands?
Q4. Which cross-artifact links exist — research, plan, code, commit, PR — and where do they break?

Out of scope: research convergence, owned by docs/plan-autonomous-research-convergence.md.

## Summary

Every signed-off plan dates from 2026-08-06 to 2026-08-11. [Q1]
Every plan since — three, two today — shipped code with `reviews[]` absent; approvals live in replies saying "accepted in chat".
No plan records phase progress.

Enforcement is real inside a document and vanishes at the only human gate. [Q2]
A signoff is free text with no actor check, `request_review` verifies nothing, and nothing checks a signoff exists before implementation.
Every agent-side discipline rule is prose.

At session start an agent learns what is blocking and what has new replies — nothing else. [Q3]
No phase, next step, cursor, or chain.

The only push channel, `watch`, never fires on a document edit. [Q3]
Citations resolve files, headings, and threads, not commits, PRs, or URLs; a moved line passes silently. [Q4]
The plan→research link `analyze` computes is never persisted.
The `as-built` template that would close the loop is referenced by no workflow.

## Findings

### F1 — every plan since 2026-08-11 shipped code without a recorded signoff [Q1]

Six of seven plans carry sidecars.
Three have a `reviews[]` entry — plan-tui-in-context (2026-08-06), plan-rpi-loop-strength (2026-08-07), plan-compounding-rpi (2026-08-11).
Each was implemented within fifteen minutes of approval (commits 61e68d2, 781ae2e, ed487d9).

The three written since have `reviews` absent from the sidecar entirely:

- docs/plan-landscape-improvements.md.comments.json
- docs/plan-autonomous-research-convergence.md.comments.json
- docs/plan-autonomous-research-eval-pilot.md.comments.json

All three have implementation in the working tree today:

- landscape-improvements Phases 2–3: pkg/comment/baseline.go, pkg/tui/model.go:73 (this session, uncommitted)
- autonomous-research-convergence Phases 1–3: cmd/comments/analyze.go, pkg/comment/analyze.go (a Codex session, uncommitted)
- eval-pilot: scripts/eval/autonomous-research/ scaffolding only

plan-artifact-quality-eval, research-artifact-quality-eval, and research-rpi-templates have no sidecar at all.
Signoff does not imply closure either.
docs/research-speckit-handoff.md.comments.json is `approved` (2026-08-12) over four open threads, one high-priority, untouched since 2026-08-09.

### F2 — approvals reach the record as agent replies saying "in chat" [Q1]

The skill requires that decisions made outside threads be recorded as threads the human ratifies (skills/review-comments/SKILL.md:272-274).
Two rounds show the two ways that rule is met.

On 2026-08-11 a chat decision was pulled back into a blocking thread on the plan.
The human ratified it by resolving, then an `approved` record landed (docs/plan-compounding-rpi.md.comments.json, thread c8j4g).

On 2026-08-21 the same pattern produced only agent replies (docs/plan-autonomous-research-convergence.md.comments.json, threads causo, c80rt, c2kre):

- "Human approved implementation in chat on 2026-08-21"
- "Accepted in chat: Phase 2 is good as scoped"
- "Accepted in chat: Phase 1 is good as scoped"

Every author in that sidecar is `codex`; no human wrote a thread or a review record.
The plan's one high-priority human-zone question (causo) is still open while the code exists.
The rule held in prose both times; only the first left a record the next round can inherit.

### F3 — phase progress is recorded nowhere; phase gating happened once, by hand [Q1]

No plan records per-phase status in the document, the sidecar, or a template field (pkg/comment/template.go:57-105, pkg/comment/storage.go:17-24).
The only phase-level evidence is commit messages ("Phase 2 decided — panel wins", 85d0b34) and the dogfood journal.
Success Criteria are demanded as review prose (pkg/comment/templates/plan.yaml:41) and checked by nothing after implementation — no code references them.

One plan was phase-gated.
plan-tui-in-context carries three `approved` records on one day (12:01, 13:54, 14:35), with a document edit between the second and third.
That is the intended pattern, executed manually, unrepeated since.
The journal names the cost: reviewers flagged items earlier sessions had already fixed, and "plan docs should record what the review snapshot was taken against" (docs/dogfood-journal.md:194).

The sidecar cannot answer that either.
It stores the template by name (pkg/comment/storage.go:23); `gate` re-resolves the name at gate time (cmd/comments/gate.go:85-92); templates changed eight times since June.
A signoff attests to a contract no record can reconstruct.

### F4 — authority is strong inside a document and absent at the implement boundary [Q2]

The tool refuses or fails on three things:

- agents resolving `zone: human` threads, on both surfaces (pkg/comment/actor.go:58-75; cmd/comments/main.go:661; pkg/mcp/tools.go:326)
- structural violations and blocking threads, via gate exit 10 (cmd/comments/gate.go:95-98,127)
- unresolvable or ambiguous `file:line` citations (pkg/comment/citations.go:86-148)

It accepts three things without question:

- `comments signoff --author <anything>` — no TTY or actor check, author free text defaulting to `$USER` (cmd/comments/gate.go:133-158)
- `comments_request_review` — only `os.Stat` before waiting on any newer review record (pkg/mcp/gate.go:72-78,95)
- implementation — no call site anywhere checks that a signoff exists

The actor concept exists in one place, `ResolveActor` (pkg/comment/actor.go:29-40), consulted by one verb: `resolve`.
Reply, accept, reject, and signoff never ask who is calling.
The refusal message itself names the override (`COMMENTS_ACTOR=human`, pkg/comment/actor.go:74).
The gate is meant to stay authoritative over recorded state only (docs/research-gate-authority.md:23-33); a signoff-exists check is state, not judgment, and is absent.

### F5 — every agent-side discipline rule is prose [Q2]

Directive against code:

- reanchor after editing (skills/review-comments/SKILL.md:228-232): orphan count reported by `status` (cmd/comments/parity.go:196-202), never required
- inbox first, decision second (skills/review-comments/SKILL.md:80-84): `BuildInbox` is a pure reporter (pkg/comment/inbox.go:26-72)
- never resolve blocking without a fix (skills/review-comments/SKILL.md:51-52): `ResolveThread` sets `Resolved` unconditionally (pkg/comment/helpers.go:166-175)
- ≤5 open threads, ≤50-word comments (skills/review-comments/SKILL.md:409-412,427-430): no count or length check on add or reply (pkg/mcp/tools.go:284,310)
- `analyze ready:true` before review (skills/review-comments/SKILL.md:328-340): advisory, exits 0 regardless (cmd/comments/analyze.go:13-14,85)
- interview, reviewer allowlist, history-first, no re-proposal: no code path

Thread-crossing is the observed failure.
Five replies in docs/research-skill-quality-surfacing.md.comments.json begin "(previous reply mis-mapped)", each correcting an answer posted to a sibling thread.

Editing the markdown leaves no actor trace.
The sidecar stores a content hash (pkg/comment/storage.go:19,107) and `watch` snapshots the sidecar only (pkg/comment/watch.go:61-64).
"Do not modify the document while waiting" (skills/review-comments/SKILL.md:165) is undetectable beyond a changed-lines count.
Suggestions never re-anchor: the cascade and `reanchor` move `Line` only (pkg/comment/anchor.go:160-162, pkg/comment/reanchor.go:59), so `StartLine`/`EndLine` go stale on a long-lived plan.
Accept refuses on an `OriginalText` mismatch (pkg/comment/applier.go:42) — loud, not repaired.

### F6 — session start answers "what is blocking", never "where are we" [Q3]

`comments_inbox` returns unresolved root threads that are blocking or have a reply newer than a caller-supplied `since` (pkg/comment/inbox.go:41-55).
On a directory it is a flat concatenation (pkg/comment/inbox.go:38-70).
`comments_status` adds counts, staleness, and — since this session — `changed_since` for one reviewer (pkg/mcp/tools.go:227-238).
`gate` on a directory aggregates decisions and counts (cmd/comments/gate.go:34-44).

Nothing reports which document is current, which phase is in progress, what was implemented, or what the next step is.
The skill's "inbox first" step is written for the moment a signoff arrives within a session (skills/review-comments/SKILL.md:80-89), not for a cold start days later.

### F7 — no cursor, no chain: the links an agent needs are recomputed or carried by hand [Q3] [Q4]

Every `since` is a caller-supplied RFC3339 string (pkg/mcp/inbox.go:27-33; pkg/mcp/gate.go:83,138).
`request_review` with `blocking:false` returns a timestamp handle the agent must keep (pkg/mcp/gate.go:80-87); no file persists it.
The one persisted per-reviewer marker is the review baseline (pkg/comment/baseline.go:37-40), scoped to one document and one verdict, and gitignored (.gitignore:44) — a fresh clone loses it.

`analyze --against` computes a real plan→research link — findings cited, excluded, or uncovered (pkg/comment/analyze.go:198-243) — and writes nothing.
`Analysis.Against` is output-only (pkg/comment/analyze.go:27).
The sidecar's only cross-artifact pointer is the template name (pkg/comment/storage.go:23).
No field exists for a linked document, an implements pointer, a phase, a PR, a commit, or an owner.

### F8 — citations reach files, headings, and threads; not commits, PRs, or moved code [Q4]

Three forms parse (pkg/markdown/refs.go:28,36,42): `[text](doc.md#heading)`, `thread:c1abc` or `thread:doc.md#c1abc`, and bare file-line, as in pkg/comment/gate.go:39-64.
Targets containing `://` are filtered out explicitly (pkg/markdown/refs.go:103-105); no SHA or `#123` form exists.
Git already carries the reverse link — commit bodies say "Implements docs/plan-X.md" (ed487d9, db63ee5) — and the tool never reads it: pkg/comment imports no `os/exec`.
The TUI peeks all three (pkg/tui/keys_refpeek.go:85-124); no CLI or MCP call resolves a citation for an agent.

Validation covers bare file citations only (pkg/comment/citations.go:95-97): existence, basename ambiguity, reversed range, past EOF (pkg/comment/citations.go:112-144).
A markdown link to a missing doc or a `thread:` to a dead ID is never reported.
A cited line whose code moved within the file passes and peeks the wrong lines.
No snippet, no hash, no re-anchoring exists for citations; the anchor cascade covers comments only (pkg/comment/storage.go:107-121).

### F9 — the as-built template is the closing artifact, and no workflow produces it [Q4]

`as-built` is the one template that gestures at the implement side.
It asks for "the exact commit or branch" (pkg/comment/templates/as-built.yaml:16) and marks each gap "already-ticketed (with the ticket) or unowned" (pkg/comment/templates/as-built.yaml:57).
Both are review prose, neither is a field.

It is referenced by the root README, USAGE.md:149, and docs/plan-compounding-rpi.md:27, which names the lineage "research → plan → as-built".
It appears in neither skills/review-comments/SKILL.md nor the root CLAUDE.md RPI paragraph.
That chain ends at "ONE human sitting on the plan".
docs/examples/as-built.md exists as a showcase; no RPI run has produced one.

### F10 — the wake-up channels carry review state, never document state [Q3]

`comments watch` is the agent's only push channel.
Its event set is review-only — comment, reply, resolve, suggestion, signoff, gate (pkg/comment/watch.go:12) — and its snapshot holds no document hash (pkg/comment/watch.go:49-57).
A plan edited mid-implementation fires nothing for a waiting agent; drift is invisible over the channel built to wake it.

The MCP side offers no push either.
`registerResources` registers two read templates (pkg/mcp/server.go:63-81); pkg/mcp contains no subscribe handler or resource-updated notification.
The root CLAUDE.md's MCP section describes them as "2 subscribable resources" — prose ahead of code.

## Code References

- docs/plan-*.md.comments.json — `reviews[]` present on three plans, absent on three (F1)
- docs/plan-autonomous-research-convergence.md.comments.json — "accepted in chat" replies (F2)
- pkg/comment/actor.go:29-40,58-75 — the only actor check, consulted by `resolve` alone (F4)
- cmd/comments/gate.go:133-158 — signoff with free-text author (F4)
- pkg/mcp/gate.go:72-95 — request_review pre-checks (F4)
- pkg/comment/inbox.go:26-72 — the session-start surface (F6)
- pkg/comment/analyze.go:27,198-243 — computed, unpersisted plan→research link (F7)
- pkg/comment/storage.go:17-24 — six sidecar fields, no links (F7)
- pkg/markdown/refs.go:28-42,103-105; pkg/comment/citations.go:95-144 — citation reach (F8)
- pkg/comment/templates/as-built.yaml:16,57 — the orphaned closing template (F9)
- pkg/comment/watch.go:12,49-57; pkg/mcp/server.go:63-81 — review-only wake-up channels (F10)

## Open Questions

- Does this justify a plan at all? The findings could become two skill rules plus a habit; no-action is legitimate if you prefer to keep the tool small.
- Scope pick: the evidence points at three candidates — a session-start "where are we" surface, a mechanical signoff-before-implement boundary, or a persisted doc chain with commit/PR citations. The plan will choose one; redirect now if one is off the table.
- The working tree holds two uncommitted, unsigned implementations (F1). Park them, review them, or commit them before the plan lands?
