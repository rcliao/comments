# Comment Anchoring & Ergonomics Refactor (v2.1)

## Problem

Comments anchor by line number + section path. Agent rewrites between review rounds — the tool's core workflow — degrade those anchors:

- **Line precision is discarded at creation.** A comment on line 7 (a specific sentence) is silently snapped to line 3 (its section heading). The reviewer pointed at a sentence; the agent addressing it sees only a section.
- **Section rewrites keep comments alive at heading granularity only** — anything finer is already gone by the second round.
- **Threads pile up on heading lines** (a side effect of snapping), so multiple comments on one heading are hard to scan in the TUI without opening each thread.
- **IDs are 19-digit timestamps** (`c1785950795763444000`): unreadable in listings, untypeable in `reply --thread`, collision-prone in batch loops.

This hurts most in the v2.1 review-gate workflow — seeded template threads, blocking comments, `request_review` — which assumes threads stay meaningfully attached across many rewrite cycles. Today each cycle erodes anchor quality until orphaning is the only honest fallback.

## Goals / Non-Goals

**Goals:**

- **G1** — comments keep their exact position across agent rewrites whenever the referenced text still exists
- **G2** — IDs are friendly to read in listings; humans should rarely need to type one
- **G3** — reviewing happens in the TUI with the sidebar following the cursor: threads anchored at the focused line auto-expand for a glance, dive in to reply
- **G4** — existing sidecars keep working with no manual migration; gate/template/MCP surface unchanged

**Non-goals (out of scope):** real-time/multi-user collaboration or CRDTs; alternative storage backends (git notes, database); non-markdown formats; MRSF import/export (separate effort). A review-first TUI redesign is *in* scope in principle (per review), with G3 as this doc's slice — the full redesign gets its own design doc.

## Proposed Design

Each element is tagged with the goal it serves.

**Content anchors (G1).** Each comment gains an `Anchor` captured at creation: the exact text of its target line plus one line of context before and after. Suggestions already store `OriginalText`; this generalizes the same idea to comments.

**Re-anchor cascade on load (G1).** When the document hash changes, each comment re-anchors individually, best match wins: (1) target line unchanged at stored position — keep; (2) exact text search — move there; (3) normalized search (whitespace/case-insensitive) — move, marked `anchor_confidence: fuzzy`; (4) section-path fallback — attach to the heading, marked `section-level`; (5) orphan, via the existing lifecycle. Confidence appears in `list`/gate output so fuzzy re-anchors are spot-checkable. Step 3 is normalized matching only in v2.1 (decided); edit-distance can come later if orphan rates warrant it.

**Agent-assisted anchor migration (G1).** The agent editing a commented doc knows exactly how its edits moved text — so the editor is responsible first: after editing, the agent calls a new `comments_reanchor` MCP tool (batch `comment_id → new line/section`) to migrate the anchors it displaced, and the skill makes this a required post-edit step. The load-time cascade stays as the safety net for edits made outside the loop (humans, other tools). Precision is kept at edit time rather than reconstructed afterward.

**Stop snapping to headings (G1, G3).** `add --line N` stores N exactly. Section path stays as computed metadata for display, filtering, and cascade step 4 — it no longer overwrites the line. This also removes the root cause of heading pile-up.

**Short display IDs (G2).** New comments get random base36 IDs, 4 chars with collision check, growing on collision (`c7f3k`). Existing long IDs stay valid — lookup goes through `FindCommentByID` unchanged. Division of labor per review: humans use `view` and friendly IDs for referring; agents use full IDs. No CLI prefix-matching (decided: unnecessary).

**Focus-follows-cursor review view (G3).** As the reviewer moves through the document line by line, the sidebar tracks the focused line: threads anchored there auto-expand for glanceable context, and stacked threads show grouped with a count badge (`Line 17 · 3 threads`); enter to dive in and reply. This is the review slice of the larger TUI redesign (follow-up doc).

**Lazy anchor backfill (G4).** Sidecar format stays `2.0`-compatible: `Anchor` is a new optional object. On first load of an old sidecar, anchors are captured from current document content (correct when the hash matches; stale sidecars follow the existing orphan path first). No migration command.

**Schema unification (G4).** MCP tool outputs adopt the CLI's snake_case JSON shape — one serializer shared by `list --format json`, gate, and MCP — removing the two-schema split agents currently face.

## Options Considered

### Option 1: Content anchors with re-anchor cascade (recommended)

Per-comment anchors with graceful degradation — the Hypothesis/MRSF/Commentary consensus design. Most implementation work of the three, but it fixes precision loss, rewrite churn, and heading pile-up with one mechanism, offline, no new dependencies. Recommended because it addresses G1 and G3 at the root rather than symptomatically.

### Option 2: Git-diff line rebasing

Map old line numbers to new via `git diff` between the sidecar's last-seen state and the working tree. Substantially less code. Rejected: mid-session agent edits aren't commits — the diff base is usually wrong between commits — and it ties core behavior to git presence. Viable later as a cascade optimization, not a foundation.

### Option 3: Inline anchor markers in the markdown

Invisible HTML comments at anchor points (md-redline's approach): perfectly self-anchoring, but they pollute the document and diffs, and agents rewriting a section routinely delete or duplicate them — silent, unrecoverable failure.

## Risks

Only risks without a safe blanket mitigation are listed:

- **Fuzzy re-anchor mis-attaches a comment** to a similar line. Contained: fuzzy matches carry visible confidence labels, and matches below the bar orphan instead of guessing. Residual risk accepted — mis-attachment is visible and correctable, unlike today's silent precision loss.
- **Anchor backfill on an already-stale sidecar** could capture wrong text. Contained: backfill runs only when the hash matches or after the orphan pass has flagged mismatches. Residual risk accepted.

## Unresolved Questions

TODOs for the human reviewer — resolve the matching thread or edit here:

- [x] Fuzzy matching depth: normalized-only for v2.1 (decided in review, 2026-08-05)
- [ ] Should accepted/rejected suggestions auto-resolve their thread so they leave the default `list` view?
- [ ] Should `--section` accept suffix matches ("Problem Statement" for the full path)? Cheap via the template engine's matcher, but changes CLI behavior.
- [ ] Does the `Anchor` field subsume suggestions' `OriginalText` drift-guard, or do suggestions keep their own verification field?
