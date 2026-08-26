#!/usr/bin/env python3
"""Score a paired baseline/treatment autonomous-research evaluation.

The scorer is deliberately mechanical. Human or independent-agent judges fill
the per-run metrics from blinded artifacts; this script validates the paired
shape, computes directional guardrails, and reports whether the result is ready
for the final blinded human go/no-go call.
"""

import argparse
import json
import re
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

FIXED_CASE_IDS = (
    "gate-contract",
    "anchor-survival",
    "template-contract",
    "review-baseline",
    "citation-resolution",
)
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")


def coverage_rate(value):
    covered = value["covered"]
    total = value["total"]
    if (not isinstance(covered, int) or isinstance(covered, bool) or
            not isinstance(total, int) or isinstance(total, bool) or total <= 0):
        raise ValueError("coverage metrics require integer covered and positive total")
    if covered < 0 or covered > total:
        raise ValueError("coverage covered must be between zero and total")
    return covered / total


def validate_run(run, case_id):
    variant = run.get("variant")
    for field in ("research", "plan"):
        if not isinstance(run.get(field), str) or not run[field].strip():
            raise ValueError(f"{case_id}/{variant}: {field} must be a non-empty artifact path")
    if variant == "treatment":
        if not isinstance(run.get("trace"), str) or not run["trace"].strip():
            raise ValueError(f"{case_id}/treatment: trace must be a non-empty artifact path")
        if run.get("termination_reason") not in {
            "converged_no_surviving_gap",
            "three_pass_cap_exhausted",
        }:
            raise ValueError(f"{case_id}/treatment: invalid termination_reason")
    metrics = run.get("metrics", {})
    missing = [name for name in REQUIRED_METRICS if name not in metrics]
    if missing:
        raise ValueError(f"{case_id}/{run.get('variant')}: missing metrics {missing}")
    source = coverage_rate(metrics["source_question_coverage"])
    plan = coverage_rate(metrics["plan_research_coverage"])
    if metrics["source_question_coverage"]["total"] != 4:
        raise ValueError(f"{case_id}/{variant}: source coverage total must match four frozen golden questions")
    for name in REQUIRED_METRICS[1:]:
        if name == "plan_research_coverage":
            continue
        if not isinstance(metrics[name], int) or isinstance(metrics[name], bool) or metrics[name] < 0:
            raise ValueError(f"{case_id}/{run.get('variant')}: {name} must be a non-negative integer")
    if metrics["semantic_passes"] == 0:
        raise ValueError(f"{case_id}/{variant}: semantic_passes must be positive")
    return source, plan


def score(payload):
    required_text_provenance = ("date", "model", "reasoning", "revision", "context_hash")
    required_budget_provenance = ("token_budget", "time_budget_seconds")
    if payload.get("eval") != "autonomous-research-convergence":
        raise ValueError("unexpected eval identifier")
    for field in required_text_provenance:
        if not isinstance(payload.get(field), str) or not payload[field].strip():
            raise ValueError(f"{field} must be a non-empty string")
    if not SHA256_RE.fullmatch(payload["context_hash"]):
        raise ValueError("context_hash must be a lowercase SHA-256 digest")
    for field in required_budget_provenance:
        if not isinstance(payload.get(field), int) or isinstance(payload[field], bool) or payload[field] <= 0:
            raise ValueError(f"{field} must be a positive integer")
    cases = payload.get("cases", [])
    if not isinstance(cases, list):
        raise ValueError("cases must be an array")

    details = []
    baseline_totals = {"unsupported_claims": 0, "execution_questions": 0, "human_open_threads": 0}
    treatment_totals = {"unsupported_claims": 0, "execution_questions": 0, "human_open_threads": 0}
    treatment_passes = []
    baseline_plan_coverage = []
    treatment_plan_coverage = []
    preferences = {"baseline": 0, "treatment": 0, "tie": 0, "unscored": 0}

    for case in cases:
        case_id = case.get("id", "<missing-id>")
        runs = case.get("runs", [])
        by_variant = {run.get("variant"): run for run in runs}
        if set(by_variant) != {"baseline", "treatment"} or len(runs) != 2:
            raise ValueError(f"{case_id}: need exactly one baseline and one treatment run")

        base_source, base_plan = validate_run(by_variant["baseline"], case_id)
        treatment_source, treatment_plan = validate_run(by_variant["treatment"], case_id)
        for name in baseline_totals:
            baseline_totals[name] += by_variant["baseline"]["metrics"][name]
            treatment_totals[name] += by_variant["treatment"]["metrics"][name]
        treatment_passes.append(by_variant["treatment"]["metrics"]["semantic_passes"])
        baseline_plan_coverage.append(base_plan)
        treatment_plan_coverage.append(treatment_plan)

        preference = case.get("blind_human_preference", "unscored")
        if preference not in preferences:
            raise ValueError(f"{case_id}: invalid blind_human_preference {preference!r}")
        preferences[preference] += 1
        details.append({
            "id": case_id,
            "baseline_source_coverage": round(base_source, 4),
            "treatment_source_coverage": round(treatment_source, 4),
            "source_coverage_improved": treatment_source > base_source,
            "baseline_plan_coverage": round(base_plan, 4),
            "treatment_plan_coverage": round(treatment_plan, 4),
            "blind_human_preference": preference,
        })

    # The approved pilot freezes five cases and requires four improvements.
    # Do not silently weaken this to a moving majority if a case is omitted.
    required_improvements = 4
    improved = sum(1 for case in details if case["source_coverage_improved"])
    case_ids = [case["id"] for case in details]
    enough_cases = len(case_ids) == len(FIXED_CASE_IDS) and set(case_ids) == set(FIXED_CASE_IDS)
    guardrails = {
        "five_fixed_cases": enough_cases,
        "source_coverage_improves_on_four": improved >= required_improvements,
        "no_plan_research_coverage_regression": sum(treatment_plan_coverage) >= sum(baseline_plan_coverage),
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
        "plan_research_coverage_totals": {
            "baseline": round(sum(baseline_plan_coverage), 4),
            "treatment": round(sum(treatment_plan_coverage), 4),
        },
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
