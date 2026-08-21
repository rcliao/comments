#!/usr/bin/env python3
"""Score a paired baseline/treatment autonomous-research evaluation.

The scorer is deliberately mechanical. Human or independent-agent judges fill
the per-run metrics from blinded artifacts; this script validates the paired
shape, computes directional guardrails, and reports whether the result is ready
for the final blinded human go/no-go call.
"""

import argparse
import json
import statistics
import sys

REQUIRED_METRICS = (
    "source_question_coverage",
    "unsupported_claims",
    "plan_research_coverage",
    "execution_questions",
    "human_open_threads",
    "semantic_passes",
)


def coverage_rate(value):
    covered = value["covered"]
    total = value["total"]
    if not isinstance(covered, int) or not isinstance(total, int) or total <= 0:
        raise ValueError("coverage metrics require integer covered and positive total")
    if covered < 0 or covered > total:
        raise ValueError("coverage covered must be between zero and total")
    return covered / total


def validate_run(run, case_id):
    metrics = run.get("metrics", {})
    missing = [name for name in REQUIRED_METRICS if name not in metrics]
    if missing:
        raise ValueError(f"{case_id}/{run.get('variant')}: missing metrics {missing}")
    source = coverage_rate(metrics["source_question_coverage"])
    plan = coverage_rate(metrics["plan_research_coverage"])
    for name in REQUIRED_METRICS[1:]:
        if name == "plan_research_coverage":
            continue
        if not isinstance(metrics[name], int) or metrics[name] < 0:
            raise ValueError(f"{case_id}/{run.get('variant')}: {name} must be a non-negative integer")
    return (source + plan) / 2


def score(payload):
    cases = payload.get("cases", [])
    if not isinstance(cases, list):
        raise ValueError("cases must be an array")

    details = []
    baseline_totals = {"unsupported_claims": 0, "execution_questions": 0, "human_open_threads": 0}
    treatment_totals = {"unsupported_claims": 0, "execution_questions": 0, "human_open_threads": 0}
    treatment_passes = []
    preferences = {"baseline": 0, "treatment": 0, "tie": 0, "unscored": 0}

    for case in cases:
        case_id = case.get("id", "<missing-id>")
        runs = case.get("runs", [])
        by_variant = {run.get("variant"): run for run in runs}
        if set(by_variant) != {"baseline", "treatment"} or len(runs) != 2:
            raise ValueError(f"{case_id}: need exactly one baseline and one treatment run")

        base_score = validate_run(by_variant["baseline"], case_id)
        treatment_score = validate_run(by_variant["treatment"], case_id)
        for name in baseline_totals:
            baseline_totals[name] += by_variant["baseline"]["metrics"][name]
            treatment_totals[name] += by_variant["treatment"]["metrics"][name]
        treatment_passes.append(by_variant["treatment"]["metrics"]["semantic_passes"])

        preference = case.get("blind_human_preference", "unscored")
        if preference not in preferences:
            raise ValueError(f"{case_id}: invalid blind_human_preference {preference!r}")
        preferences[preference] += 1
        details.append({
            "id": case_id,
            "baseline_coverage": round(base_score, 4),
            "treatment_coverage": round(treatment_score, 4),
            "coverage_improved": treatment_score > base_score,
            "blind_human_preference": preference,
        })

    required_improvements = len(cases) // 2 + 1
    improved = sum(1 for case in details if case["coverage_improved"])
    enough_cases = 3 <= len(cases) <= 5
    guardrails = {
        "three_to_five_cases": enough_cases,
        "coverage_improves_on_majority": improved >= required_improvements,
        "no_unsupported_claim_regression": treatment_totals["unsupported_claims"] <= baseline_totals["unsupported_claims"],
        "no_execution_question_regression": treatment_totals["execution_questions"] <= baseline_totals["execution_questions"],
        "no_human_review_burden_regression": treatment_totals["human_open_threads"] <= baseline_totals["human_open_threads"],
    }
    eligible = all(guardrails.values())
    all_blind_scored = preferences["unscored"] == 0 and len(cases) > 0
    human_supports = all_blind_scored and preferences["treatment"] >= preferences["baseline"]

    if not eligible:
        recommendation = "do_not_promote"
    elif not all_blind_scored:
        recommendation = "eligible_for_blind_human_review"
    elif human_supports:
        recommendation = "promote_to_dogfood"
    else:
        recommendation = "do_not_promote"

    return {
        "eval": "autonomous-research-convergence",
        "case_count": len(cases),
        "coverage_improved_cases": improved,
        "required_improved_cases": required_improvements,
        "guardrails": guardrails,
        "baseline_totals": baseline_totals,
        "treatment_totals": treatment_totals,
        "treatment_semantic_passes": {
            "values": treatment_passes,
            "median": statistics.median(treatment_passes) if treatment_passes else None,
            "max": max(treatment_passes) if treatment_passes else None,
        },
        "blind_preferences": preferences,
        "cases": details,
        "recommendation": recommendation,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("results", help="paired result JSON")
    args = parser.parse_args()
    try:
        with open(args.results, encoding="utf-8") as handle:
            payload = json.load(handle)
        print(json.dumps(score(payload), indent=2))
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        print(f"eval error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
