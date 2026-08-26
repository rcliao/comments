---
name: review-comments
description: Create template-guided OKF document bundles, load related context, annotate drafts, process human review comments one at a time, and wait for signoff using the comments CLI/MCP. Use for drafting or reviewing a managed Markdown artifact, including Research → Plan → Implement workflows.
---

# Review Comments Workflow

You are addressing human review feedback on a markdown document managed by the
`comments` tool (OKF-compatible Markdown plus sidecar `.comments.json` files).
Work through comments **one at
a time** — never batch-dismiss feedback.

## Prerequisites

The `comments` binary must be on PATH, or the `comments` MCP server connected
(`comments serve-mcp`). Every MCP tool has a CLI equivalent backed by the same
code, so the two are genuinely interchangeable; examples below show CLI.

Threads in a template's `zone: human` sections cannot be resolved by you on
either surface — the CLI detects an agent caller by the absence of a TTY. Reply
with your input and leave the resolve to the human.

## The loop

1. **Check the gate** to see what needs attention:

   ```bash
   comments gate <doc.md> --json
   ```

   Exit code 0 means approved (nothing blocking); exit code 10 means changes are
   requested. The JSON lists `blocking`, `non_blocking`, and
   `pending_suggestions`, each with document context.

   In a configured bundle, first run `comments context <doc.md> --for review
   --include-threads` (MCP: `comments_context`). Read the explicit related
   concepts and backlinks it returns; do not search the whole docs tree by
   default.

2. **Process each unresolved comment individually**, blocking comments first,
   then non-blocking. For each comment, choose exactly one action:

   - **Answer it**: if it is a question, reply with the answer:
     `comments reply <doc.md> --thread <id> --author <you> --text "..."`
     Do NOT resolve a question thread unless the answer fully settles it —
     leave resolution to the human when in doubt.
   - **Apply it**: if the fix is unambiguous, edit the document, then reply
     explaining what changed and resolve:
     `comments resolve <doc.md> --thread <id>`
   - **Propose it**: if the fix is a judgment call, create a suggestion instead
     of editing directly, and reply linking it:
     `comments suggest <doc.md> --start-line N --end-line M --author <you> --text "..." --original "..." --proposed "..."`
     The human accepts or rejects it; do not accept your own suggestions.
   - **Push back**: if you believe the comment is mistaken, reply with your
     reasoning and leave the thread unresolved for the human to decide.

   Never mark a blocking comment resolved without either applying the fix or
   getting human agreement in the thread.

3. **Re-check the gate** after processing all comments (`comments gate <doc.md>`).

4. **Hand off and listen for another human pass** when you have addressed
   everything or need decisions. Ask the human for **one** command — the TUI
   review ends in a signoff, so do not also ask them to run `comments signoff`:

   ```bash
   comments view <doc.md>     # review, then q -> a/c (n adds a note for you)
   ```

   Then wait on the signoff instead of asking them to tell you they are done:

   ```bash
   comments watch <doc.md> --until signoff
   # {"event":"signoff","file":"doc.md","author":"rcliao",
   #  "decision":"changes_requested","note":"pin the prompt, don't hash it"}
   ```

   `watch` exits 0 on the first matching event, so it is a blocking wait you can
   run directly; the event carries the decision and the reviewer's note. It sees
   every writer of the sidecar, so it fires whether the human signed off from
   the TUI verdict or from `comments signoff`. Point it at a directory to wait on
   a whole spec folder, and `--until signoff,gate_changed` to also wake on gate
   flips.

5. **When the signoff arrives: inbox FIRST, decision second.** Humans answer
   threads and then pick whichever verdict is nearest — their replies are the
   payload, the decision is the envelope. Before acting on any decision, run
   `comments_inbox` (or read replies since your last pass) and process every
   reply. Then interpret the decision:

   - `commented` — a reply-pass: the human answered your threads and handed
     the turn back without judging the doc. Process replies, iterate,
     re-request review. Never treat it as approval.
   - `approved` — proceed to the next phase, but only after the inbox is
     drained; an approve with unprocessed replies means the human trusts you
     to fold them in first.
   - `changes_requested` — process replies, fix, re-request.

6. **Repeat** until the gate passes (exit 0 / decision "approved").

## Drafting mode (no comments yet)

When drafting a new document under a template (design-doc, adr, rfc, mini for
small changes, or a project template):

- If `.comments/bundle.yaml` exists, create the artifact with
  `comments new <slug> --template <name> [--from <related.md>]`. This selects
  the review-friendly folder, emits OKF frontmatter with
  `comments.template`, creates the sidecar, and refreshes indexes. Do not hand
  assemble the folder or metadata.
