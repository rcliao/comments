---
name: review-comments
description: Process human review comments on a markdown document one at a time using the comments CLI/MCP, then request further review or pass the gate. Use when a document you drafted has review comments to address, or after finishing a draft that needs human review.
---

# Review Comments Workflow

You are addressing human review feedback on a markdown document managed by the
`comments` tool (sidecar `.comments.json` files). Work through comments **one at
a time** — never batch-dismiss feedback.

## Prerequisites

The `comments` binary must be on PATH, or the `comments` MCP server connected
(`comments serve-mcp`). CLI and MCP tools are equivalent; examples below show CLI.

## The loop

1. **Check the gate** to see what needs attention:

   ```bash
   comments gate <doc.md> --json
   ```

   Exit code 0 means approved (nothing blocking); exit code 10 means changes are
   requested. The JSON lists `blocking`, `non_blocking`, and
   `pending_suggestions`, each with document context.

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

4. **Request another human pass** when you have addressed everything or need
   decisions. Via MCP call `comments_request_review` (blocks until the human
   runs `comments signoff <doc.md>`); without MCP, tell the human you are ready
   for re-review and ask them to run:

   ```bash
   comments signoff <doc.md>            # records approved/changes_requested
   ```

5. **Repeat** until the gate passes (exit 0 / decision "approved").

## Drafting mode (no comments yet)

When drafting a new document under a template (design-doc, adr, rfc, or a
project template):

1. **Before writing**, read the template as your writing brief:
   `comments template show <name>` (CLI) or `comments_get_template` (MCP).
   Respect section order, word budgets, and use `[NEEDS CLARIFICATION: ...]`
   markers wherever you would otherwise guess at the human's intent.
2. **After drafting**, self-correct structure until clean:
   `comments validate <doc.md> --template <name>` (exit 0 = conforms).
3. **Self-review, then post SPECIFIC callouts** — never dump the template's
   generic criteria on the human. For each template criterion, judge your own
   draft against it and post a comment about what YOU actually did, anchored at
   the exact line it concerns (batch-add is ideal):

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
   generic checklist.
4. **Seed the ambiguity markers**: `comments seed <doc.md> --template <name> --markers-only`
   — turns each NEEDS CLARIFICATION marker into a blocking Q thread at its line
   and records the template so the gate enforces structure. (Full `seed` without
   the flag also posts the generic criteria threads — for human-only workflows
   with no agent to do step 3.)
4. Request review: call `comments_request_review` (MCP) with the file path, or
   ask the human to review with `comments view <doc.md>` and sign off with
   `comments signoff <doc.md>`. While waiting, do not modify the document.

Zone rule: threads in sections the template marks `zone: human` cannot be
resolved by you over MCP — reply with your input and leave resolution to the
human. Seeded criteria threads answer questions about the human's judgment of
your writing; address the underlying issue in the doc, reply with what you
changed, and let the human resolve.

## After editing a document that has comments (required)

You know exactly how your edits moved text — the tool doesn't. After ANY edit
to a commented document, migrate the anchors your edits displaced by calling
`comments_reanchor` (MCP) with the batch of moves:

```json
{"filepath": "doc.md", "moves": [
  {"comment_id": "c7f3k", "line": 42},
  {"comment_id": "c9b21", "section": "Proposed Design"}
]}
```

Only list comments whose target text you moved, rewrote, or deleted-and-replaced;
untouched comments keep their anchors. The load-time re-anchor cascade is a
safety net for edits made outside this loop, not a substitute for this step.

## Conventions

- Author yourself as `claude` (or your agent name) in every comment/reply.
- Mark comments you add for the human as non-blocking unless they truly gate
  progress; reserve `--blocking` for must-answer questions.
- Prefer suggestions over direct edits whenever the human wording preference
  matters (tone, structure, scope).
