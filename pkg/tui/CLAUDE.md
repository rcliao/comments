# TUI patterns and gotchas (Bubbletea)

The TUI follows The Elm Architecture: all state in `Model`, `Update(msg) → (Model, Cmd)`, `View() → string`.

## Mode state machine

Mode transitions happen in **key handlers**, not in `Update()`:

```go
// Good: State transition in key handler
case "c":
    m.mode = ModeLineSelect
    m.refreshCursorView()
    return m, nil

// Bad: Don't put transitions in updateByMode
```

**Adding a new view mode** — modes register in ONE place, the `modeRegistry`
table in `registry.go`; `handleKeyPress()`, `View()`, and `updateByMode()` all
dispatch through it:

1. Add to `ViewMode` enum in `modes.go` (+ `String()` case)
2. Write `handle<Mode>Keys()` and `view<Mode>()` in the mode's `keys_*.go` file (or a new one)
3. Add ONE `modeRegistry` entry wiring `handleKeys`, `view`, and (only if the mode's component consumes non-key messages — mouse wheel, blink ticks) an `updateViewport` func

An unregistered mode has dead keys and renders "Unknown mode";
`TestModeRegistryCoversEveryMode` fails if the enum and registry drift (keep
`allModes` in `dispatch_test.go` in sync too).

## File layout

- `model.go` — Model state, constructors, Update/View, resize, registry dispatch
- `registry.go` — the mode-descriptor table (single registration point)
- `keys_browse.go`, `keys_lineselect.go`, `keys_input.go`, `keys_thread.go`, `keys_verdict.go`, `keys_filepicker.go`, `overlays.go` — per-mode key handlers + views
- `rendering.go` — pure render helpers (document, sidebar, thread, markdown spans)
- `styles.go` / `theme.go` — styleSet construction and theme registry

## Styles and themes

Styles live on `m.styles` (a `*styleSet` built from a `Theme` at model
construction), NOT in package globals. `SetTheme(name)` only sets the
mutex-guarded startup theme for models constructed afterwards — call it before
`NewModel`/`NewModelWithFile`. Rendering never reads mutable package state, so
theme switching cannot race renders (`-race`/`t.Parallel` safe). Pure render
helpers that need styling are methods on `*styleSet` (e.g.
`st.styleMarkdownLine`, `st.lineMarker`, `st.renderTOC`).

## Viewports

Initialize viewports only when `ready = false`, inside `handleResize()`:

```go
if !m.ready {
    m.documentViewport = viewport.New(docWidth, m.height-2)
    m.documentViewport.SetContent(m.renderDocument())
    m.ready = true
}
```

**Common bug**: forgetting to call `handleResize()` after loading a file from the file picker — always check dimensions exist before initializing viewports.

The model keeps multiple viewports that must stay in sync (`documentViewport` ~60% width, `commentViewport` ~40%, `threadViewport` full-width, `commentInput` textarea). On a mode change, update the **active** viewport's content, not all of them.

After any cursor move in line-select mode, call `m.refreshCursorView()` — it
re-renders the cursor view, keeps the cursor line visible, and syncs the
sidebar. Don't hand-roll that triple.

## Rendering

All rendering functions are **pure functions** on the Model (`renderDocument`, `renderComments`, `renderThread`): Model state in, strings out, no side effects, no Model mutation.

Both document renders go through `renderDocumentView(withCursor)` and wrap at
`docWrapWidth()` (viewport width − 12, the cursor-layout gutter). One shared
width means wrap points — and therefore scroll rows — are identical in browse
and line-select, so mode switches never reflow the document under a saved
scroll offset. Don't reintroduce per-mode wrap widths.

## Debugging

If the TUI hangs or doesn't respond:

- Check the mode has a `modeRegistry` entry (and an `updateViewport` if its component needs non-key messages)
- Verify viewport init happens in `handleResize()` when `ready = false`
- Look for blocking operations in `Update()` — all I/O should be async
