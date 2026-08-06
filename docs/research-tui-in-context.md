# Research: In-Context TUI Display (overlays, panels, rendering)

- date: 2026-08-06
- researcher: claude
- topic: why actions screen-swap today, what the ecosystem now supports, how peers keep review in context
- status: complete

## Research Question

Reviewing in `comments view`, most actions (diving into a thread, replying, resolving, even help) jump to a new screen instead of appearing in context, forcing the UI to re-print document context inside dialogs. What are the actual display mechanics today, what does the 2026 Bubbletea ecosystem support for true overlays and panels, and is rendered markdown with line-anchored comments now feasible?

## Summary

The TUI has zero real overlays: one mode fully swaps the screen and nine "modals" center a box over blank whitespace — a v1-era limitation, since Bubbletea v1 has no compositor. That era ended in February: Bubbletea/Lipgloss v2 are stable with a native Layer/Canvas/Compositor API (real z-index, mouse hit-testing), and Charm's own Crush uses a dialog-stack over composited layers in production. Peer review tools converge on persistent panels + transient overlays + threads-as-contextual-panels (octo.nvim keeps the diff visible beside the thread). Rendered markdown with line anchors remains unsolved ecosystem-wide — glamour v2 still exposes no source-line mapping — so in-place span styling stays the right rendering call.

## Findings

### F1 — Today, nothing overlays the document

Only browse/line-select and range selection keep the doc on screen. Thread view is a full swap: the doc is replaced by a full-width thread viewport (pkg/tui/keys_thread.go:23), entered from browse (pkg/tui/keys_browse.go:91) and line-select (pkg/tui/keys_lineselect.go:173). The nine dialog modes build a box and `lipgloss.Place` it over *blank whitespace*, not the live view: add-comment (pkg/tui/keys_input.go:399), reply (pkg/tui/keys_input.go:495), resolve (pkg/tui/keys_input.go:552), add-suggestion (pkg/tui/keys_input.go:619), choose-target (pkg/tui/keys_lineselect.go:511), suggestion-type (pkg/tui/keys_lineselect.go:579), verdict (pkg/tui/keys_verdict.go:56), help (pkg/tui/overlays.go:91), TOC (pkg/tui/overlays.go:202). The new ref peek renders its box on an empty screen with no placement at all (pkg/tui/keys_refpeek.go:182).

### F2 — The tell: dialogs re-print the context they erased

Because the doc vanishes, the input modals rebuild "context" inside themselves — reply and add-comment construct contextText blocks of document lines (pkg/tui/keys_input.go) and the reply modal re-prints recent replies. This is compensation for the missing background, and it duplicates rendering logic per dialog.

### F3 — Lipgloss v2 (stable, Feb 2026) has native compositing

Bubbletea, Lipgloss, and Bubbles shipped stable v2 together on 2026-02-23 (charm.land/blog/v2; current: bubbletea v2.0.8, lipgloss v2.0.5). Lipgloss v2 adds `NewLayer(...).X().Y().Z()`, `NewCanvas(w,h)`, and `NewCompositor(...)` with real z-index and mouse hit-testing (`Hit(x,y)`) — the official close of the old "no z-axis" issue (#65). The community string-splicer (rmhubbert/bubbletea-overlay) now recommends lipgloss v2 compositing over itself. Migration is breaking: `charm.land/*/v2` import paths, `View()` returns a `tea.View` struct, `tea.KeyMsg` becomes an interface (`KeyPressMsg`, `msg.Code`). Input routing above the compositor is still app logic — there is no official dialog-stack or focus-manager bubble.

### F4 — Peer review TUIs: panels + transient overlays, threads beside content

The convergent pattern across lazygit (persistent panels + context-stack popups), magit/transient (temporary keymap popup, buffer stays visible), k9s (drill-down stack + transient command box), and gh-dash (table + toggleable preview pane): persistent panes for orientation, transient overlays for actions, Esc unwinds. octo.nvim — the closest analog — shows threads as a **contextual buffer beside the visible diff** ("hold the cursor on a line with a comment to show a thread buffer"; new comments open in the alternate window), reserving floats for the final submit step only. Crush (Charm's own bubbletea-v2 app) is the reference implementation of the modern shape: chat pane + sidebar + a stack-based `dialog.Overlay` whose open dialog intercepts `Update` first, rendered as lipgloss layers above the main content.

### F5 — Rendered markdown with line anchors is still unsolved everywhere

glamour v2 (v2.0.1) improved wrapping (proper CJK/emoji via lipgloss.Wrap) but its API exposes no source-position mapping — no tool surveyed (glow, frogmouth, slides, inlyne) combines reflowed rendering with per-line interaction. Tools that anchor comments to lines (octo.nvim, GitHub) render the *source* so row↔line stays 1:1. This confirms the parked decision in docs/design-markdown-render.md: in-place span styling (shipped, anchors truthful) remains the approach; full rendered mode would require building the block↔line map ourselves against goldmark internals glamour does not expose.

### F6 — Our registry is well-positioned for a dialog-stack refactor

The mode-descriptor registry (pkg/tui/registry.go) already routes keys/view/viewport per mode through one table. Crush's architecture (dialog stack intercepts first, then focus-based routing) maps cleanly onto it: dialogs become a stack of layered components instead of full modes, while heavyweight surfaces (file picker) stay modes. The audit's nine faux modals are all candidates; thread view is the design decision (overlay vs panel), not a mechanical conversion.

## Code References

- pkg/tui/keys_thread.go:23 — thread view full-screen swap
- pkg/tui/keys_input.go:399, 495, 552, 619 — input dialogs Placed over blank whitespace
- pkg/tui/keys_lineselect.go:511, 579 · pkg/tui/keys_verdict.go:56 · pkg/tui/overlays.go:91, 202 — same pattern
- pkg/tui/keys_refpeek.go:182 — peek box, unplaced
- pkg/tui/registry.go — mode table a dialog stack would slot into
- docs/design-markdown-render.md — the parked B5 rendered-mode decision this research re-confirms
- External: charm.land/blog/v2 · lipgloss v2 Layer/Canvas/Compositor · crush dialog-stack architecture · octo.nvim thread-buffer model

## Open Questions

- (blocking) Foundation: migrate pkg/tui to Bubbletea/Lipgloss v2 first (breaking API sweep across the whole package, but native compositing + the maintained path), or prototype overlays on v1 with string-splicing and migrate later? v2-first is cleaner; v1-first ships something visible sooner.
- (blocking) Thread display target: true overlay floating near the anchor line (octo's hover-thread feel), a side-panel takeover (sidebar becomes the thread, doc stays), or a bottom drawer? This is the daily-feel decision the plan hangs on.
- (non-blocking) Should the doc dim under overlays (Crush-style) or stay full-brightness? Cosmetic, decide at implementation.
