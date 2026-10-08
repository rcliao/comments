---
comments:
    phase: done
    template: living
status: draft
title: One Doc Workflow
type: Living
---

# One Doc Workflow

How comments should support one doc that starts as a researched brief and becomes the source of status while we build.

## Now

- **State:** done. All four slices are built, reviewed twice, and committed. Your doc now shows its phase and this line in `comments view` and in Claude Code's status line.
- **Next:** the dogfood run's picker fix and its three skill and template tweaks ship with PR #28, as plugin 3.1.0. Your own next task is still the last check.
- **Needs you:** nothing.

## Why

In your words: "an initial doc where agent research to produce a quality brief, but as we build that doc is kept updated as a source of status". One doc for plan, recap, status and explanation; chat is the interface; you refer back to the doc for the latest state; the doc keeps improving and guides the build.

## Findings

- F1. The living doc covers the build half (live reload, the seen baseline, the mod's drift count) but not the research half. Its template has no Findings section and no criteria for a brief's quality, and it leaves `check_citations` off. That check exists and resolves citations; it does not require one per finding. pkg/comment/templates/living.yaml pkg/comment/template.go:484
- F2. The view's rail leads with the verdict badge, and its hint says "q verdict". A living doc has no verdict, and there q records what you saw. pkg/tui/rail.go:61 pkg/tui/rail.go:89 pkg/tui/keys_browse.go:20
- F3. The field moved attention to two points: the human signs the plan once and the finished work once. Everything between them is a queue, not a stop. docs/research-notes/plan-mode-mods-2026-10.md:205
- F4. A hosted living doc already exists: Claude Docs, with no approval and no history. Ours differs in four ways: it lives in the repo, tints what changed since you looked, counts drift, and records decisions. docs/research-notes/plan-mode-mods-2026-10.md:187
- F5. Plans are mostly thrown away: 85 plan files committed across 36,710 repos. Ending the doc as the recap is our bet on what makes it worth keeping, not a finding of the source. docs/research-notes/plan-mode-mods-2026-10.md:167
- F6. The skill leads with the brief flow; living docs are a side section. There are ten templates to choose from. skills/review-comments/SKILL.md:33 skills/review-comments/SKILL.md:87
- F7. Frontmatter already has a validated `status` (draft, stable, deprecated), so a lifecycle field must not duplicate it. pkg/comment/metadata.go:147
- F8. Claude Code's recap is a one-line summary of up to 400 characters. It shows when you return after 3+ minutes, or on `/recap`. No hook event carries it, no plugin can feed or read it, and no file stores it. The session name and the status line are what a plugin can reach. docs/research-notes/plan-mode-mods-2026-10.md:213

## Plan

**Slice 1: shaping is part of the doc** (template, validate; F1, F5)

- `living` gains an optional Findings section and turns on `check_citations`. A new rule fails any Findings line without a citation. Plan cites findings by number.
- Review criteria for a quality brief: every premise is cited, rejected options are recorded, every check is a command, and a fresh reviewer has read it before go.
- The criteria for Now and Explanation say what done looks like: Now holds the result and its evidence, and Explanation is the recap of what was built.

**Slice 2: the view reads like a status page** (pkg/tui; F2)

- For a living doc, the rail shows the phase and the first line of Now instead of the verdict badge, and the hint reads "q  close".

**Slice 3: one flow in the skill** (F6)

- The skill leads with this flow: research, then brief, then build, then recap, in one doc. Briefs, plans and research docs move to legacy. Nothing is deleted.

**Slice 4: the doc's Now is the recap a plugin can show** (mods/comments-review; F8)

- The status line shows the doc's phase and the State line of Now, so the recap sits beside Claude Code's own and comes from the doc.
- Now's criteria say its State line reads as a recap: one line, under 400 characters, rewritten each turn.

**Not doing**

- No attempt to feed or read Claude Code's own recap: no supported way exists (F8).

- No hosted page, no deleting templates, no change to threads.

## Decisions

- One doc from research to recap, not a research doc plus a brief: two docs drift apart. (chat, 2026-10-08)
- **Go: 2026-10-08.** A go in chat is the sign-off; it is recorded here, with no verdict. (chat)
- The phase lives in `comments.phase` (shaping, building, done), leaving OKF's `status` alone (F7). You did not object to the recommendation. (agent, 2026-10-08)
- Integrate with Claude Code's recap, so the session's own recap and the doc's Now tell the same story. (chat, 2026-10-08)
- Rejected: Claude Docs as the surface. It is not in the repo, and it has no changed-since view or drift count. (agent, F4)
- Folded into slice 1: a separate recap slice; it is template wording. (agent, after review)
- Ship the dogfood run's fix and skill tweaks on PR #28 and merge to main. (chat, 2026-10-08)

## Explanation

**The lifecycle.** You ask for a piece of work. I research and write Findings, then draft Why from your words, Plan and Checks. That is the shaping phase. A fresh reviewer reads it before I ask you.

You steer in chat until you say go, and I record that in Decisions. In building, every turn that changes code changes Now, and decisions get lines. At done, Now carries the evidence and Explanation becomes the recap.

**How you follow it.** Keep `comments view` open on the doc. The rail says the phase and Now's first line, and tinted lines are what changed since you last closed it. The status line under Claude Code shows the same recap, plus how many code edits the doc is behind. Claude Code's own recap stays as it is; the doc's sits beside it.

**What the review changed.** A fresh review of the diff found that the citation rule checked list lines inside code fences, and that numbered, `+` and indented lists slipped past it. Both are fixed with tests. Now is parsed the same way as findings, and `comments new` scaffolds `phase: shaping`.

**What was built.** Slice 1: Findings with `cite_each_item`, `comments.phase`, and criteria for a brief and for done. Slice 2: the living rail. Slice 3: the skill leads with this flow. Slice 4: `inbox --json` carries `files[].living`, which the mod shows in the status line.

## Checks

- Slice 1: `comments validate` passes on this doc. A test in `pkg/comment` adds a Findings line without a citation and expects the new rule's violation.
- Slice 2: a test in `pkg/tui` renders the rail for a living doc and expects the phase and the first line of Now, with no "APPROVED" and no "verdict".
- Slice 3: `grep -n '^## ' skills/review-comments/SKILL.md` shows the one-doc flow before any brief heading.
- `./scripts/ci.sh` prints `All CI gates passed.`
- A fresh-context reviewer reads this doc against the code before go (done once), and the diff against Plan before done.
- Manual: the next real task runs this way end to end, and you say whether the doc was the place you looked.
