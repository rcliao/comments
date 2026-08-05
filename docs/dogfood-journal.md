# Dogfood Journal

Running notes from using `comments` on its own development. Each entry: what we did, what worked, what hurt, and what it taught us. Friction observed here feeds the backlog — this file is the tool's own user-research log.

Convention: newest entry first. Tag friction items `[friction]`, validated wins `[win]`, ideas born here `[idea]`.

---

## 2026-08-05 (late) — TUI research + dive-to-reply shipped mid-review

**What we did:** Researched terminal reading/review UX (glow, frogmouth, octo.nvim, prr, revdiff, neomutt, epy, Crush — report 4), drafted the review-first TUI design doc under our own template (seeded, awaiting Eric's review), and shipped two interactions Eric asked for while reviewing in the TUI.

- `[win]` octo.nvim independently validates our focus-follows-cursor pattern — its refinements (virtual text, thread-in-other-pane, verdict-on-exit) shape the design doc.
- `[win]` Shipped from live feedback: `r` in line-select dives into the cursor line's thread (Esc returns to the cursor), `Tab` cycles threads stacked on one line, sidebar selection follows the cursor. 3 new interaction tests (TUI suite: 12).
- `[win]` Second design doc drafted through the full template flow; seeded threads now show short IDs (`ceqp4`) and the Eric-tuned Risks criterion — earlier feedback compounding.
- `[friction]` The gap Eric hit (no path from cursor to reply) was already named in the design doc as G3 — the doc lagged the need; shipping the minimal version mid-review was the right call, doc will be updated to match during review processing.
- `[idea]` From research, cheapest next wins: `u` next-unresolved motion, verdict-on-quit, `Ctrl+E` editor compose.

**State at entry close:** dive/cycle committed; `docs/design-tui-review-first.md` gate red with 8 blocking threads awaiting Eric.

---

## 2026-08-05 (night, cont.) — Focus-follow verified by Eric; replies now expand inline

**What we did:** Eric ran the interactive TUI pass (screenshot) — focus-follows-cursor works in the wild. His immediate ask: expand the *whole thread* at the focused line, replies included, not just the root + a reply count. Shipped.

- `[win]` First human-verified TUI session: cursor on line 20 expanded the right group, gutter markers + resolved ✓ visible, keybar readable.
- `[win]` Expanded groups now render full threads: replies nested recursively with dimmed author/time meta lines, all text word-wrapped to the sidebar width (screenshot showed text running off the edge — fixed).
- `[win]` 3 new rendering tests: replies inline in the focused group only, wrapping bound, recursive nesting. TUI suite now 9 tests.
- `[idea]` Next TUI step per the redesign doc: dive-in-to-reply from the expanded group (reply without leaving line-select mode).

**State at entry close:** all green; sidebar is now a real review pane — glance at the focused line's full conversation, no thread-diving needed.

---

## 2026-08-05 (night) — Foundation: commits, CI, TUI markers + first TUI/MCP tests

**What we did:** Committed the day's arc (2 commits, each buildable), added CI, improved the document-pane gutter, and closed the two zero-test gaps.

- `[win]` `pkg/mcp` now has real tests: a live client↔server session over the SDK's in-memory transport covers tool registration (17), snake_case round-trip, gate lifecycle, human-zone refusal, and reanchor. The `serve-mcp` startup panic class is now caught by `go test`, not by a human.
- `[win]` `pkg/tui` has its first tests (6): gutter marker variants, grouping/badges, focus-follows-cursor, blocking/`~fuzzy` markers, document-order sorting — pure rendering functions made this cheap.
- `[win]` Gutter markers now carry review state: `⛔N` for lines with unresolved blocking threads, `💬N` unresolved count (roots only — was inflated by replies), quiet `✓` for fully-resolved lines; decided suggestions no longer count as open.
- `[win]` CI (GitHub Actions): build, vet, test, plus the full review-flow smoke (validate → seed → gate exit 10 → resolve → signoff → gate exit 0) — dry-run locally before committing.
- `[friction]` Interactive TUI verification still pending a human pass — rendering is tested, feel is not.

**State at entry close:** all packages tested and green; interactive TUI check + SDD integration recipes are next.

---

## 2026-08-05 (evening) — MCP 2026-07-28 spec + dead-code trim

**What we did:** Upgraded the MCP go-sdk v1.1.0 → v1.7.0-pre.1 (the 2026-07-28 stateless-spec beta) and trimmed the CLI to the recommended flow (agent drafts → seed → human `view` review → agent processes → gate unblocks → implement).

- `[win]` SDK upgrade needed zero code changes (same module path, no package split); server now negotiates both 2025-06-18 and 2026-07-28 clients — verified over stdio, 17 tools on both.
- `[win]` Removed: `batch-accept`, `status`, `reattach`, `cleanup` commands; `export`/`publish` usage text (commands never existed); deprecated `ValidateSidecar`/`ValidateAndArchiveIfStale`/`ArchiveStaleSidecar`; dead conflict-detection half of positions.go; all stale LLM/`ask`/pkg-llm references in CLAUDE.md (package was already gone). Command surface: 17 → 17 tools MCP-side, 18 → 14 CLI commands, all in the flow.
- `[win]` Full flow re-verified end-to-end on the trimmed binary: validate → seed → gate exit 10 → resolve+signoff → gate exit 0 → suggest/accept.
- `[friction]` CLI vs MCP JSON drift resurfaced in testing (`is_suggestion` missing CLI-side; added) — the design doc's "one serializer" unification is only half-done: CLI `list` still has its own struct.
- `[idea]` Handle-based `comments_request_review` (durable review-request handle in the sidecar + `check_review` polling tool) — aligns with the sessionless spec direction and survives agent restarts; deferred, still open.
- `[idea]` Pin SDK to stable v1.7.0 when released (currently on -pre.1).

**State at entry close:** all tests pass, both MCP protocol versions verified, working tree holds review layer + templates + v2.1 anchoring + spec upgrade + trim, uncommitted.

---

## 2026-08-05 (later) — Implemented the approved anchoring design (v2.1)

**What we did:** Implemented `docs/design-anchoring-refactor.md`: content anchors + re-anchor cascade, no more snap-to-heading, short base36 IDs, `comments_reanchor` MCP tool, MCP/CLI schema unification, and the G3 focus-follows-cursor sidebar (grouped by line, count badges, expanded group tracks the cursor in line-select mode).

- `[win]` Cascade verified end-to-end: comment followed its sentence through a section insertion (exact) and a rewording (fuzzy, labeled `anchor_confidence: fuzzy` in output); orphaning still catches truly-deleted targets.
- `[win]` Old sidecars (long IDs, no anchors) backfilled anchors transparently on load; the design doc's own review threads survived unchanged and the gate stayed green.
- `[win]` "Section moved" stderr spam replaced with one summary line ("N comment(s) re-anchored") — journal friction item closed same-day.
- `[win]` Anchor capture folded into `UpdateCommentSection`, which every creation path (CLI/batch/MCP/TUI/seed) already calls — one hook covered all sites.
- `[friction]` TUI focus-follow implemented but only verified by compilation + code review, not interactively — needs a human `comments view` pass (test coverage for pkg/tui is still zero).
- `[idea]` ID prefix matching stayed out (per review decision); revisit only if short IDs still feel unwieldy in practice.

**State at entry close:** all tests pass; anchoring/IDs/reanchor/schema verified E2E via CLI and MCP; TUI awaiting human verification with `comments view`.

---

## 2026-08-05 — First full review loop on our own design doc

**What we did:** Reviewed the tool hands-on (CLI + MCP), built the review gate layer (`gate`/`signoff`/`--blocking`), the template layer (`template`/`validate`/`seed` + zones), then dogfooded everything: drafted `docs/design-anchoring-refactor.md` under the `design-doc` template, ran three human↔agent review rounds through comment threads, and drove the gate from red (8 blocking) to green (approved).

**Wins:**

- `[win]` Anchored feedback beat chat. Every human reply landed on a specific thread — the agent never guessed what "make it concise" referred to. Seven replies mapped 1:1 to quality dimensions.
- `[win]` Seeded criteria worked as review rails: human walked the checklist instead of free-form skimming; every criterion got an explicit verdict.
- `[win]` Section re-anchoring survived two full document rewrites — all threads followed their sections.
- `[win]` The negotiation history is now a decision record: 11 threads in the sidecar capture what was asked, what changed, and why — including two decided design questions.
- `[win]` Zone enforcement held: agent structurally could not close human-decision threads over MCP.
- `[win]` Human feedback improved the *template* mid-flight (Risks criterion tightened to "no boilerplate" + 200-word cap) — templates as living encodings of reviewer philosophy.

**Friction:**

- `[friction]` Thread pile-up on headings: seeded threads + snap-to-heading stacked multiple threads on one line; human couldn't scan them in `view` without opening each. → became goal G3 (focus-follows-cursor review view) in the anchoring design doc.
- `[friction]` Mis-threaded reply: with stacked threads, one human reply landed on the wrong thread (fuzzy-matching Q got the human-TODO answer). Same root cause as pile-up.
- `[friction]` Snap-to-heading discards line precision at add time (`--line 7` → stored as line 3). → core motivation of the approved anchoring refactor.
- `[friction]` 19-digit timestamp IDs are unreadable/untypeable; humans needed copy-paste for every `resolve`/`reply`. → short display IDs in the refactor.
- `[friction]` `--section` requires the full root path ("Doc Title > Intro > X"); docs' short-form examples don't work. → open TODO in the design doc.
- `[friction]` `accept --preview` dumps the whole document instead of a focused diff.
- `[friction]` Accepted suggestions linger as unresolved threads in default `list`. → open TODO in the design doc.
- `[friction]` "Info: Section moved" messages spam stderr on every load after edits (12 lines for one batch-reply); useful signal, wrong volume.

**Ideas born this session:**

- `[idea]` Agent-assisted anchor migration (Eric): the agent that edits a doc should migrate the anchors it displaced (`comments_reanchor` MCP tool + required post-edit skill step); load-time cascade becomes the safety net. Adopted into the approved design.
- `[idea]` Focus-follows-cursor sidebar (Eric): navigating the doc line-by-line auto-expands that line's threads for glanceable review. Adopted as G3.
- `[idea]` Human-TODO checklist section in docs (Eric): explicit callout of decisions waiting on the human, complementing inline NEEDS CLARIFICATION markers. Adopted in the design doc's Unresolved Questions format.
- `[idea]` Review-first TUI redesign deserves its own design doc (scoped out of the anchoring refactor).

**Also fixed along the way:** `serve-mcp` startup panic (bad jsonschema struct tags); MCP section-based addressing was unimplemented; MCP comments lacked section metadata.

**State at entry close:** anchoring design doc approved (gate green); 2 non-blocking human-zone threads awaiting Eric's resolve + `signoff`. Implementation queue: content anchors + cascade, `comments_reanchor`, short display IDs, focus-follows-cursor view, schema unification.

---
