# Autonomous research convergence eval

This paired eval decides whether the new research loop is ready for dogfood.
It does not call a model or silently grade prose; any agent runtime can execute
the fixed conditions, and judges fill one common result envelope.

## Protocol

1. Use the three tasks in `cases.json` (add at most two before the pilot).
2. For each task, run one fresh agent with `baseline.md` and one with
   `treatment.md`. Pin the same model, reasoning setting, repository revision,
   task text, source allowlist, and time/token budget.
3. Rename the two artifact pairs to random A/B labels before judging.
   Judges do not see condition prompts, variant names, or thread authors.
4. A source-derived judge scores `source_question_coverage` against each case's
   golden questions and counts `unsupported_claims` by checking evidence.
5. A cold implementation judge receives only the plan and records
   `execution_questions`; `comments analyze --against` supplies
   `plan_research_coverage`.
6. Record the open threads the human would face, semantic passes, and the
   blinded human preference. Fill `result-example.json`'s shape.
7. Run `python3 score.py results.json`. Phase 4 is eligible only when the
   recommendation is `promote_to_dogfood`.

The automated rule requires coverage improvement on a majority of 3–5 cases,
with no aggregate regression in unsupported claims, execution questions, or
human open-thread burden. Raw scores remain in the result file; the small pilot
estimates variance and observed pass cost rather than inventing significance.
