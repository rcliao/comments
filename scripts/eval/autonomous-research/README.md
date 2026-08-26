# Autonomous research convergence eval

This paired eval decides whether the research loop is ready for dogfood. Agent
runtimes execute the fixed conditions and versioned judge prompts; repository
scripts validate the resulting traces and score the blinded pairs. No script
calls a model or silently grades prose.

## Frozen protocol

1. Use all five tasks in `cases.json`. Do not change tasks, allowlists, golden
   questions, judge prompts, or promotion rules after generation starts.
2. For each task, run one fresh agent with `baseline.md` and one with
   `treatment.md`. Pin the same model, reasoning setting, repository revision,
   task text, source allowlist, context hash, and time/token budget.
3. During each treatment semantic pass, use independent runtimes with
   `judges/coverage-v1.md` and `judges/evidence-v1.md`. The coverage judge never
   sees the draft; the evidence judge never sees coverage output.
4. Reconcile every coverage candidate into an accepted or rejected comment
   thread. Accepted candidates receive the next contiguous Qn. Record one
   completed `convergence-run/v1` envelope; `run-example.json` shows the shape.
5. Emit an immutable trace (the output path must not exist):

   ```bash
   python3 runner.py completed-run.json --output traces/case-treatment.json
   ```

   The runner validates source boundaries, judge independence, citations,
   contestable reconciliation, recursive Qn assignment, clean convergence, and
   visible three-pass exhaustion. It never decides whether prose is correct.
6. Rename the artifact pairs to random A/B labels before final judging. Judges
   do not see condition prompts, variant names, label mapping, or thread authors.
7. A source-derived judge scores `source_question_coverage` against each case's
   golden questions and counts `unsupported_claims` by checking cited evidence.
   A cold implementation judge receives only the plan and records
   `execution_questions`; `comments analyze --against` supplies
   `plan_research_coverage`.
8. Record the open threads the human would face and blinded human preference in
   the five-case shape in `result-example.json`, then run:

   ```bash
   python3 score.py results.json
   ```

## Promotion rule

Treatment must improve source-derived coverage on at least four of the five
fixed tasks. In aggregate it may not regress plan-to-research coverage,
unsupported claims, execution questions, or human open-thread burden. A passing
mechanical result becomes `promote_to_dogfood` only after every A/B pair has a
blinded human preference. Raw judgments and traces remain the reviewable source
of truth; the pilot estimates observed cost rather than statistical significance.

## Contract tests

```bash
python3 -m unittest discover -s scripts/eval/autonomous-research -p 'test_*.py' -v
python3 runner.py run-example.json
python3 score.py result-example.json
```
