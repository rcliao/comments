# Research: how comment anchors survive document edits

## Research Question

Q1. What does a comment anchor store at creation?
Q2. Through what steps does a displaced anchor re-attach after the document changes?
Q3. When does re-anchoring give up, and what does the reader see?

## Summary

An anchor stores the target line's text plus one context line each side, captured at creation. [Q1]
On document change, a cascade tries exact position, exact text search, then normalized search, with context agreement and proximity breaking ties. [Q2]
Failing all three, it falls back to the comment's section heading, and only then orphans — the thread survives with a stated reason, never silently dropped. [Q3]

## Findings

### F1 — Creation captures content, not just a number [Q1]

`CaptureAnchor` records the selected line and its neighbors (pkg/comment/anchor.go:26-39).
`add --line N` stores N exactly; the anchor is what makes N recoverable later.

### F2 — The cascade prefers staying put [Q2]

Step 1 checks the stored position for identical text (pkg/comment/anchor.go:57-59) — an untouched line never moves.
Steps 2-3 search the whole document, exact then whitespace/case-normalized, the latter labeled `fuzzy` (pkg/comment/anchor.go:80-90).
Multiple matches resolve by context agreement first, proximity second (pkg/comment/anchor.go:95-117).

### F3 — Section fallback, then honest orphaning [Q3]

With no text match, the comment lands on its recorded section's heading, labeled `section-level` (pkg/comment/anchor.go:138-148).
If the section is gone too, the thread orphans with a human-readable reason and survives in listings (pkg/comment/types.go:12-15).
Blank anchor lines never chase matches — they would match everywhere (pkg/comment/anchor.go:75-77).

## Code References

- pkg/comment/anchor.go:26-39 — capture; :57-90 — cascade; :95-117 — tie-breaks
- pkg/comment/types.go:12-15 — lifecycle states including orphaned

## Open Questions

- Should `fuzzy` re-anchors surface in review the way orphans do, or is silent success correct?
