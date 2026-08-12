# Golden facts v3: how `comments gate` decides its exit code (19 facts)
1-10 original; 11-13 injected round 2; 14-16 injected round 4; 17-19 injected round 5.

1. Exit code 0 = approved; exit code 10 = changes requested (revdiff/Plannotator convention).
2. A document passes the gate when no unresolved blocking threads remain.
3. Any unresolved blocking thread makes the decision changes_requested.
4. Unresolved NON-blocking threads do not fail the gate in normal mode (reported only).
5. In strict mode, any unresolved thread OR pending suggestion fails the gate.
6. Resolved threads never block, and decided (accepted/rejected) suggestions never block.
7. Pending suggestions are tracked separately from comment threads and only fail the gate in strict mode.
8. The decision "commented" is a reply-pass and never a gate outcome — the gate always derives from blocking threads.
9. For a directory target, only markdown files WITH sidecars are gated.
10. A single-file target with no sidecar passes (treated as an empty, passing doc).
11. Template structural violations ALSO flip the decision to changes_requested (exit 10) at the CLI layer when a template is recorded or passed.
12. A suggestion marked blocking never blocks in normal mode — suggestions divert to the pending bucket before the blocking check.
13. Recorded signoffs/review records play no role in the gate computation — the gate derives only from current thread state.
14. File/load errors (bad path, unreadable file, template load failure) surface as command ERRORS, not as gate decisions.
15. Multi-file runs aggregate worst-across-files: any single changes_requested file makes the whole run exit 10.
16. --json emits a machine-readable report (per-file decision, blocking/non-blocking/pending lists, summary counts).
17. With no template recorded or passed, document structure goes UNCHECKED — the gate flags structure_unchecked rather than validating.
18. When the template sets check_citations, citation violations also count as violations (and flip the decision).
19. The most recent review record (last_review) is attached to the report for context, though it never affects the decision.