- Before writing an existing bundle concept, run
  `comments context <doc.md> --for drafting --include-threads`. Treat explicit
  relationships as the working set; tag matches are suggestions, not evidence.

0. **Decompose the question first.** Where a template asks for enumerated
   sub-questions (`Q1.`, `Q2.`, ...), write them before drafting and tag each
   finding with the clause it answers (`### F1 — the cascade [Q1]`). This is not
   bookkeeping: a three-clause question written as prose produced a document
   answering two of them, and it passed every other check — word caps, citations,
   tone — because conformance says nothing about omission. Decomposing also
   improves the investigation, since each clause is searched for separately.
   Answer a "what would it take" clause as what is currently ABSENT; designing
   past it is the plan phase's job.

1. **Before writing**, read the template as your writing brief:
   `comments template show <name>` (CLI) or `comments_get_template` (MCP).
   Respect section order, word budgets, and use `[NEEDS CLARIFICATION: ...]`
   markers where you would otherwise guess at the human's intent — but stay
   under the template's marker cap: spend markers on the few questions that
   genuinely need the human (scope > security > UX > technical detail), make
   informed decisions on the rest, and record those as assumptions in the doc.
2. **After drafting**, self-correct until `comments validate <doc.md> --template <name>`
   reports **no structural violations**. It separates the two kinds: structural
   defects are yours to fix, and intentional `[NEEDS CLARIFICATION]` markers are
   listed as expected and left for the human. A doc that deliberately carries a
   marker still exits non-zero, so read the report, not the exit code.
