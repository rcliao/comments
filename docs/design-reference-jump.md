# Reference Jump: Citations Between Docs (RPI Support)

## Problem

In the RPI flow (Research → Plan → Implement), the plan is the review unit — but its claims trace back to the research doc via citations like `research/2026-08-06-anchors.md:42` or `[findings](./research.md#cascade)`. Reviewing a plan in `comments view`, those citations are dead text:

- Checking a citation means leaving the TUI, opening the file, finding the line — the review flow breaks every time.
- So citations don't get checked, and an agent-written plan can cite research that doesn't say what the plan claims. Citation verification is exactly the reviewer's job, and today the tool makes it the most expensive action in the loop.

This is the gap between us and the RPI workflows we benchmarked: their plans mandate `file:line` references, but no terminal review tool makes them followable.

## Goals / Non-Goals

**Goals:**

- **G1** — verify a citation without losing review position: peek at the cited finding, come back to the same cursor line
- **G2** — references are first-class in the doc view: visibly styled, jumpable, resolvable
- **G3** — read-only and small: no editing, no buffer management; ship in verified steps

**Non-goals:** web URLs; editing the target file; cross-file comment threads; a backlinks index (research doc showing which plans cite it) — journal [idea] if wanted later.

## Proposed Design

Three pieces: detection, display, interaction.

**1. Detection** (new `pkg/markdown` pass, pure functions): recognize two reference forms per line —

- Markdown links to local files: `[text](path.md)`, `[text](path.md#heading)`
- Bare file:line tokens: `research.md:42`, `pkg/tui/model.go:123` — must look like a path (contains `/` or ends in a known text extension) to avoid false positives on ratios like `3:5`

Resolution mirrors template discovery: try doc-relative first, then walk up toward the repo root. Non-markdown targets (Go files, YAML) are legal — rendered as plain text.

**2. Display**: recognized, resolvable references get link styling (theme link color + underline) through the existing span-styling engine. Unresolvable ones stay plain — no styling lies.

**3. Interaction** (line-select mode; decided in review — peek first, editor for depth):

```
  plan.md (cursor on line with citation)
      │  f                                    ┌─────────────────────┐
      ▼                                       │ PEEK: research.md   │
  ┌─ peek overlay ── read-only excerpt,       │  40│ ...            │
  │  ~20 lines centered on :line/#heading ────│ ►42│ finding text   │
  │                                           │  44│ ...            │
  │  esc = close                              └─────────────────────┘
  │  enter = open $EDITOR at file:line (TUI suspends, resumes on quit)
  ▼
  back in plan.md, same cursor line, review continues
```

- `f` on a line with one reference → peek overlay. Multiple references on the line → repeated `f` cycles them (same convention as Tab for stacked threads).
- `enter` inside the peek → suspend the TUI and open `$EDITOR` at the cited location (`nvim +42 research.md` style, via tea.ExecProcess); quitting the editor resumes the review exactly where it was. The whole-file case belongs to the editor — no in-TUI doc switching, no jump stack.
- Unresolvable reference → `f` still opens the peek, showing the resolution error (decided in review: syntax stays permissive, failures surface at peek time).

Phasing: step 1 detection + styling (pure, testable), step 2 peek, step 3 editor handoff. Each lands with tests before the next.

## Options Considered

### Option 1: Peek overlay + $EDITOR handoff (chosen in review)

- Good, because peek answers the actual review question ("does the research say that?") without any context switch
- Good, because the deep-reading case gets full editor power (search, syntax, muscle memory) for ~30 lines of tea.ExecProcess
- Good, because dropping the in-TUI jump kills its whole maintenance surface: no jump stack, no breadcrumb, no second-doc state
- Bad, because the editor detour suspends the TUI (sidebar and threads vanish until quit) — accepted: peek covers the common case, so the detour is rare

### Option 2: In-TUI full jump with back-stack (original proposal)

- Good, because the target doc's comment threads come along free (it's just comments view pointed elsewhere)
- Good, because review position is restored mechanically by the stack
- Bad, because it duplicates what the editor already does well, behind a second navigation model to learn and maintain
- Rejected in review: peek + editor covers both depths with less machinery; a cited doc's own review state is a separate `comments view` away when actually needed

### Option 3: Splits / multi-buffer TUI (plan and research side by side)

- Good, because both docs stay visible — no flipping at all
- Bad, because split layout, focus management, and per-pane state are the most expensive TUI surface we could build, against one use case
- Bad, because at common terminal widths two docs + sidebar don't fit anyway
- Rejected: the peek overlay captures most of the value at a fraction of the surface

## Risks

- **False-positive references** (a `word:12` that isn't a path) styled as links would erode trust in the styling. Mitigated: the token must parse as `path.ext:line` (extension before the colon) or contain `/`; only what resolves on disk gets link styling. Resolution runs once at doc load into a cached ref map — rendering stays pure (decided in review); styling can go stale if a target file appears mid-session, accepted.
- **Tiny terminals** — peek overlay on small windows. Mitigated: peek shrinks with the help-overlay sizing conventions and keeps working — `enter` to the editor is always available; peek never silently escalates to a jump (decided in review).

## Definition of Done

- automated: reference-parser unit tests — md links with/without `#heading`, `path.ext:line` parsed as a unit (the `research.md:42` form), `/`-paths, resolution order (doc-relative then walk-up), false-positive rejection (`3:5`, `word:12`), unresolvable stays unstyled
- automated: cached ref map built at load (no filesystem access in render); peek render tests at normal and tiny sizes, including the unresolvable-reference error state; full suite green under `-race`
- manual: in a plan doc citing `research.md:N`, `f` peeks the finding at line N, `enter` opens `$EDITOR` at that line, quitting resumes the review at the same cursor line
- manual: one real RPI pass — review an actual plan against its research doc using peek + editor only
- Out of scope for this handoff: URL handling, backlinks, editing from the peek, splits, in-TUI doc switching

## Unresolved Questions

- (non-blocking) Should references inside *comment text* (not just doc lines) also be followable? Decide during implementation of step 3.
