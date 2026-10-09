---
comments:
    phase: done
    template: living
status: draft
title: Hooks Check Lies
type: Living
---

# Hooks Check Lies

## Now

- **Done:** `ci.sh` was the liar; it now asks git where hooks run from, so your absolute `core.hooksPath` reads as wired. Four smoke cases guard it.
- **Evidence:** `./scripts/check-hooks.sh` here: exit 0, no note. `./scripts/ci.sh`: `✓ hooks check follows where git runs hooks from`, `All CI gates passed.`, no note. Fresh diff review: no blockers.
- **Needs you:** nothing. Committed on the worktree branch, not pushed.

## Why

> ./scripts/ci.sh keeps telling me my git hooks aren't wired up, but my pre-push hook clearly ran on my last push. one of them is lying, figure out which and fix it.

## Findings

- **F1.** `ci.sh` decided "wired up" by comparing the raw config string to the literal `.githooks` (scripts/ci.sh:69, before this change).
- **F2.** Your clone sets `core.hooksPath` to the absolute `/Users/rcliao/src/rcliao/comments/.githooks`, read from `.git/config` (docs/research-notes/hooks-check-lies-notes.md:7).
- **F3.** That absolute path holds executable `pre-commit` and `pre-push` (docs/research-notes/hooks-check-lies-notes.md:9), and `pre-push` ends in `exec ./scripts/ci.sh` (.githooks/pre-push:34). So the hook does run the suite; the hook is telling the truth.
- **F4.** `git rev-parse --git-path hooks` returns the directory git will actually run hooks from: `/abs/x` for an absolute setting, `../.githooks` for a relative one asked from a subdirectory (docs/research-notes/hooks-check-lies-notes.md:14, docs/research-notes/hooks-check-lies-notes.md:16).
- **F5.** Because your path is absolute, this worktree's effective hooks dir is the main clone's `.githooks`, not its own copy (docs/research-notes/hooks-check-lies-notes.md:12). A relative `.githooks` would resolve to each worktree's own copy instead (fresh reviewer's scratch-repo probe, git 2.50.1).
- **F6.** Nothing tested the check; the only references were the check itself and setup docs (./CLAUDE.md:26, ./CLAUDE.md:32, .githooks/pre-push:6).

## Plan

1. **Resolve, don't compare** (F1, F4). New `scripts/check-hooks.sh` checks the repo it is run from. Unset or empty `core.hooksPath` is not wired (D3). Otherwise resolve `git rev-parse --git-path hooks` from that cwd; wired means its `pre-commit` and `pre-push` are both executable. Exit 0 wired, 1 not wired with the note.
2. **Wire it** (F1). `ci.sh` calls `./scripts/check-hooks.sh || true` where the inline check was; still informational.
3. **Regression check** (F6). `scripts/smoke-test.sh` runs it in a scratch repo under `workdir` for: unset, relative, absolute, non-executable pre-push.

Not doing: an info line naming which copy runs (dropped per your go), more smoke cases, changing your git config or the hooks. CLAUDE.md's setup lines stay accurate as written.

## Decisions

- 2026-10-08, chat: go, but smaller. D1, D2, D3 yes; drop the which-copy-runs slice; smoke cases limited to absolute, relative, unset, non-executable.
- **D1.** An absolute path to another checkout's `.githooks` counts as wired, with no extra line (chat).
- **D2.** Separate `scripts/check-hooks.sh`, exercised by the smoke test (chat).
- **D3.** Unset `core.hooksPath` is not wired, even if `.git/hooks` has hooks (chat).
- Agent: smoke cases run with `GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1`, so a developer's global `core.hooksPath` cannot break the unset case (diff reviewer's note).
- Shipping, the pre-push hook failed the unset case: git exports `GIT_DIR` to hooks, so the scratch repo read (and would have written) the real repo's config. The smoke cases now unset `git rev-parse --local-env-vars` first. (Claude, while shipping)
- Rejected: normalizing the string (strip `$repo_root/`) — still misses `~` paths, symlinks, and the exec-bit case.
- Rejected: `GIT_CONFIG_COUNT` overrides in Checks — Claude sessions already export `GIT_CONFIG_COUNT=2` for `safe.directory`, and index 0 would clobber it (fresh reviewer).

## Explanation

`scripts/check-hooks.sh` reports on the repo you run it from. Unset `core.hooksPath` is not wired. Otherwise git resolves the hooks dir (`git rev-parse --git-path hooks`), and wired means `pre-commit` and `pre-push` there are executable. `ci.sh` calls it after the gates as a note, never a failure. Nothing is left open.

## Checks

- `./scripts/check-hooks.sh` in this worktree (absolute path, Eric's layout): exit 0, no note.
- `./scripts/smoke-test.sh`: prints `✓ hooks check follows where git runs hooks from`.
- The old line-69 logic, run against the absolute-path case, reports not wired (the bug, reproduced).
- `./scripts/ci.sh` ends with `All CI gates passed.` and no "not wired up" note.
- A fresh-context review of the diff against Plan finds no blockers.