3. **Self-review, then post SPECIFIC callouts** — never dump the template's
   generic criteria on the human. For each template criterion, judge your own
   draft against it and post a comment about what YOU actually did, anchored at
   the exact line it concerns (batch-add is ideal; use `anchor` — quote the
   target line — instead of grepping for line numbers):

   - Weakest reasoning: "I rejected option B mainly on argument X — my least
     confident step, please check" (type Q, blocking)
   - Assumptions: "Assumed the API is idempotent based on the client code —
     verify" (type Q, blocking if the design hinges on it)
   - Invented or from-memory facts: "This 40% figure is from memory, no
     source" (type Q, blocking)
   - Judgment calls the criterion asks about: "Non-goals exclude Y because of
     Z — widen if you disagree" (type S, non-blocking)

   A criterion your draft clearly satisfies needs no comment — silence is the
   signal that you checked it. The human reviews your specific doubts, not a
   generic checklist. For feature-sized docs, follow with the fresh-context
   reviewer pass (see RPI mode) before requesting human review.

   **On large artifacts, your open threads are also the human's walkthrough**
   (`P` in the TUI sorts the sidebar priority-first): the reviewer may have
   minutes of face time to discuss a design that took hours to write, and
   your threads are the slides. So:

   - Mark the 2-4 threads the design genuinely turns on `priority: high` —
     each names the decision, what hinges on it, and the ask, in one breath
     ("Chose skill-prose over a Go command; hinges on prompt quality — veto
     if you want it enforceable"). Everything else stays medium/low.
   - Anchor each high thread where the decision lives in the doc, so
     opening it puts the relevant detail beside it as the backdrop.
   - High-priority is a walkthrough slot, not emphasis — if six threads are
     high, none are.
4. **Annotate ambiguity markers yourself** with the existing `comments add`
   or `comments_batch_add` surface. Each marker gets a specific anchored,
   blocking Q comment that states the decision needed. Do not generate generic
   template-criterion threads. Template identity belongs in
   `comments.template` frontmatter (or is inferred from an unambiguous bundle
   collection); comments remain discussion, not configuration.
5. Request review by asking the human to use `comments view <doc.md>` (the
   verdict on exit records the signoff), then listen without requiring a nudge:
   `comments watch <doc.md> --until signoff`. While waiting, do not modify the
   document. On a re-request, first call `comments_status` with the
   reviewer's name and quote its `changed_since.changed_sections` in your
   message — the reviewer sees the same lines tinted in the TUI, and naming
   the sections you touched is what lets them skip the rest.

Zone rule: threads in sections the template marks `zone: human` cannot be
resolved by you over MCP — reply with your input and leave resolution to the
human. Address agent-authored annotations the same way as human threads: update
the document or reply with the decision, but leave human-zone resolution to the
human.

## Revising under review: rewrite, don't append (required)

Review feedback tends to be answered by adding — a clarifying sentence, a caveat,
a new subsection. Each addition is locally reasonable and the document gets worse:
it stops reading like something written once and starts reading like a transcript
of its own review. Measured across this repo's shipped RPI artifacts, half went
out over their word caps, and the overflow was always in the body section
(`Findings`, `Implementation Phases`) — never in the framing sections.

So when you address a comment:

- **Rewrite the passage it concerns.** The reviewer's point should be
  indistinguishable from the original argument once you are done. If a reader
  can tell which sentences were added in response to review, revise again.
- **Budget is a forcing function, not a limit to creep up to.** A section at
  90% of its cap has no room for the next round; make room by cutting what the
  new material replaces, in the same edit.
- **Never trim a different section to fit.** `comments validate` names the
  offending subsection precisely so the fix is local. Trimming elsewhere is how
  padding survives and cohesion dies.
- **Never split a finding or phase to dodge a word cap.** Templates cap
  subsection COUNT as well as size for exactly this reason. Two thin findings
  are worse than one dense one.
- **Deleting is a valid response to feedback.** If a comment reveals a passage
  is unnecessary, cut it and say so in the reply. A shorter doc after a review
  round is a good outcome, not a suspicious one.
- **Declare what you evicted.** A section at its cap behaves like a cache:
  seating new material silently drops something older, and the reviewer who
  asked for the addition never learns what it cost (measured in
  scripts/eval/logs/cap-pilot-2026-08-11.json — an overloaded section shed
  established detail to seat the newest comment, every run, announcing
  nothing). So when answering a comment forces content out, say so in the
  reply: "dropped X to seat this — raise the cap or accept". Never mistake
  the trade for a private editorial decision; the reviewer may value X more
  than their own comment. Losing detail you cannot name means you rewrote
  blind — re-read the section before replying.
- **A section pinned at its cap round after round is a template signal**, not
  a writing problem: propose raising the cap or splitting the section rather
  than compressing a third time. Compression at the margin is where meaning
  breaks — the same eval found qualifiers silently dropped ("fails on any
  unresolved thread" losing "or pending suggestion") while the prose stayed
  fluent.

Re-run `comments validate <doc> --template <name>` after processing feedback.
Treat any `subsection_over_length` or `max_subsections` violation as a rewrite
instruction, not a trimming exercise.

**Reply to the thread you are answering.** When replying in bulk, copy each
thread ID from the tool output rather than reconstructing it from memory —
sibling threads anchored at the same line are easy to cross, and a reply that
answers a different thread is invisible to the gate and confusing to the human.

## After editing a document that has comments (required)

You know exactly how your edits moved text — the tool doesn't. After ANY edit
to a commented document, migrate the anchors your edits displaced with
`comments_reanchor` (MCP) or `comments reanchor` (CLI):

```bash
comments reanchor doc.md --comment c7f3k --line 42
comments reanchor doc.md --json moves.json   # batch
```

```json
{"filepath": "doc.md", "moves": [
  {"comment_id": "c7f3k", "line": 42},
  {"comment_id": "c9b21", "section": "Proposed Design"}
]}
```

Check your work with `comments status <doc>` — a non-zero orphan count means
an anchor you displaced still needs migrating.

Only list comments whose target text you moved, rewrote, or deleted-and-replaced;
untouched comments keep their anchors. The load-time re-anchor cascade is a
safety net for edits made outside this loop, not a substitute for this step.

## Compounding: rounds build on doc + thread history (required)

The thread record is the design's memory; rounds inherit it, never restart.

- **History first**: before ANY drafting round, read the doc's full thread
  history — open AND resolved (the resolved trace is the reasoning archive).
  When drafting a plan, also read the research doc's threads.
  The fresh-context reviewer's allowlist includes the sidecar(s): fresh
  context applies to the draft, not to the review record.
- **Vetoes move into the doc**: when a design dies in a thread, write it
  under Options Considered (or What We're NOT Doing) citing its thread
  (`thread:c1abc`, cross-doc `thread:research.md#c1abc` — peekable with f),
  reply what was recorded, resolve the thread.
- **Following a citation you encounter**: paste it verbatim —
  `comments get 'thread:research.md#c1abc' --from <the-doc-you-read-it-in>`
  (MCP: `comments_get` with `cite` + `from`). The `--from` matters:
  citation paths are relative to the CITING doc, not your cwd.
- **No silent re-proposal**: never re-propose a recorded veto. New evidence
  that justifies revisiting must cite the vetoed thread and say what changed.
- **Decisions made outside threads don't exist**: a decision reached in chat
  gets recorded as a thread the human ratifies by resolving — otherwise the
  next round cannot inherit it.

## RPI mode (Research → Plan → Implement)

For feature-sized work, split drafting into two phase docs with the dedicated
templates. These steps are opinionated about what makes a quality artifact,
not a ceremony to perform: scale them to the work.

In a configured bundle, start the chain with:

```bash
comments new <slug> --template research-deep
comments context docs/artifacts/research/<slug>.md --for drafting --include-threads
# after research convergence:
comments new <slug> --template plan --from docs/artifacts/research/<slug>.md
comments context docs/artifacts/plans/<slug>.md --for drafting --include-threads
```

The shared slug makes navigation predictable; `--from` records the durable
`informed_by` edge instead of relying only on a prose citation.

**The autonomous chain is the DEFAULT** (decided in review, 2026-08-11: the
human's value lands at plan review — research signoff was ceremony). After the
interview, run question → research → plan WITHOUT a mid-chain human gate:

- Research uses the `research-deep` template (agent-audience caps — depth
  beats brevity; the human reaches detail via citations from the plan) and
  converges through the fresh-context reviewer alone. No human signoff.
- Minimize markers in autonomous research: decide and record assumptions.
  Human questions that survive CARRY FORWARD as priority-high threads on the
  PLAN — answered in the one review sitting, costing at most a revision round
  if an assumption was wrong. Carried threads ask SCOPE and PRIORITY
  questions (is this worth doing now, is the fence right) — implementation
  details are yours to decide, not to delegate upward.
- **Pause-on-shape**: a question whose answer would change the plan's SCOPE
  or direction (not its details) stops the chain — request review early with
  what exists. Use the interview ranking: scope > security > UX > technical.
- **The plan gets extra teeth** (the only human gate must be the strongest):
  TWO fresh-context reviewer passes with distinct lenses — one for
  correctness/coverage against the research, one for implementability as
  written — pass cap 3 total. Every citation verified, both directions.
- One human sitting reviews the PLAN (research one peek away as backdrop;
  carried questions lead the walkthrough). Research needs no signoff of its
  own; the human may `q → a` it in the same sitting if they read it.

### Autonomous research convergence (before drafting the plan)

Research must survive two INDEPENDENT, fresh-context roles; a generic review
pass is not a substitute. If subagents are unavailable, use separate fresh
sessions with the same allowlists.

1. **Coverage scout — source-derived, draft-blind.** Start from
   `comments context <research.md> --for coverage-scout --json`. This mode
   exposes the numbered Research Question as `focus`, while forcibly omitting
   bodies, threads, and draft-derived backlinks. Give the role ONLY the research
   question, repository access, and resolved coverage-rejection threads from
   earlier passes — NEVER the draft. It proposes missing questions, each with
   an expected answer and file:line evidence. Post each candidate as a
   `coverage-scout` thread.
2. **Evidence verifier — draft-derived.** Start from
   `comments context <research.md> --for evidence-verifier --include-body
   --include-threads --json`. Give it ONLY the research draft, its
   thread history, template criteria, and the files its citations name. It
   checks each material claim for support, contradiction, and overstatement;
   findings that would mislead the plan are blocking `evidence-verifier`
   threads.
3. **Reconcile one candidate at a time.** Accept a real coverage gap by adding
   the next `Qn` to Research Question, investigating it, and tagging its
   finding(s). Reject a duplicate or irrelevant candidate by replying with the
   evidence and resolving it; this is the negative-coverage memory. Fix or
   rebut verifier threads through the normal review loop.
4. **Converge, don't merely finish a pass.** Repeat both roles until
   `comments analyze <research.md> --json` reports `ready: true`, the latest
   scout pass adds no accepted question, and no blocking verifier thread
   remains. Record a resolved round-summary thread even on a clean pass so the
   no-new-question result is visible rather than inferred from silence.
5. **Bound exhaustion visibly.** Until the paired eval sets a measured cap,
   stop after 3 semantic passes. Carry survivors as priority-high PLAN threads;
   if any survivor changes scope or direction, pause before planning under the
   existing pause-on-shape rule.
6. **Prove the handoff.** Before human review, run
   `comments analyze <plan.md> --against <research.md> --json`. Cite every
   uncovered finding in the plan or explicitly fence it out with rationale.
   `ready: true` is required to request review, but remains advisory tool state;
   only the human plan signoff authorizes implementation.

The two-gate flow below remains for when the human ASKS to review research
(learning a domain, high-stakes direction) — say "gate the research" to get
it.

0. **Interview before drafting** (each phase doc): present your understanding
   of the question and the relevant code, plus only the questions you genuinely
   cannot answer from the codebase — then WAIT for answers before writing.
   Confirming your understanding IS the wait-for-question gate; answers land in
   the first draft (no separate write-back step), and questions that surface
   mid-draft use `[NEEDS CLARIFICATION]` markers as usual. A bad line of
   understanding becomes a bad plan section becomes hundreds of bad lines of
   code — redirecting you is cheapest before prose exists. Fast path: a trivial
   doc may state "no questions — drafting" and proceed.

   **Verify, don't trust**: read cited files yourself before delegating
   searches to subagents; when the human corrects you, verify the correction
   in code before building on it — do not just accept it.

1. **Research** (`comments template show research` is your brief): produce a
   documentarian findings doc — discrete findings (F1, F2, ...) each carrying
   file:line evidence, a Code References section a plan can cite, and every
   open question in Open Questions (zone: human). Open questions are
   DISPOSITION questions — act, park, or redirect — never facts you could
   verify yourself; one must ask whether the findings justify the next phase
   at all, and "no action" is a legitimate answer that ENDS the chain at
   research. Then the reviewer pass (below). In the DEFAULT autonomous
   chain, proceed straight to the plan once the reviewer converges — unless
   your own findings answer the disposition question with "nothing worth
   doing": then stop and hand the research to the human instead of
   manufacturing a plan. In gated mode, wait on the signoff, don't poll:
   `comments watch <doc> --until signoff` blocks until the review lands
   (run it in the background in harnesses that support it).
2. **Plan** (`comments template show plan`): decisions only — the marker cap
   is 1 because open questions belong to the research phase. Cite the research
   doc by `file:line` (e.g. `research-foo.md:23`) for every Current State and
   design claim: the human reviews the plan in the TUI and peeks each citation
   with `f`, so an uncited claim is an unverifiable claim. Keep the plan near
   200 lines; no code blocks — decide and point (file:line) instead.
3. Every phase in Implementation Phases carries Success Criteria split
   **automated** (a command or test) vs **manual** (human judgment).
4. Gate green + signoff on the plan → implement, phase by phase, verifying
   each phase's criteria before the next.

### Fresh-context reviewer pass (before requesting human review)

Your context wrote the doc, so it finds the doc's prose convincing — that is
the failure mode, not a safeguard. After your self-review callouts, spawn a
reviewer with fresh context and a strict input allowlist: the doc path, its
template criteria (`comments template show <name>`), and — for a plan — the
research doc path. Nothing else; a reviewer that inherits your drafting
context is theater, and if you cannot spawn subagents, a fresh session given
the same allowlist is an equally valid reviewer.

The reviewer posts findings as comments under its own author (blocking only
for what would mislead implementation), including the coverage question you
structurally cannot ask yourself: which research findings does the plan
silently drop, and which claims cite nothing. Process its findings through
the normal comment loop (apply / propose / push back), then re-run
`comments gate`:

- **Terminate on gate green** — a clean doc converges in one pass; this is
  not a fixed number of rounds.
- **Cap at 2 reviewer passes** (provisional default, tuned by dogfood
  metrics); leave survivors open for the human rather than spinning.

### Attention budget and thought-trace

Open threads are the human's reading list — keep them few enough to actually
read (~5, provisional default), priority-ordered; if more survive the
reviewer pass, consolidate before requesting review. Your working notes,
iteration rationale, and processed-feedback trace belong in threads you
resolve yourself immediately: the human reads them on demand (`R` toggles
resolved threads in the TUI), and the open set stays reserved for decisions
that are genuinely theirs.

## Conventions

- **Semantic line breaks in every doc you write or edit:** break lines at
  sentence boundaries, never at column width.
  Renderers join them invisibly, but the tool is line-addressed:
  one sentence per line means comments anchor to a sentence,
  an edit touches exactly its own line with no reflow below it
  (no anchor displacement, less reanchor churn),
  and diffs read as sentence-level track changes.
  A long sentence may break after a clause; never mid-phrase.
- **Comments are ≤50 words, one point each.** Lead with the actionable claim;
  evidence is a file:line, not a paragraph. If you need two sentences of
  context, the second is usually the comment. Split multi-point comments into
  separate threads so each can be resolved on its own.
- Author yourself as `claude` (or your agent name) in every comment/reply.
- Mark comments you add for the human as non-blocking unless they truly gate
  progress; reserve `--blocking` for must-answer questions.
- Prefer suggestions over direct edits whenever the human wording preference
  matters (tone, structure, scope).
