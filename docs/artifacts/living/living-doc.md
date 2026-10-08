---
comments:
    template: living
status: draft
title: Living Doc
type: Living
---

# Living Doc

One doc per piece of work, kept current by the agent while it builds. You read it whenever you want the state; chat is where you steer. The doc and the build are two tracks that move together: every turn that changes the code changes the doc, and you can see when the doc has fallen behind. This file is the first living doc, and the trial of the idea.

## Now

- **State:** built, reviewed and committed on `feat/one-brief`. A fresh review found three holes (a sticky doc that read "current" after a restart, no reload without a sidecar, a stale thread panel); all fixed and tested.
- **Next:** you run the next real task this way (the manual check).
- **Needs you:** try it. `comments view docs/artifacts/living/living-doc.md` in a pane; `/review-doc --park` to drop the parked close-on-approve lock.

## Why

The brief flow made the doc a contract and threads the conversation, so effort went into closing threads and keeping approvals fresh (settle-on-approve, close-on-approve). What works in practice is one doc for plan, recap, status and explanation that the agent keeps updated, with chat as the interface and the doc improving as we go.

## Plan

**Slice 1: the doc and the read path (pkg/comment, pkg/tui)**

- A `living` template: Now, Why, Plan, Decisions, Explanation, Checks. Word caps keep it readable; Now is capped hardest. No `zone: human`, no approval.
- Changed since you last looked: closing `comments view` records a *seen* baseline, so the tint shows what moved since your last read, not since your last verdict. Verdict baselines stay as they are.
- `comments view` reloads when the file changes on disk, so a pane left open beside the session stays live.

**Slice 2: keep the two tracks together (mods/comments-review)**

- No code lock for living docs; the contract gate stays for briefs and plans.
- Writing a `template: living` doc makes it the session's living doc. After compaction the mod re-injects "Living doc: X, N code edits since it last changed" instead of the contract note.
- `/review-doc --park` (you only) sets a parked brief aside so its lock lifts. A living doc is never locked.
- Drift: the mod counts code edits since the doc last changed and shows it in the status line ("doc: X · 6 code edits since the doc"). A nudge, not a gate.

**Not doing**

- No removal of briefs, plans, verdicts or gates. They stay for work where you want me locked until you agree. Tear-down waits until this has run a while.
- No artifact (published page) view yet; markdown in the repo is the source.

## Decisions

- Markdown in the repo, not a published artifact: durable, diffable, and the changed-since tint works on it. (chat, 2026-10-08)
- `close-on-approve` is parked: it fixes a cost this flow removes. (chat, 2026-10-08)
- No `/doc` command: writing the doc is enough to make it the session's doc; `--park` covers the lock it was meant to lift. (agent)
- The living doc and its drift count persist per repo across restarts, so the next session picks up where this one stopped; `/review-doc --done` ends it. (agent, after review)
- `./scripts/ci.sh` now runs the mod's tests when `claude` is on PATH. (agent, after review)
- Live reload applies to every doc, not only living ones, but only in browse and the thread panel. (agent)
- "Parallel doc and build track" means the doc updates in the same turn as the code, and drift between them is visible. (chat, 2026-10-08; my reading)

## Explanation

**How a turn works.** You ask in chat. I do the work, then edit this doc in the same turn: Now always, Plan or Decisions when something changed, and Explanation when how-it-works changed. A decision reached in chat gets a line in Decisions with where it came from.

**How you read it.** Open `comments view docs/artifacts/living/living-doc.md` in a pane and leave it. Lines that changed since you last looked are tinted. Comments still work for pointing at an exact line, but they are optional.

**How drift is counted.** The mod watches Edit and Write. A write to the living doc resets the count; any other file edit adds one. The count is a nudge in the status line, never a lock. There is no turn-end hook, so it cannot stop a turn that leaves the doc behind.

**What keeps it honest.** Now is short and overwritten, not appended; history is in git. Why stays in your words; if I change it, the tint shows you.

## Checks

- Slice 1: `go test ./pkg/comment ./pkg/tui` covers the template, the seen baseline (view → edit → reopen tints only the edit), and live reload.
- Slice 2: the mod tests cover no lock on a living doc, the re-injected note, and the drift count.
- `./scripts/ci.sh` prints `All CI gates passed.`
- A fresh-context reviewer checks the diff against this Plan.
- Manual: you run the next real task this way and say whether you opened the doc, whether it was accurate, and whether you needed a thread.
