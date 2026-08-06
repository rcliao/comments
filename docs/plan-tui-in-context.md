# Plan: In-Context TUI (v2 compositing, dialogs over the doc, thread display)

**Progress**: Phase 1 done 2026-08-06 — branch `tui-v2-spike` (61e68d2, 4d4b7f1): port was mechanical, all criteria green (suite + teatest integration + lint), compositor proven by test, human pass "identical so far". Discoveries: v2 clips rows at terminal width (title-bar mode indicator already invisible with long paths — fix in Phase 3); frame assertions must target composed content. Phase 2 in flight on `tui-v2-proto`.

## Overview

Convert every screen-swapping action into in-context display: migrate pkg/tui to Bubbletea/Lipgloss v2 for native compositing, verify by prototype before committing (review decision on research-tui-in-context.md:38), let the human pick the thread-display shape by driving three live candidates, then convert all dialogs to layered popups over the visible document.

## Current State

- Nothing overlays the document: thread view fully swaps the screen and nine dialog modes are boxes Placed over blank whitespace (research-tui-in-context.md:18); input dialogs re-print document lines inside themselves to compensate (research-tui-in-context.md:22, the F2 tell at pkg/tui/keys_input.go).
- We are on Bubbletea v1, which has no compositor; Lipgloss v2 (stable Feb 2026) added Layer/Canvas/Compositor with real z-index (research-tui-in-context.md:26).
- Peers keep threads beside visible content and route dialogs through a stack over composited layers (research-tui-in-context.md:30); our mode registry (pkg/tui/registry.go) is the claimed slot-in point — claimed, not verified, which is why Phase 1 is a spike (research-tui-in-context.md:38).

## Desired End State

No action erases the document. Threads open in the shape the human picks from the prototype; dialogs are popups layered over the live doc; Esc unwinds one layer at a time. Verify: every dialog shows the document behind it (automated render assertions), and one full review session runs without a single full-screen swap except the file picker.

## What We're NOT Doing

- No rendered-markdown mode: glamour v2 still has no source-line mapping and nobody in the ecosystem has solved reflow-with-anchors (research-tui-in-context.md:34); building our own renderer is a separate project, explicitly declined in review. In-place span styling stays.
- No v1 string-splicing interim: the review chose prototype-on-v2 over shipping a v1 stopgap twice.
- No mouse-driven UI in this pass (v2 hit-testing enables it later; keyboard stays primary).
- No file-picker rework — it legitimately owns the screen when no document is open.

## Implementation Phases

### Phase 1: v2 migration spike (verify before committing)

On a branch, port pkg/tui to charm.land v2: import paths, `tea.View` struct returns, `KeyPressMsg` interface, styleSet against lipgloss v2. Deliverable is a compiling, test-passing branch plus honest notes on what broke — this verifies the registry-fits-dialog-stack claim before any feature work builds on it.

**Success Criteria**
- automated: `go build ./...` and the full suite green under `-race` on the branch; dispatch tests still cover every mode
- automated: integration tests via teatest (Charm's Bubbletea test harness — decided in review): drive a real program through open → navigate → thread → back, asserting frames, so the port is verified end-to-end rather than only unit-by-unit
- manual: `comments view` runs a normal review session on the branch with no visible regressions (themes, cursor, sidebar, peek)

### Phase 2: thread-display prototype — pick by driving, not debating

On the spike branch, implement all three thread-display candidates behind a debug cycle key: floating overlay near the anchor line, side-panel takeover (sidebar becomes the thread, doc stays), bottom drawer. Same thread content in each; `r` reply flow works in at least one to feel the full loop.

**Success Criteria**
- automated: render tests per shape (thread content present AND document content still present in the same frame)
- manual: the human drives all three on a real doc and picks one; the losers are deleted the same day (no dead prototype code lingers)

### Phase 3: dialog stack — the nine faux modals become popups

Introduce a small dialog stack over the registry: open dialogs intercept keys first, render as layers composited over the live browse/line-select view, Esc pops one layer. Convert all nine (add-comment, reply, resolve, add-suggestion, choose-target, suggestion-type, verdict, help, TOC) plus the ref peek's placement. Delete the contextText compensation blocks from input dialogs — the document behind them IS the context now.

**Success Criteria**
- automated: per-dialog render assertions that document text is visible in the same frame; dispatch/registry tests keep passing; contextText code deleted (grep-clean)
- manual: reply to a thread and queue a suggestion verdict with the doc visibly behind both dialogs; dim-vs-full-brightness decided here by look

### Phase 4: thread display final build + peek polish

Build the chosen thread shape for real: replaces ModeThreadView's screen swap, keeps doc + cursor visible, `r`/`x`/`a` work in place. Fold in the peek discoverability fix from review: surface the multiple-references cycle hint (`f/Tab`) on the line-select hint bar when the cursor line has more than one reference.

**Success Criteria**
- automated: thread-shape render tests (doc + thread same frame, NEW badges and round separators intact); peek hint test
- manual: one full RPI review session end to end without losing document context once; signoff recorded from inside the new verdict popup

## Risks

- **v2 migration surface** — the whole pkg/tui touches new APIs at once. Mitigated: Phase 1 is a spike branch with the full test suite as the safety net; nothing merges until green.
- **Two UIs during transition** — main keeps shipping v1 fixes while the branch lives. Mitigated: phases are days not weeks, and Wave-3's per-mode file split keeps rebases small; accepted.
