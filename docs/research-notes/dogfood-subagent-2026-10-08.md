# Dogfood: Claude as Eric, a subagent as the agent (2026-10-08)

One small real fix, run end to end as a living doc: on a living doc picked from the file picker, `q` opened the verdict dialog.
Claude played Eric in chat. A subagent ("builder") in its own worktree played the agent and followed the branch skill.
Result: commit `922ee32` on `worktree-agent-a10bf66dff87efe80`. Its doc is `docs/artifacts/living/picker-living-quit.md` on that branch.

## Timeline

1. Eric's ask (one chat line). The agent shaped in about 2 minutes: 8 cited Findings, 3 slices, one open decision (D1), and it validated clean. It asked for a go.
2. Eric read Now, then D1, then Plan, replied "go, yes D1", and added scope in chat: the picker also forgets your reading position.
3. The agent built in about 5 minutes. Decisions got a dated line ("chat: go, D1 accepted, reading position folded in as slice 4"), Plan got slice 4, and phase went to done. Three tests fail on the old code. `./scripts/ci.sh` passed, and a fresh review found no blockers.
4. Eric ran the manual check in tmux: picker, then living/, then the doc. The rail showed `DONE` with Now's first line, and the hint read `q: quit`. `q` exited 0 with no dialog.

## What worked

- Now's first line was enough to know the state at each step, with no need to read further.
- The scope change made in chat reached Decisions and Plan in the same turn, unprompted.
- One open decision, named, with the rejected option and its reason: answering took one word.

## Friction, and what to change

| # | Seen | Change |
|---|---|---|
| 1 | The chat ask repeated the whole Plan, so the human read it twice. | Skill: the ask is the decisions plus a pointer to the doc ("Go? D1: q quits (recommended). Plan: <doc>"), not a copy of Plan. |
| 2 | The agent said "I can't drive the TUI from here" and handed off the manual check, but tmux was on PATH; the check took 1 minute with `tmux new-session -d` + `send-keys` + `capture-pane`. | Skill: run TUI checks yourself in a detached tmux session before you hand them to the human; hand off only what really needs eyes. |
| 3 | `long_sentence` failed the human's own quoted words in Why. A blockquote is exempt by design (`pkg/comment/style.go:110`), but nothing says so. | Living template, Why criterion: "quote their words in a `>` blockquote". |
| 4 | `comments new` refreshed `plans/index.md` with two plans committed without index entries. The stray change had to be kept out of the commit by hand. | Not a bug in `new`: the committed index was stale. Commit the refreshed indexes once. The main checkout has the same diff. |
| 5 | The worktree was cut from `main`, not the feature branch, so `--template living` did not exist; the agent fast-forwarded its branch unasked. | Claude Code's worktree base, not comments. Note it in the dogfood prompt next time. |
| 6 | Done step: the skill says nothing about Checks the agent cannot run. | Covered by 2; what is left goes on Now's "Needs you" line, which the agent did on its own. |

Done in PR #28 (plugin 3.1.0): 1 and 2 in the skill's flow steps 2 and 5, and 3 in the living template's Why criterion.

Not exercised: `comments view` live reload during the build (the human only looked at the end), the mod's drift count (subagents do not get the status line), and threads.
