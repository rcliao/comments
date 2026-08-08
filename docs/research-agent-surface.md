# Research: agent-surface friction in the comments CLI/MCP

## Research Question

Which operations on the agent surface (CLI + MCP) cost agents the most
round-trips or dead-ends, evidenced from real sessions, and what capability
gaps do they imply? Corpus: two independent agent sessions on 2026-08-07 —
one external (design-readiness review) and this repo's RPI dogfooding.

## Summary

Four gaps, three hit by both sessions independently. Worst by frequency:
comment creation demands a line number the tool cannot supply, forcing a
grep per anchor (~10/session each) and enabling silent mis-anchors. Worst by
severity: list's output shape — it flattens roots and replies into one list,
handing out reply IDs the write path rejects, while nesting the reply text
both sessions believed was absent; the confusion presents as lost feedback.
Validate's word-cap reporting left one agent trimming blind for eight passes.
Seed's template-recording semantics are implicit on MCP. The zone-refusal
error and batched section-based reanchor are the surface's proven strengths.

## Findings

### F1 — anchoring requires a line number the tool never provides

`comments_add`/`suggest` accept only `line` or `section` (pkg/mcp/types.go:39).
No operation maps text to a line, so both sessions ran `grep -n` before every
add (~10×), and lines go stale after each edit. This session mis-anchored a
thread by guessing (131 vs 135, caught by hand). The tool already prefers
name over number elsewhere: `reanchor` accepts `section` moves.

### F2 — list's flattened shape defeats both ID use and reply reading

MCP list flattens roots AND replies into one list (`GetAllComments`,
pkg/mcp/tools.go:86,137) — each reply appears as a standalone item with its
own ID, which `AddReplyToThread` then rejects: lookup walks roots only
(pkg/comment/helpers.go:132-152). Meanwhile `NewCommentView` nests full reply
text under each root (pkg/comment/json.go:32,60-62) — the text both sessions
believed absent WAS present, duplicated below its flattened copies. Result:
the external agent hit "thread not found" twice and fell back to reading the
sidecar raw; neither session found the nested replies. Filters apply to the
flattened set, so a filter can surface a reply without its parent. One shape
produces the dead end AND the invisible payload.

### F3 — word-cap violations don't name the culprit (external report)

The external agent saw only the doc total ("2003 > 2000") across eight trim
passes and computed per-section counts itself to find the real offender.
This repo's validator DOES emit per-section `over_length`
(pkg/comment/template.go:296-302, observed firing this session) — so their
miss implies the section matcher failed for that heading, or a doc-total-only
report drowned it. Unverified which; and when caps pass there is no way to
see per-section counts at all for informed trimming.

### F4 — seed's template-recording semantics are implicit

CLI seed says "template %q recorded" (cmd/comments/template.go:144). MCP seed
returns `template` + `seeded: []` (pkg/mcp/template.go:66-70) — the name is
present but nothing states it was RECORDED and gate-enforced, and the external
agent read the empty list as the whole answer. Both sessions paused here.
Recording a template is a distinct act from seeding threads; the surface
merges them.

### F5 — what already works

The zone-refusal error ("…is in a human-decision zone; reply with your input
instead") stopped the external agent from resolving a human question — its
best message. Batched section-based `reanchor` and `inbox`/`watch` worked
every time they were reached for. Exercised without friction across both
sessions: batch_add/batch_reply, suggest/accept/reject, request_review/
check_review, status. Not exercised at all: MCP resources — no evidence
either way.

## Code References

- pkg/mcp/types.go:39 — add takes line/section only
- pkg/comment/helpers.go:132-152 — reply lookup walks roots only
- pkg/comment/json.go:32,60-62 — reply text nested; pkg/mcp/tools.go:86,137 — flattened list
- pkg/comment/template.go:296-302 — per-section over_length exists
- cmd/comments/template.go:144 — CLI seed records template; MCP output differs

## Open Questions

- F3: is the external agent's missing per-section violation a heading-match
  bug or a reporting-order problem? Needs a reproduction against their
  design-doc template before the plan fixes the wrong layer.
