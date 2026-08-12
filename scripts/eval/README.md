# Template evals

Empirical checks that the template system's format/length constraints produce
distilled quality, not truncated or corrupted text.
Method and results live here; `check-doc.py` is the mechanical layer
(structure, style, citation checks) that `validate` complements.

## Harness pattern (LLM writers + LLM judges)

Every experiment follows the same shape:

1. **Fixture**: a task answerable from a small, fixed code surface
   (canonical: "how does `comments gate` decide its exit code?" from
   `pkg/comment/gate.go` + `cmd/comments/gate.go`), with a golden fact list
   extracted from that code (`fixtures/golden-facts-gate.md`).
2. **Writers**: fresh-context subagents produce the doc under the condition
   being tested (a cap, a format rule, an edit instruction).
   Two independent runs per condition minimum — consistency across runs is
   itself a signal.
3. **Judges**: fresh-context subagents score semantically (never regex):
   - *checklist judge* — per golden fact: covered / absent / **mangled**
     (present but a dropped qualifier changed the meaning — the truncation
     artifact worth hunting);
   - *rubric judge* — no fact list; reads the source itself and scores
     accuracy/completeness/clarity/review-efficiency, then ranks.
     Rubric judges also DISCOVER facts the golden list missed — use them to
     refresh the fixture, then the cheap checklist for repeated scoring.
4. **Log**: results append to `logs/` as JSON — scores are for the loop,
   not for humans to review (decided 2026-08-07).

## Results so far (logs/cap-pilot-2026-08-11.json)

Seven experiments on the gate fixture, headline findings:

- **Caps distill, they don't truncate**: at or above the content's
  information load a cap is free (10/10 facts at 120w vs 633w uncapped);
  models drop whole peripheral facts by choice, never chop mid-meaning.
- **The danger zone is slightly-too-tight**: caps at ~1/6-1/12 of the load
  produced fluent qualifier-dropping corruption; absurdly tight caps
  degraded gracefully back to the single core fact.
- **Edit-in-place beats fresh regeneration**: a reviewed capped doc is
  compressed memory of prior editorial decisions — edits absorbed new facts
  at 13/13 while fresh rewrites lost 3 originals and inverted one.
  Rewrite the prose, keep the doc.
- **Saturation evicts by importance, silently**: overloaded docs behave like
  an importance-ranked cache (no corruption at 2.25x load), but evictions
  are invisible — the newest reviewer feedback displaces mid-age nuance.
  Candidate rule: agents must declare evictions ("dropped X to seat Y").
- **Citations cost ~1 fact per 120w** (11-12% of budget) — argues for
  excluding file:line tokens from word-cap counting.
- **Sentence caps are protective**: short-sentence rules produced zero
  mangles where free sentences squeezed qualifiers into falsehoods;
  omission (visible) beats corruption (invisible).

## Real-template round (logs/template-guardrails-2026-08-12.json)

Second fixture (`fixtures/golden-facts-anchoring.md`) is mechanism narrative
rather than enumerable rules, so cap and format effects can be compared across
content types.

- **Format must follow section PURPOSE, not house style.** Lookup and narrative
  select different formats and the gap is wide: a table scored 5/5 on
  navigability where prose scored 2, but prose was the ONLY format that carried
  forward-looking invariants ("preserve that separation or context will stop
  winning"). Bullets carried the most facts per word and the fewest errors —
  the right default for mixed sections.
- **Each format loses what its shape has no slot for**, and the losses are
  disjoint: tables drop definitional setup (a table cannot define its own
  subject), bullets drop the failure branch of a rule whose bullet is already
  spent, prose drops enumerable constants. A table also mangles by omission —
  a cell must be short and unconditional, so its exceptions vanish. Prescribing
  a table without a "what breaks if you change it" column is what loses
  rationale.
- **Per-subsection caps: null result, reported as such.** The 150-word
  per-finding cap never bound (largest finding 135), so the cannibalization it
  guards against never occurred and pooled sections self-balanced. Do not cite
  this run as evidence for the current `max_subsection_words` values — rerun
  with a section cap tight enough that findings actually compete.

## Rerunning / extending

There is no runner script yet — experiments are orchestrated from a Claude
session (writers and judges as subagents, prompts recorded in the session).
To extend: add a fixture + golden list under `fixtures/`, keep two runs per
condition, always include a mangle category in judging, and append a new
top-level key to the current log file (or start a new dated one).
