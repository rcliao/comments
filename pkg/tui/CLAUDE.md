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
- `dialogs.go` — dialog composition over the live view (`baseView`, `dialogOver`)
- `keys_browse.go`, `keys_lineselect.go`, `keys_input.go`, `keys_thread.go`, `keys_threadpanel.go`, `keys_verdict.go`, `keys_refpeek.go`, `keys_filepicker.go`, `overlays.go` — per-mode key handlers + views
- `rendering.go` — pure render helpers (document, sidebar, thread, markdown spans)
- `styles.go` / `theme.go` — styleSet construction and theme registry

## Dialog composition (popups over the live view)

No dialog erases the document. Every dialog mode (add-comment, resolve,
add-suggestion, choose-target, suggestion-type, verdict, help, TOC, ref peek)
renders its box and composites it over the live screen with the two helpers in
`dialogs.go`. **Reply is the exception**: it docks inside the thread panel
(see below) instead of floating, because the reply belongs beside the thread
it answers, not over the middle of the document.

- `baseView()` — the screen the dialog interrupts: the thread panel view
  (doc + panel) while a thread is open (`selectedThread != nil`), otherwise
  `viewBrowse()`. The title bar keeps announcing the dialog's mode because
  `viewBrowse` renders `m.mode`.
- `dialogOver(base, dialog)` — lipgloss v2 compositor, dialog layer centered
  over the base at Z(1). The base renders at full brightness; dimming it later
  is one style call on the base layer here. With no laid-out screen
  (`!m.ready`) it returns the dialog alone.

There is deliberately NO dialog-stack machine: dialogs never nest more than
one deep over a base view. Modes stay modes (registry entries, key handlers),
only their VIEW is a composition; Esc pops the one layer by returning to the
mode underneath. Don't re-print document or thread lines inside a dialog box
("context" blocks) — the live view behind the popup is the context.

## Thread panel (the thread display)

Opening a thread (`enter` from browse, `r` from line-select) never swaps the
screen: `ModeThreadView` composites the thread over the live browse view as a
side-panel takeover (`keys_threadpanel.go`) — the panel replaces the
comment-sidebar region (right of the doc pane), full content height; when the
sidebar is hidden it takes the right 40%. `applyThreadPanel()` sizes/fills the
threadViewport (call it on open/resize); `refreshThreadPane()` re-renders
content preserving scroll (reply added, decision queued). The panel chrome
(`renderThreadPanelBox`) draws the thread's ONE header (icon + section path +
line); `renderThreadWidth` renders only the thread body — no location line, no
document-context box.

**The reply composer docks in the panel and grows as you type.** `ModeReply`
is not a floating dialog: `renderThreadPanelBox` is mode-aware and, while
composing, draws a separator + the textarea under the thread and swaps the
panel hint line for `Ctrl+S: save reply • Esc: cancel` (the panel keys it
replaces are dead anyway). No second box, no repeated title — the panel header
is still the thread's one header.

The composer is its own textarea (`replyInput`, built by `newReplyTextarea`),
NOT the shared `commentInput` — it mutates four fields the add-comment dialog
would have to get back, and save/restore of four fields is a standing bug
factory. `proposedTextInput` and `verdictNote` are dedicated for the same
reason.

Growth is bubbles' `DynamicHeight` (recalculated inside `textarea.Update`,
soft wraps included) between `MinHeight = composerMinRows` and a `MaxHeight`
cap that `applyComposerLayout` derives from the panel so
`composerMinThreadRows` of thread always stay visible. **`MaxContentHeight`
must be set explicitly** (`composerMaxContentRows`): at 0 the textarea falls
back to blocking input at `MaxHeight` logical lines, which would silently make
any reply longer than the visible composer impossible to type.

`composerRows()` derives the space from the textarea and `threadPaneRows()`
takes it out of the thread viewport, so the panel never grows past its layout.
`syncComposerLayout()` re-fits the pane and must run on EVERY path that feeds
the textarea a message: the key-handler tail AND `updateReplyInput` — a
bracketed paste is a non-key message and can add many rows at once. It
restores `GotoBottom()` when the pane was at the bottom, or a shrinking pane
would cut off the newest reply. Note the box clips at `MaxHeight`, so a missed
resync silently swallows thread rows instead of overflowing the screen —
assert on `threadViewport.Height()`, not on rendered height.

Call `applyComposerLayout()` on reply open and on resize; on exit `Reset()`
already shrinks the textarea, so `closeComposer()` only hands the rows back
(Ctrl+S then goes through `applyThreadPanel()` so the view lands on the reply
just posted).

**Box widths**: lipgloss v2 `Width(n)` counts the border, so text inside the
panel must fit `lay.w - 4`, not `lay.w - 2`. Truncating to the wrong width
wraps the last characters onto their own line (this bit the header with long
section paths).

## Focus rules while the panel is open

The screen still reads as browse, so keys split three ways
(`handleThreadViewKeys`):

- **Thread actions stay on the panel**: `j/k` scroll the THREAD (not the
  document), `r` docks the reply composer inside the panel, `a`/`x` queue
  suggestion decisions (or `x` resolves), `Esc` closes the panel back to
  where it was opened (browse or line-select, cursor intact).
- **Browse-shaped keys fall through with browse semantics** instead of dying:
  `c` closes the panel and starts the comment flow at the cursor line, `q`
  opens the verdict dialog (Esc restores the panel; `q` does NOT quit from
  the panel), `?` opens help over the doc+panel view.
- **Everything else is ignored** (notably `S`/`L`/`t` — close the panel
  first). If you add a fall-through key, it must behave exactly as it does in
  browse and must close or preserve the panel deliberately.

## Verdict and signoff parity

`q` opens the verdict dialog; `a`/`c`/`r` apply the queued suggestion decisions
and write a `ReviewRecord` through `comment.AddReviewRecord` — the SAME record
`comments signoff` writes, note included. `r` records decision `commented`
(reply-pass: answered threads, turn handed back, gate untouched, exit 0).
`recordVerdict` calls `refreshDocFromDisk()` FIRST — a session open while an
agent edits must not sign off from stale memory — and only the suggestion
accepts write the markdown (`SaveDocumentContent`); `SaveToSidecar` never
touches the .md. Every TUI mutation path (reply, resolve, add, suggest,
verdict) refreshes from disk before mutating: last-writer-wins per action,
not per session. `n` opens `ModeVerdictNote`, a
separate mode so `a`/`c` are plain letters while typing; Esc/Ctrl+S returns to
the dialog keeping the text, and `recordVerdict` trims it into
`ReviewRecord.Note`. Keep the two writers producing identical records: agents
waiting on `request_review`, `check_review` or `watch --until signoff` key on
the record, not on who wrote it. The note deliberately survives `q` → Esc →
`q` within a session (you drafted it, going back to check a thread shouldn't
discard it) — that is intended, not a leak.

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
