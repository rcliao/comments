# Code-Debt Plan (3-agent review, 2026-08-05)

All findings verified against code by reviewers. Full reports in session; this is the merged priority order.

**Progress**: Wave 1 (P0 #1-6) done — commit 41b98e7, 2026-08-06. P3 #14 (gofmt/lint/CI) done — a8b5c8a. Wave 2 in flight.

## P0 — Correctness bugs (fix first)

1. Markdown parser treats `#` inside code fences as headings (parser.go:20) — corrupts sections/anchors/templates for any doc with code blocks. No fence test exists.
2. Suggestion StartLine/EndLine never recalculated after an accept (positions.go:10) — second accept edits wrong lines when OriginalText absent.
3. `comments_batch_reply` MCP silently drops failures, reports success (mcp/tools.go:708) — CLI validates-first; MCP must match or report `failed:[]`.
4. `comments_status` counts replies as threads + hardcoded `is_stale:false` (mcp/tools.go:187-216).
5. TUI error-state trap: no key clears `m.err` (tui/model.go:1219); UTF-8 byte truncation in 6 spots (mojibake); `get` misses nested replies (main.go:442).
6. Template suffix false-positive: "Problem" matches "Big Problem" (template.go:143).

## P1 — Root layering fix (enables everything)

7. `LoadFromSidecar` prints to stderr + writes files on READ (storage.go:96-118); `SaveToSidecar` rewrites the md non-atomically every save (storage.go:127). Split: `Load` returns issues; persist decision + printing move to cmd/mcp; temp+rename writes; write md only when Content changed. Three call sites already bypass the layer (watch, gate, mcp).
8. One serializer: `commentJSON` → pkg/comment; consume from CLI list/gate, MCP tools, MCP resources (currently 3+ shapes incl. PascalCase resources).
9. cmd handlers return error; stderr + single exit point (39 stdout errors, 72 os.Exit) — unlocks CLI tests (currently zero).

## P2 — Structure

10. Mode-descriptor registry in TUI (kills 4-switch registration trap); delete dead IsModal/IsInteractive; split model.go (2052 lines) per-mode; `refreshCursorView()` helper (8+ dupes); merge renderDocument/WithCursor clones (width consts disagree -10 vs -12 — latent scroll mismatch).
11. MCP `withDoc` wrapper + jsonToolResult sweep (tools.go predates helper); dedupe handleAccept; comments_list section filter → GetCommentsInSection.
12. pkg/comment dedupe: RecomputeAllSections==ComputeSectionsForComments, findCommentByID dupes, delete dead exports (IsRoot/IsReply always-const, Position, GetPath stub, ~12 unused exports).
13. Shared `comment.ApplyAndAcceptSuggestion` (cmd + tui duplicate the accept+recalc math).

## P3 — Hygiene

14. gofmt: 17 files fail across repo — add gofmt/golangci-lint (govet,staticcheck,errcheck,unused) + `-race` + macos leg to CI; timeout-minutes; drop python3 from smoke.
15. Usage-text truth: batch-accept/--sort priority/v1 suggest flags documented but nonexistent; CLAUDE.md drift (backup archival, conflict detection — neither exists).
16. Typed status constants; time.Now injection; modernize pass (min/max/SplitSeq); rune-safe truncate helper; theme globals → styleSet on Model (race under t.Parallel); template dir cwd-dependence (MCP server cwd ≠ project root).
17. Test gaps: sections.go ~0%, pickCandidate multi-match, FindGateTargets, LoadFromSidecar error branches, cmd/* everything, tui updateByMode/handleResize.

## Suggested waves

- Wave 1 (bugs): P0 items 1-6 — parallel agents, disjoint files.
- Wave 2 (foundation): 7+8+9 together (they touch the same seams).
- Wave 3 (structure): 10-13, then hygiene 14-17 rides along per file touched.
