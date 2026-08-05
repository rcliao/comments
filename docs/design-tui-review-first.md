# Review-First TUI Redesign

## Problem

The TUI works but is built for *browsing* comments, not *conducting a review*:

- **Reading raw markdown is tiring.** Headings are styled, but body prose shows `**bold**`, `` `code` ``, and list syntax verbatim — reviewers parse markup instead of reading. Review here is mostly *design docs*, not source code: prose comfort is the primary need (decided in review).
- **No review motions.** Getting to "the next thing needing my attention" takes line-by-line cursoring; there is no next-open-comment jump, no TOC, no way to triage a long spec section by section.
- **Acting breaks the flow.** Composing happens in a small textarea (painful for long feedback); accepting/rejecting a suggestion has no in-TUI path at all.
- **Exiting is a dead end.** After reviewing, the human must separately run `comments signoff` — the TUI knows the review state but never asks for a verdict, so the gate handshake lives outside the tool's main surface.
- **State evaporates.** Reopening a doc starts at the top; a half-finished review of a 300-line spec loses its place.

Research (dogfood journal, report 4) shows no terminal tool combines rendered prose with line anchors; the closest tools (octo.nvim, prr, revdiff) stay source-first because they review code. We review prose — so we take the render-first path they didn't need.

## Goals / Non-Goals

**Goals:**

- **G1** — reading is comfortable by default: rendered prose, not raw markup; anchors stay truthful through a block↔line mapping
- **G2** — "what needs me next" is one keystroke: open-comment motions and section-level triage
- **G3** — every review action (comment, reply, resolve, accept/reject) completes without leaving the reading flow
- **G4** — exiting asks for the verdict: `view` is the interactive gate

**Non-goals (out of scope):** directory/multi-file review sessions (separate design); web or rendered-HTML UI; mouse-first interaction; editor round-trip mode (deprioritized in review — focus on the tool, not editor integration); changing any CLI/MCP surface.

## Proposed Design

Each element is tagged with the goal it serves.

**Rendered reading view as the default (G1) — decided in review.** The document renders as styled prose (glamour or equivalent), block by block: the renderer processes each top-level markdown block (paragraph, list, code fence, heading) separately, and each rendered block carries its source-line range from the existing section/block parser. Anchors therefore stay truthful at block granularity: gutter markers and virtual-text summaries (`· @rcliao ×2, 1 open` — on by default, `L` toggles) attach to the rendered block containing the comment's line.

**Cursor moves by block in rendered mode (G1, G2).** `j`/`k` step through rendered blocks (a paragraph is one stop, not five wrapped lines); the focused block drives the sidebar exactly as the focused line does today. Adding a comment in rendered mode anchors it to the focused paragraph (its first source line) — when an exact line matters, press `v` for source mode first (decided by default in review: paragraph-level is right for prose feedback).

**Source mode on `v` (G1).** Toggles to today's line-grid view (line numbers, exact line cursor, in-place heading styling) at the same scroll position — for precise line anchoring, suggestion ranges, and agent-eye verification. Render-first, source-on-demand: the inverse of the pre-review draft, per Eric's call.

**Review motions (G2).** `]c`/`[c` next/previous open comment (vim-adjacent, decided in review; resolved threads are skipped), `t` TOC overlay from the section parser with per-section open-comment counts, Enter jumps. Reading position and focused thread persist per document; reopening resumes the review.

**Compose without leaving the flow (G3).** `c` opens a centered modal overlay pre-filled with anchor context; `Ctrl+E` from any compose drops into `$EDITOR` for long text. `r` on an expanded thread replies in place (shipped 2026-08-05); `R` toggles resolve on the focused thread.

**Suggestions in-TUI (G3) — decided in review.** Focused suggestion threads show a word-level old→new preview; `a`/`x` mark them accept/reject **pending** — nothing mutates mid-review. All queued decisions apply atomically when the verdict is submitted (`q` dialog), so the document you read is the document you reviewed.

**Exit is a verdict (G4) — decided in review.** `q` opens the verdict dialog: **Approve** (writes signoff, exits 0) / **Request changes** (writes signoff, exits 10) / **Back to review**. `Ctrl+C` remains the silent escape path, quitting without a review verdict. An SDD checkpoint becomes: run `comments view`, review, pick a verdict.

**Sidebar density toggle (G1) — decided in review.** Sidebar stays visible by default; `S` cycles full → condensed (one line per thread, counts only) → hidden, for reading-heavy passes.

## Options Considered

### Option 1: Render-first with block-level anchors (recommended — chosen in review)

Rendered prose by default, cursor and markers at block granularity, source mode one key away for line precision. Chosen because this tool's documents are tech design docs, not source code — reading comfort is the job — and the block↔line mapping keeps anchors honest where octo.nvim-style tools never needed to try.

### Option 2: Source-first with in-place styling

Keep the line grid, style spans in place, never reflow. The pre-review recommendation — rejected as the default by review: it optimizes for anchor precision the reviewer rarely needs over reading comfort they always need. Survives as the `v` source mode, where its precision belongs.

### Option 3: Editor round-trip only (prr model), minimal TUI

Quote the doc into `$EDITOR`, comments as interleaved text, parsed on save. Deprioritized in review: build the tool's own surface first; revisit if editor-native reviewers materialize.

## Risks

Only risks without a safe blanket mitigation are listed:

- **Block↔line mapping is the new hard part.** Glamour renders whole documents and exposes no source positions, so we must render per block and own the mapping; code fences, tables, and nested lists are the fiddly cases. Contained: the section/block parser already produces line ranges; per-block rendering is testable as pure functions; fall back to source mode (`v`) whenever mapping is ambiguous. Residual risk accepted — worst case is a block-level anchor where a line-level one was wanted, visible and correctable in source mode.
- **Rendered mode hides what agents will see.** Comments anchored while reading rendered prose may sit on unexpected source lines (e.g. a list item's continuation). Contained: virtual-text summaries show the anchor's resolved position; source mode verifies. Accepted.

## Unresolved Questions

TODOs for the human reviewer — resolve the matching thread or edit here:

- [x] Rendered (glamour) reading experience: **yes, and make it the default** (review, 2026-08-05)
- [x] `q` verdict prompt: **yes — `q` opens the verdict dialog; `Ctrl+C` is the silent escape** (review, 2026-08-05)
- [x] Virtual-text summaries: **on by default** (review, 2026-08-05)
- [x] `--editor` round-trip: **deprioritized — tool over editor integration** (review, 2026-08-05)
- [x] New comments in rendered mode anchor to the **focused paragraph**; `v` (source mode) for exact lines (decided by default, review round 3)
- [x] In-TUI accept/reject: **queued, applied together at the verdict dialog** (review, 2026-08-05)
