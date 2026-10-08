---
name: review-comments
description: Run a piece of work from one living doc with the comments CLI/MCP — research it into a cited brief, build on the human's go in chat, keep the doc's Now and Decisions current every turn, and finish with the doc as the recap. Also gated briefs, and processing human review threads on any managed Markdown doc.
---

# Review Comments Workflow

`comments` keeps review threads beside a Markdown file (a `.comments.json`
sidecar) and records the human's verdict. You work through threads **one at a
time** and never decide for the human.

## Commands — one per purpose

| You want to | Use |
|---|---|
| start a piece of work | `new <slug> --template living` (a gated brief: `--template brief`) |
| load the writing brief and related docs | `context <doc> --for drafting` |
| check your draft | `validate <doc>` |
| annotate | `add` — flags for one, `--json` for many; `--anchor "quoted line"` |
| record a decision you made alone | `add --pick <option>` — never blocking; settled at the end |
| wait for the human | `watch <doc> --until signoff --since <hand-off time>` |
| see what needs you | `inbox <doc> --json` — the only read while iterating |
| look up one thread, or resolved ones | `get` |
| respond | `reply`, with `--resolve` once a fix is applied and explained |
| propose an edit to a human section | `suggest` |
| fix anchors after your edits | `reanchor` |
| check the contract while building | `context <doc> --for implementation` (approval freshness) |

**Decisions are the human's, and you have no command for them.** Verdicts,
accepting or rejecting suggestions, and resolving threads in `zone: human`
sections happen only in `comments view` or `comments serve`. A refused
`--resolve` posts nothing: reply without it. Never set `COMMENTS_ACTOR`.

## The flow: one doc, research to recap

One living doc carries the work from start to finish. The human reads it
whenever they want the state and steers in chat; nothing is handed off or
approved. `comments.phase` in its frontmatter says where it is.

1. **Shaping.** `comments new <slug> --template living` starts it in phase
   `shaping`; add a `## Findings` section. Research first: numbered Findings (F1, F2), one claim each, every one with
   its own `file:line` or `thread:` citation that you have opened, not
   recalled. Put sources you read outside the repo in a research note and cite
   its line. Then Why in the human's words, Plan in slices that cite the
   findings they rest on, Decisions with rejected options, and Checks as
   commands. `comments validate` until clean; it fails a finding without a
   citation.
2. **Fresh review, then ask once.** A fresh-context reviewer reads the doc
   against the code; fold in what it finds. Then ask the human for a go,
   naming the open Decisions with your recommendation for each. The ask is
   those decisions and the doc's path, not a copy of Plan: they read the doc.
3. **Go.** A go in chat starts the build. Record it as a dated line in
   Decisions and set phase `building`.
4. **Building.** Every turn that changes code changes the doc in the same
   turn. Now always: overwrite it, never append. Its first line is the doc's
   recap: one line, under 400 characters, readable on its own. Then a
   Decisions line for each decision made in chat or alone, saying where it
   came from, and Plan or Explanation when they change. The comments-review
   mod shows how many code edits the doc is behind; bring it back to current
   before you end a turn.
5. **Done.** Run every check and a fresh review of the diff against Plan.
   Run a manual TUI check yourself before handing it over: `tmux new-session
   -d`, then `send-keys` and `capture-pane -p`. Only what really needs the
   human's eyes goes on Now's "Needs you" line. Set
   phase `done`. Now holds the result and its evidence, and Explanation
   becomes the recap: what was built, how it works, what is left. The doc
   stays as the record.

The human reads it live: `comments view <doc>` reloads as you write, its rail
shows the phase and Now, and `q` marks what they saw, so the next open tints
only what changed since. Threads still work for pointing at a line; answer
them in the doc and in chat.

## Gated briefs (when the human wants edits locked until they approve)

One brief per piece of work, read in tiers. The human owns everything but How.

| Section | Read | Owner | Holds |
|---|---|---|---|
| Why | 1 min | human | the problem and the decision |
| What | 3 min | human | outcome, not doing, invariants, stop-and-ask triggers, rejected options |
| Shape | optional | human | data flow, data model, interfaces — only when the work changes them |
| Checks | 5 min | human | each check is a command with its expected result; one per invariant written to break it |
| How | 10 min | you | premises, changes, steps, status — yours to rewrite |

The approval covers the frontmatter, the `#`/`##` outline, and the whole of
Why, What, Shape and Checks. Edit How as often as you like; any other edit
makes the approval stale and locks code edits until the human approves again.

### The gated flow

1. **Draft.** `comments new <slug> --template brief`. Why is the human's: use
   their words, and if you paraphrase, post a non-blocking thread saying so —
   never a blocking one, since you cannot close it and approving the brief
   owns the Why. List premises first in How, each with a `file:line` citation, and
   verify them before you draft the rest — a false premise changes the brief.
   Write checks as commands, and include one where a fresh-context reviewer
   attacks each invariant. `comments validate` until clean.
2. **Self-review.** Post a few specific threads where the brief turns: your
   weakest step, assumptions, rejected options you are least sure of. Mark the
   two or three that matter `priority: high`. Silence means you checked.
3. **Hand off once.** Fold every reply in before you hand off; the hand-off is
   for the decision, not a round trip. Tell the human `comments view <doc>`,
   then `comments watch <doc> --until signoff --since <now>`.
4. **On the verdict: inbox first.** Replies are the payload, the decision is
   the envelope. Process every item, then: `approved` → build; `commented` →
   a reply pass, iterate and hand off again; `changes_requested` → fix and
   hand off again.
5. **Build.** Work against the checks, not a step list.
   - Decide alone where you can and file a pick where it applies:
     `comments add <doc> --anchor "..." --author <you> --type Q --text
     "Cache or index? Index: reads dominate." --pick index`.
   - Stop and ask — a blocking thread, then end your turn — only on the
     brief's stop-and-ask triggers, a false premise, or an invariant that
     would bend.
   - Keep How's status current. If a human section is wrong, do not edit it:
     post a thread at that line or a `suggest`, and keep going where you can.
   - Read the inbox at each stopping point; the human steers with threads.
6. **Finish.** Run every check and quote the output in How's status. The
   fresh-context adversarial review is required, not optional: give it the
   brief and the diff, and fix or file what it finds. Close every thread you
   fixed wherever you may (`reply --resolve`); reply in human zones and on
   picks, which only the human's approval settles. `comments
   gate <doc> --strict` must then fail only on your open picks: approving the
   finished work accepts each one nobody replied to. Hand off once more,
   naming the picks and the evidence.

## Working with threads (any doc)

- **One action per thread**, blocking first: answer it, apply it (edit, then
  `reply --resolve` saying what changed), propose it (`suggest`), or push back
  with reasons and leave it open. Never resolve a blocking thread without the
  fix or the human's agreement.
- **Copy thread IDs from tool output** when replying in bulk; sibling threads
  on one line are easy to cross. `reply --json` is atomic, so keep human-zone
  threads out of a resolving batch.
- **Rewrite, don't append.** Answer feedback by rewriting the passage it
  concerns, within the section's cap. Deleting is a valid answer. If seating
  new material evicts something, say what in the reply.
- **After editing a commented doc, reanchor** the threads your edits moved
  (`reanchor --json` for many); `inbox` shows any still `orphaned`.
- **Decisions made outside threads don't exist.** A decision reached in chat
  gets a thread the human ratifies by resolving it.

## Legacy workflows

Research → plan chains, design docs, RFCs and ADRs still work; their templates
load and gate as before. Use them only when the human asks for one by name —
see [legacy.md](legacy.md).
