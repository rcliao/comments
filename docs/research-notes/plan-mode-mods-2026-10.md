# How people bring plan review back with mods (2026-10-05)

Question: how do people do plan / RPI review in Claude Code now that mods exist
(2.1.287+) and auto mode is the default (2.1.283+)? What should `comments` copy?

## Findings

- **Plan mode lost its default slot, not its existence.** v2.1.283 made auto mode the
  starting mode; plan mode is "a prompt plus an edit block — not a sandbox"
  ([Aleinikov, 2026-09-28](https://www.alekseialeinikov.com/en/blog/topics/ai/claude-code-modes-compared-why-plan-mode-died)).
  bcherny on [HN "Plan mode is dead"](https://news.ycombinator.com/item?id=49840054):
  "plan mode has always been a prompt … planning had become interactive and iterative."
- **What people do instead:** write a markdown plan, review it in Plannotator, fresh
  session from the plan, implement, review again (HN: stingraycharles, soleveloper).
  soleveloper asks for "github PR reviews combined with Google docs comments & suggestions"
  with per-section tracking — the gap `comments` fills.
- **Plannotator** ([docs](https://docs.plannotator.ai/open-source/agents/claude-code)):
  intercepts ExitPlanMode; browser-only review UI; feedback returned as structured
  markdown (quote + comment + line, edits as unified diff). Since 2.1.287 it ships a
  **mod** (non-blocking): Claude ends its turn, the decision "arrives in the session as a
  message from the Plannotator plugin", status line shows pending reviews. Limit it states:
  no OSS review session survives across feedback rounds.
- **vellum** ([PR #104](https://github.com/bengous/claude-code-plugins/pull/104), merged
  2026-09-18): function-hooks module. Key mechanics:
  - `classic.PermissionRequest` on `ExitPlanMode` answers in ~20 ms with
    `{ decision: { behavior: "deny", message: "Plan vN is open for review in the browser. End your turn; the review arrives as a new prompt." } }`
    — so the terminal approval dialog never appears.
  - feedback: server writes `.review/vN.feedback.md`, mod wakes Claude with `$.prompt.submit`.
  - approval: two-step — prompt Claude to call ExitPlanMode again; that call gets
    `allow` + `updatedInput.plan` + `setMode default` ("an allow without updatedInput is ignored").
  - subagent ExitPlanMode calls pass through (`next(e)`); status line `⚠ vellum: plan vN under review`.
- **Mod catalogs** (explainx 50-mod list, onewave, ruvnet field guide, az9713 mod pack,
  issue #91870): **no mod does plan or document review in a terminal pane.** Closest: approval
  gates on `tool.call` (Blast Radius, Ship Gate), a "Markdown Map" viewer, Decision Log.
  ruvnet's guide confirms the pane limits we hit: inline = a third of the screen; dock only in
  fullscreen ≥110 cols; long trees scroll only while focused.

## Implication for comments

Everyone splits the job: **mod = control loop** (intercept ExitPlanMode, status, wake Claude,
approve via the second ExitPlanMode call), **browser = reading surface**. `comments` already
has the browser surface (`comments serve`) and adds what Plannotator lacks: threads that persist
across rounds, agent replies, the gate, templates.

## Round 2 — who else builds review loops as mods (2026-10-05, three parallel researchers)

**Correction:** vellum's ExitPlanMode flow above is history (commit 1b267b5). Since a50caf6
(2026-09-16) vellum runs its own mode (`mcp__vellum__submit`, `tool.check` gating, subagents
skipped via `e.agentId`). The live reference for the ExitPlanMode loop is Plannotator's mod.

### Mods doing doc/plan review (all read from source, all with `"modules"` in hooks.json)

| Mod | Surface | Verdict back to Claude | Persistent threads? |
|---|---|---|---|
| [revelationnow/claude-doc-review](https://github.com/revelationnow/claude-doc-review) `doc-review/hooks/register.tsx` (1527 lines, verified) | docked Pane, block-by-block, heading+quote anchors, diff vs last review; opens itself on Write/Edit of matching globs | `$.prompt.submit({asUser:true})` of all comments; approve phrase | `$.store` only; side questions via `$.model.fork` |
| [backnotprop/plannotator](https://github.com/backnotprop/plannotator/tree/main/apps/hook/hooks/mod) `apps/hook/hooks/mod/register.ts` | browser (detached CLI) | result file → 1 s `$.clock.every` → `$.prompt.submit`; >12 KB to file | no — one-shot markdown per round |
| [devenjarvis/moot PR #17](https://github.com/devenjarvis/moot/pull/17) (merged 2026-10-03, verified) | Pane replacing the plan dialog; sections, outline, notes per section | 1 approve / 2 notes back / 3-Esc native dialog | no |
| [masahide/agent-kit doc-desk](https://github.com/masahide/agent-kit/tree/main/plugins/doc-desk) | browser form | blocking tool ~300 s, then `clock.every` + `$.prompt.submit`; `tool.check` locks the doc | no |
| [Alexander-Prime/revdiff-relay](https://github.com/Alexander-Prime/revdiff-relay) (verified) | `zellij run --floating` revdiff | FIFO read by `$.process.spawn(cat)` → `$.prompt.submit` per flush | no |
| [davekiss/md](https://github.com/davekiss/md) | markdown Pane, quote anchors, suggestions | own registered tools (`mcp__md__comment/reply/suggest`) | in-mod only |
| speckit-companion, infiquetra /plan-view, meganemura/draft-pane, konnokai plan-tally, opum-ai/lore-cli (OKF bundle CLI + mod — direct overlap) | various | — | — |

Classic-hook (non-mod) peers: [umputun/revdiff](https://github.com/umputun/revdiff) (916★, TUI, exit 0/10,
launcher cascade tmux popup → zellij → **herdr** → kitty/wezterm/…, verified) and cc-thingz planning
(warns its unbounded sentinel wait can hang the session); hyperlogue/r3 (browser; wakes sessions via
`CLAUDE_CODE_MESSAGING_SOCKET`); 7rah/humanize (Stop-hook RLCR loop).

### Mechanics worth copying (cited to source by the researchers)

- **Plannotator ExitPlanMode loop:** `tool.call` deny ("NOT approved… End your turn. The decision will
  arrive as a message") skipping `e.agentId`; on approval Claude re-calls ExitPlanMode and the
  hash-matched call gets `classic.PermissionRequest` → `allow` + `updatedInput.plan` +
  `updatedPermissions:[{type:'setMode',…}]`, answered without `next()` so the classic fallback hook
  never double-reviews. Waits run through `$.process.run`, not `$.clock`, to spare the hook budget.
- **revdiff herdr launcher:** detect via `$HERDR_ENV`; tab by default, zoomed side pane opt-in
  (`REVDIFF_HERDR_PANE=1`); copy the caller id before use; never guess a pane id when split returns none.
- **Gating while review is open:** `tool.check` refuses writes to the doc (doc-desk, vellum).
- **Ambient status:** band/status line with review state (vellum, Plannotator).

### Pane limits others hit (official docs + issues)

Inline pane = a third of the screen; dock only in wide fullscreen; unasked panes need 144 cols;
Esc only returns focus. Open asks: #99404 (`placement: inline|dock`), #99508 (desktop docking),
claude-bookmarks #18 (no band→pane focus handoff; plugins can't declare keybindings). No Anthropic
statement on post-plan-mode review or a document/takeover pane.

### Positioning

Converging: hook ExitPlanMode (or skip it in auto mode); non-blocking review returning as a plugin
turn; comments kept out of the plan text; full reading outside the pane (browser or multiplexer).
**Unclaimed:** threads that persist across rounds (replies, resolve, blocking), a machine-readable gate,
a human-only verdict, CLI/MCP parity. Every peer sends one-shot feedback and forgets it.

## Round 3 — what replaces plan mode for plan → implement (2026-10-06)

**Date corrections:** Ultraplan was removed in v2.1.222 ("Removed ultraplan feature"). Auto mode became
the default in steps: new Pro/Max/Team sessions from Aug 14 (Week 32 digest), third-party providers in
v2.1.283, every interactive terminal/VS Code session in v2.1.284 (Sep 28).

### Anthropic's own direction (changelog to 2.1.291, docs)

- [Best practices](https://code.claude.com/docs/en/best-practices) still teach "Explore → Plan →
  Implement → Commit" ("If you could describe the diff in one sentence, skip the plan"), plus: interview,
  write SPEC.md, implement in a fresh session; review the diff "against PLAN.md" with a subagent; a
  verification ladder prompt → `/goal` condition → Stop hook → verification subagent/workflow.
- [`/goal`](https://code.claude.com/docs/en/goal) (since 2.1.139) lists "Implementing a design doc until
  all acceptance criteria hold"; "auto mode removes per-tool prompts, and `/goal` removes per-turn prompts."
- [Artifacts](https://code.claude.com/docs/en/artifacts) with comment threads (2.1.221+), Claude
  auto-replies (2.1.228+); bcherny attaches artifacts to PRs (not built in: issue #95528).
- [Workflows](https://code.claude.com/docs/en/workflows): multi-angle planning; "No mid-run user input.
  For sign-off between stages, run each stage as its own workflow."
- No mod examples for planning from Anthropic.

### Practitioners (Aug–Oct)

- **Plans shrink to decisions.** superpowers [PR #2333](https://github.com/obra/superpowers/pull/2333)
  (merged 2026-09-25, verified): "a plan is decisions, not a transcript"; plans 10–12k → 0.9–1.4k lines,
  9/9 executed. HumanLayer RPI → "CRISPY" (Questions, Research, Design, Structure, Plan, Work, PR):
  ~200-line design discussion beats 1,000-line plans; long workflow prompts get steps "probabilistically
  skipped"; "we tried not reading the code for like six months, it did not end well".
- **Fewer human gates.** superpowers v6.4.1 moved to "one review at the end"; Kiro Quick Plan drops
  approval gates between phases. Pattern: one human gate early (design/plan), one at the end (diff/PR).
- **Done = executable check:** `/goal`, Stop hooks, Ralph loops, tests. Drift audits after the fact:
  Spec Kit `/speckit.converge` (labels gaps missing/partial/contradicts/unrequested), OpenSpec spec diffs.
- **Failures reported:** plan lost on compaction (#24686, closed not planned); plans nobody reads;
  AI-prose fatigue ([Nadeem, 2026-09-24](https://www.aymannadeem.com/artificial/intelligence,/developer/tools/2026/09/24/plan-mode-is-dead.html)); plans diverge from code.

### Plan-as-contract mods/hooks (read from source)

| Project | Enforcement | Notable |
|---|---|---|
| [leinardi/adversarial-review-loop](https://github.com/leinardi/adversarial-review-loop) (verified) | `/implement plan.md` freezes the plan (sha256-verified revisions); mutations denied until phases frozen; each phase commit reviewed by an isolated reviewer vs the frozen plan; Stop gate | user-only verbs (accept/pause/resume refused to the model); after compaction re-injects only gate state, never reviewer prose; fails closed |
| bengous vellum | `tool.check`: writes only inside the plan dir until approved | fail-closed `.catch` |
| masahide doc-desk | `tool.check` locks the doc; `session.compact` re-appends the human's decisions verbatim | |
| mmurakaru/cueloop | ExitPlanMode deny; unchanged approved plan passes; non-read-only tools denied while threads pending | fails open |
| 7rah/humanize | Stop hook blocks on incomplete todos / plan changed since start; Codex review loop | plan quiz before loop |
| edwinhu/workflows | Stop hold until check passes and judge agrees; approval keyed to a content hash | "crash = deny" |
| whoisclebs/openspec-governance | `tool.call` denies writes outside `openspec/` until tasks/specs are structured | structural, no human |

Shared rule among the strong ones: **approval bound to a content hash, the model cannot run the
approve/escape verbs, gates fail closed.** Unclaimed: file-level drift (edits to files the plan didn't
name); `prompt.compose` plan injection (nobody uses it; compaction handled via `session.compact` or
SessionStart `compact|resume`).

## Round 4 — repositioning: where human attention goes (2026-10-07)

Trigger: dogfooding the plan contract loop cost 4 approvals, 2 manual "hand off" prompts and 3 per-phase checks for one small plan, against a designed two human moments. Three parallel researchers: new mods/CC features, practitioner attention patterns, overlapping tools.

- [x] Practitioner attention patterns
- [x] New mods and Claude Code features (after 2.1.291)
- [x] Overlapping tools and differentiation
- [x] Spot-check key claims; positioning recommendation

### Practitioners: from approving steps to verifying outcomes (researcher report, primary sources unless marked)

- Anthropic, Measuring agent autonomy (Feb 2026, anthropic.com/research/measuring-agent-autonomy): experienced users auto-approve more (~20% → 40%+ of sessions) and interrupt more (5% → 9%); oversight = intervene when it matters.
- Anthropic, Claude Code auto mode (2026-03-25, anthropic.com/engineering/claude-code-auto-mode): 93% of permission prompts approved, named approval fatigue; a classifier replaces prompts and escalates after 3 consecutive / 20 total denials.
- Willison (2026-08-22, simonwillison.net/2026/Aug/22/more-than-just-code-review/): the skill is to "confidently verify" a change; line-by-line eyeballing never was the best way.
- AAIF, "Code is cheap. Proof is the bottleneck" (2026-09-08): policy before, evidence during, diff + trajectory review after.
- Park et al., arXiv 2609.24234 (2026-09-21, n=19): effort concentrates in planning, supervision is delegated, repeated guidance becomes reusable assets. Planning survives; approval steps do not multiply.
- Loop engineering, arXiv 2608.21884 (2026-08-22): agent loops in 217/256 mined repos — triggers, machine-checkable stop conditions, verifier sub-agents, budgets, defined human escalation points.
- Spec Kit June 2026 newsletter (github/spec-kit newsletters/2026-June.md): users report "markdown-review overhead", "review overload", "verbose"; responses are a TinySpec preset and `/speckit.converge` (check code against spec instead of another approval).
- Nearform (2026-06-24): SDD failure modes are "SDD everywhere" and rigid gates; size the process to the task. Thoughtworks Radar: SDD in Assess, ceremony can double docs (volume/date unconfirmed).
- Kiro: background Workflows (2026-09-30, kiro.dev/changelog); 0.12 "Quick Plan" collapsing three gated spec phases (reported 2026-05-06, unconfirmed on changelog).
- Codex/Cursor/Devin centre the human on a PR review queue; Devin Review (Jan 2026): "code review, not generation, is the bottleneck." Warp attaches screenshots/video to PRs as proof.
- CMU, arXiv 2607.07980 (2026-07-08): unreviewed agent PR merges fell from >50% to ~12% (human baseline ~14%); 40.1% reviewed only by the person who ran the agent; no-review rate tracks risk (tests 69%, bug fixes 25%).
- Abubakar et al., arXiv 2608.04661 (Aug 2026): only 85 committed agent-plan files across 36,710 repos — plans are mostly throwaway.
- Not found: any measurement of how much of an agent-written doc humans actually read.

### New mods and Claude Code features, Sept 25 – Oct 7 (researcher report; mods read from README only, none run)

- primestack-labs/plan-review (repo created 2026-10-07): `/plan-review plan.md` renders a reviewer-shaped plan in the browser (goal, interfaces, decisions, risks; steps collapsed), per-section approve/comment, Claude revises and opens the next round with changed sections marked. Skills + a hooks.json hook that offers review when a plan lands in `plans/`. Closest direct competitor.
- Claude Code Projects, public beta (code.claude.com/docs/en/claude-projects; ~2026-09-17 per press): a coordinator spawns parallel cloud threads; Overview groups "Waiting on you / Ready for review / Landing"; threads auto-fix PRs and notify. Docs: "don't open a PR until I've seen the plan" is memory, "not enforced settings".
- BuddyLim/claude-code-mod-parked (2026-10-04): the agent files each decision with numbered options and its own pick, proceeds on the pick unless blocked; the human answers later in a pane with one key, answers return as one batch. Companion claude-code-mod-ledger (2026-10-06) records plan tasks, gate verdicts and findings via mod tools.
- Sennjen/claude-sdlc (2026-09-24): intent → spec → plan → build gates; approval only from a real user message, sha256-bound; code edits need current approved hashes; editing an approved artifact reopens it; edits outside the plan's files get a drift note. Same idea as our contract gate.
- Claude Code 2.1.292 (2026-10-06): a mod's `tool.check` allow can no longer skip the dialog of a tool that needs the user's answer (questions, plan approval) — mods gate tools, they cannot approve for the human. Routine runs publish private artifacts without asking; `claude -p` waits for scheduled wakeups. 2.1.288–291: `tool.check` gained `agentId`/`ceiling`; auto mode compacts instead of prompting on long conversations.
- matheusbuniotto/lazyreview (2026-10-05): an "Expected" tab checks each ask against the diff and lists unrequested changes; verdict goes to GitHub or back to Claude.
- Gord1y/countersign (2026-09-28): one macOS queue for every session's permission prompts, shown only when you stop typing. pradyb/claude-mods notify-router (2026-10-05): notify on blocked only after N seconds, on done only for slow turns.
- Smaller: joonhyukyim/redpen (2026-10-07, diff line comments sent back as one prompt, not persisted); ipartington/claude-mods (handoff-on-clear, review-ledger, built from a 30-session audit); STpytut/agentic-control (human approves only the PR); claude.dev cloud guide (2026-10-01): "plan at your desk, build in the cloud, finish in your terminal".
- Gaps: GitHub code search needs auth and repo search rate-limited, so unnamed mods are likely missed.

### Overlapping tools (researcher report; spot-checks noted)

| Tool | Surface | Human-only verdict | Threads survive revisions | In repo | Agent API |
|---|---|---|---|---|---|
| Claude Code artifact comments | hosted page | no verdict; Claude resolves only threads an editor activated | yes, no re-anchoring documented | no | built-in Artifact tool |
| Claude Docs (~2026-09-16) | hosted living doc | no approval concept | text-anchored; no version history yet | no | claude.ai MCP connector |
| Ultraplan | browser plan review | — | — | — | removed (confirmed: code.claude.com/docs/en/ultraplan) |
| Google Antigravity | IDE plan comments, Proceed | optional ("Always proceed") | undocumented | local | built-in |
| Codex app | diff comments only | verdict goes to the forge | undocumented | local | built-in |
| GitHub Copilot | plans in `.copilot/plans`, PR review | yes for PRs (starter can't approve); plans uncommentable | PR threads go outdated | plans yes | CLI |
| Cursor | plan as editable markdown | no plan comments (request unanswered) | — | yes | — |
| crit (1,183★, v0.22.0 2026-10-07, confirmed) | browser + crit-tui mod | approve button; agent can `push -e approve` to a PR; local agent approval unconfirmed | yes (CarryForward remap) | no, `~/.crit` | CLI, MCP, hooks, forge sync |
| plannotator-tui (160★) | terminal annotation | no gate | per-file JSON in `~/.plannotator` | no | next agent message |
| codiff (1,409★) | desktop diff comments | no | no | yes | skills |

Long tail: five 0–3★ markdown annotators created Aug–Oct 2026. ChatGPT canvas comments: nothing found after 2024.

### Positioning (synthesis)

Commoditized: "comment inline on an agent's plan and the agent reads it" — shipped in browsers, terminals and hosted docs, with five clones in six weeks and crit as the direct substitute. Our TUI and browser surface are not the wedge.

Still ours, as far as these scans found: (1) a verdict only a human can record, on a document, enforced by a parity test — the only other hard human-only approval is GitHub's PR rule, on code; (2) the record lives in the repo, next to the doc, and survives a clone; (3) approval bound to content, with a CI exit code — the enforced version of what Claude Code Projects leaves as a soft instruction; (4) templates that keep agent docs short enough to read.

The field's direction explains our dogfooding friction: humans gate decisions and verify outcomes, and everything between is a queue, not a stop (auto mode, parked, Spec Kit converge, Kiro Quick Plan). We made every round a verdict. Repositioned: comments is the decision record and gate for agent work — the human signs the plan once and the finished work once; between them the agent proceeds and queues what it needs as threads carrying its own pick, settled in bulk without a verdict; the end review is evidence anchored to the plan's lines.

Constraint (2.1.292): a mod may gate tools but never stand in for the user's answer to a question or plan dialog. Check whether our `classic.PermissionRequest` allow of the second ExitPlanMode call still works.
