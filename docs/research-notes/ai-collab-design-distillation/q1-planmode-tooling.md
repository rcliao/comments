# Scratch: plan-mode / plan-review tooling (Q1 sub-report)

Provenance: received 2026-09-18 as an agent message from a sub-agent (id a7d7bfe63bbe65e31) not spawned directly by the lead session; presumed child of `q1-workflows`.
The sender states every quote was grep-confirmed against raw curl output.
The lead session has NOT re-verified these quotes; re-check any quote before it is pinned in the evidence note.

## 1. Anthropic: Claude Code best practices

URL: https://code.claude.com/docs/en/best-practices (raw: append .md).
https://www.anthropic.com/engineering/claude-code-best-practices now returns HTTP 308 to this page.
Org: Anthropic (Claude Code Docs).
Date: JSON-LD dateModified "2026-09-15T22:32:08.000Z".
The literal phrase "explore, plan, code, commit" is not on the live page.
The current heading is "Explore first, then plan, then code", with steps Explore / Plan / Implement / Commit.
Quote: "Letting Claude jump straight to coding can produce code that solves the wrong problem. Use [plan mode](/docs/en/permission-modes#analyze-before-you-edit-with-plan-mode) to separate exploration from execution."
Quote: "Planning is most useful when you're uncertain about the approach, when the change modifies multiple files, or when you're unfamiliar with the code being modified. If you could describe the diff in one sentence, skip the plan."
Quote: "A reviewer running in a fresh [subagent](/docs/en/sub-agents) context sees only the diff and the criteria you give it, not the reasoning that produced the change, so it evaluates the result on its own terms."
Quote: "A reviewer prompted to find gaps will usually report some, even when the work is sound".
No dedicated TDD section on the live page; remaining: "write a failing test that reproduces the issue, then fix it" and "have one Claude write tests, then another write code to pass them."
Supports: the human steers at the plan boundary; verification goes to tests and a fresh-context subagent.

## 2. Claude Code plan mode docs

URL: https://code.claude.com/docs/en/permission-modes#analyze-before-you-edit-with-plan-mode
Org: Anthropic.
Date: dateModified "2026-09-17T23:05:02.385Z".
Quote: "Plan mode tells Claude to research and propose changes without making them. Claude reads files, runs shell commands to explore, and writes a plan, but does not edit your source. Except in interactive terminal sessions with [bypass permissions available](#skip-all-checks-with-bypasspermissions-mode), edits stay blocked until you approve the plan."
Quote: "**No, keep planning**: stay in plan mode and tell Claude what to change."
Quote: "Press `Ctrl+G` to open the proposed plan in your default text editor and edit it directly before Claude proceeds."
Supports: review is an approve / keep-planning prompt; feedback is free-text chat or a whole-plan edit, with no line-anchored comments.

## 3. Cursor Plan Mode

URL A: https://cursor.com/docs/agent/plan-mode (undated).
Quote: "Plan Mode creates detailed implementation plans before writing any code. Agent researches your codebase, asks clarifying questions, and generates a reviewable plan you can edit before building."
Quote: "You review and edit the plan through chat or markdown files"
URL B: https://cursor.com/blog/plan-mode, "Introducing Plan Mode" (Cursor, 2025-10-07).
Quote: "When you’re happy with the plan, it creates a Markdown file with file paths and code references. You can edit the plan directly, including adding or removing to-dos."
Supports: the human reviews by directly editing a markdown plan, then clicks build; no annotation threads.

## 4. Plannotator

URL: https://github.com/backnotprop/plannotator (README undated; repo pushed_at 2026-09-18).
Quote: "Plannotator is a local, browser-based review surface for AI coding agents: Claude Code, Codex, Copilot CLI, Gemini CLI, OpenCode, Kiro, Droid, Amp, and Pi."
Quote (flow diagram, consecutive lines): "Agent calls ExitPlanMode" / "-> PermissionRequest hook fires" / "-> You annotate and approve/deny" / "-> Approve: agent proceeds" / "-> Deny: structured feedback sent to agent" / "-> Agent revises, plan diff shows what changed".
Quote: "No command needed. Plan mode is wired in through each harness's hooks."
The README documents no exit-code convention.
Supports: browser annotation plus a binary approve/deny verdict; deny carries structured annotations back through the hook.

## 5. revdiff

URL: https://github.com/umputun/revdiff (README on master, undated).
Quote: "TUI for reviewing diffs, files, and documents with inline annotations. Outputs structured annotations to stdout on quit, making it easy to pipe results into AI agents, scripts, or other tools."
Quote: "A separate `revdiff-planning` plugin automatically opens revdiff when Claude or Codex completes a plan, letting you annotate it before implementation. If you add annotations, the agent revises the full plan and presents it again — looping until you're satisfied. Exit code `10` means annotations were captured, not launcher failure."
Quote: "Exit status: `0` = no annotations, discarded annotations, or default mode; `10` = annotations were produced with `--exit-code-on-annotations`; `1` = real errors."
Also reported: "In Claude Code it hooks `ExitPlanMode`" and "By default revdiff exits `0` even when annotations are produced".
Also reported: an annotation containing "??" is treated as a question rather than a directive.
Supports: annotations on stdout plus opt-in exit 10; approval is implicit (quit with no annotations).

## 6. Similar tools

difit: https://github.com/yoshiko-pg/difit — Quote: "**Generate Prompts**: Comments include a \"Copy Prompt\" button that formats the context for AI coding agents". Browser diff viewer; clipboard hand-off; no exit codes documented.
tuicr: https://github.com/agavra/tuicr — Quote: "`y` or `:clip` copies a structured markdown block to your clipboard. Each comment has a number". Also `tuicr --stdout`; exit codes 0/1/2, no review-verdict code.
critique: https://github.com/remorses/critique — "Beautiful diff viewer for terminals, web previews, and agents." No annotate-back loop found; marginal.
Not searched (time budget): hunk, vibe-kanban. Not opened: Herdr Annotate, Plannotator TUI.

## 7. OpenAI Codex

URL: https://learn.chatgpt.com/docs/developer-commands?surface=cli (redirect target of developers.openai.com/codex/cli/slash-commands), undated.
Quote: "Switch to plan mode and optionally send a prompt." followed by "Ask Codex to propose an execution plan before implementation work starts."
"Read Only" is a separate approval/sandbox setting: "Relax or tighten approval requirements mid-session, such as switching between Auto and Read Only."

## Cross-cutting (sender's synthesis)

Vendor plan modes (Claude Code, Cursor, Codex) put review in chat or a whole-plan edit with approve / keep-planning; none offers line-anchored threads.
Third-party tools (Plannotator, revdiff) add anchored annotations by hooking ExitPlanMode and returning feedback to the agent.
Exit code 10 is documented by revdiff only, and there it is opt-in via a flag.
Plannotator's README documents no exit codes, so this repo's CLAUDE.md phrase "revdiff/Plannotator convention" is supported only on the revdiff side.
None of these tools persists threads or a recorded verdict in a sidecar file; Plannotator keeps a local archive of plan decisions.
