# Golden facts: the comment re-anchoring cascade (19 facts)

Extracted from `pkg/comment/anchor.go` by reading the source.
Companion fixture to `golden-facts-gate.md`: same fact count, but MECHANISM
narrative (a five-step algorithm with tie-breaks) rather than enumerable rules,
so cap and format effects can be compared across content types.

1. An Anchor stores the target line's text plus one line of context on each side (ContextBefore, ContextAfter).
2. CaptureAnchor returns nil when the line is out of bounds (below 1 or past the last line).
3. There are three confidence levels, best to worst: exact, fuzzy, section-level.
4. The cascade has five steps: exact position, exact text search, normalized search, section-path fallback, orphan.
5. Step 1 accepts the stored line when it still holds identical text, with confidence exact.
6. Step 2 searches the whole document for an exact text match, also confidence exact.
7. Step 3 searches with normalization and yields confidence fuzzy.
8. normalize collapses whitespace runs to single spaces and lowercases the text.
9. Blank or whitespace-only anchor text is never chased past step 1, because it would match everywhere.
10. Multiple candidate matches are disambiguated by context agreement first, proximity to the old line second.
11. Context agreement scores 2 points per matching side, compared after normalization.
12. Context score dominates proximity (scaled by 10000), so a distant context match beats a nearer non-match.
13. The proximity term is bounded (1000 minus distance, floored at zero), so very distant candidates stop differentiating.
14. Step 4 falls back to the comment's section path, relocating to the section's start line with confidence section-level.
15. When the section path no longer exists and the anchor text was not found, the comment orphans with a reason naming the missing section.
16. Step 5 orphans with a reason quoting the anchor text, truncated to 60 characters with an ellipsis.
17. A comment that moves records its previous line in OriginalLine, updates Line, and reports moved.
18. RelocateLine reuses cascade steps 1-3 to map a line across document versions, falling back to the original line clamped into range — this is how the TUI cursor follows its text.
19. A comment with no anchor whose line is past the end of the document orphans with "Line out of bounds and no anchor to re-locate".
