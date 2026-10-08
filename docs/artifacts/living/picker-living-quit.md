---
comments:
    phase: done
    template: living
status: draft
title: Picker Living Quit
type: Living
---

# Picker Living Quit

## Now

- **State:** done. A picked living doc's `q` now quits with no verdict, its hint says "q: quit", and a picked file resumes where you left it.
- **Evidence:** the three new picker tests fail on the old code and pass now; `./scripts/ci.sh` printed `All CI gates passed.`; a fresh review of the diff against Plan found no blockers.
- **Needs you:** the manual check below. I can't drive the TUI from here.

## Why

In your words:

> when I open `comments view` with no file and pick a living doc from the picker, q still asks me for a verdict. a living doc has no verdict. fix it

## Findings

- F1. Browse `q` takes the living path only when `m.startedWithFile` is true; otherwise it opens the verdict dialog. pkg/tui/keys_browse.go:21 pkg/tui/keys_browse.go:28
- F2. The thread panel's `q` has the same gate. pkg/tui/keys_thread.go:52
- F3. `startedWithFile` is false for a picker session and true only for `comments view <file>`. pkg/tui/model.go:172 pkg/tui/model.go:222
- F4. Picking a file goes through `loadFile`, which already rebuilds the changed-line tint from the stored baseline, so the seen baseline works for picked docs too. pkg/tui/keys_filepicker.go:22 pkg/tui/model.go:491
- F5. Ctrl+C already calls `markSeen` with no gate, so only `q` is wrong. pkg/tui/model.go:267
- F6. The browse hint says "q: back" in a picker session, which would be false once `q` quits a living doc. The living rail already says "q close". pkg/tui/keys_browse.go:259 pkg/tui/rail.go:134
- F7. A picked review doc also quits after its verdict, so no picker session returns to the picker today. pkg/tui/keys_verdict.go:86
- F8. `loadFile` never reads the saved view state; only `NewModelWithFile` does, so a picked file opens at the top. pkg/tui/model.go:239 pkg/tui/model.go:464
- F9. The existing living test opens the doc only through `NewModelWithFile`, so the picker path has no test. pkg/tui/living_test.go:42

## Plan

**Slice 1: q on a living doc never asks for a verdict** (F1, F2, F3, F5)

- Remove `&& m.startedWithFile` from both living checks. `q` then calls `markSeen`, saves view state, and quits, however the doc was opened.

**Slice 2: the hint tells the truth** (F6)

- In browse, a living doc's hint says "q: quit", even in a picker session.

**Slice 3: test the picker path** (F4, F9)

- Open a living doc through `NewModel` and `loadFile`. Press `q` in browse and in the thread panel. Assert there is no verdict mode, a quit command, no review record, and a seen baseline.

**Slice 4: a picked file resumes where you left it** (F8)

- Move the restore into one `restoreViewState` helper and call it from `loadFile` as well as `NewModelWithFile`.

Not doing: changing what `q` does on review docs, or making picker sessions return to the picker (F7).

## Decisions

- 2026-10-08 (chat): go, with D1 accepted and the picker's reading position folded in as slice 4 because it is small.
- D1 (agent, accepted in chat): `q` on a picked living doc quits, the same as when it is named on the command line. Rejected: return to the picker, because no picker session does that today (F7) and it would add a second meaning of `q` for one case.

## Explanation

The living-doc `q` was guarded by `startedWithFile`, which only `comments view <file>` sets. Both `q` handlers now check only that the doc is living, so a picked doc records the seen baseline and quits. The browse hint follows the same rule.

`restoreViewState` is the one place that reads the saved position. `NewModelWithFile` and `loadFile` both call it before layout, and `handleResize` applies the scroll offset. Left alone: a picker session still exits instead of returning to the picker (D1).

## Checks

- `go test ./pkg/tui/ -run Living` passes, including the new picker test.
- `./scripts/ci.sh` prints `All CI gates passed.`
- Manual: `./comments view`, pick `docs/artifacts/living/picker-living-quit.md`, press `q`; it quits with no verdict dialog.
