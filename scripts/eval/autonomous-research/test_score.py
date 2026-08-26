import unittest

from score import FIXED_CASE_IDS, score


def run(variant, coverage, unsupported, execution, threads, passes, plan_coverage=None):
    if plan_coverage is None:
        plan_coverage = coverage
    result = {
        "variant": variant,
        "research": f"artifacts/{variant}-research.md",
        "plan": f"artifacts/{variant}-plan.md",
        "metrics": {
            "source_question_coverage": {"covered": coverage, "total": 4},
            "unsupported_claims": unsupported,
            "plan_research_coverage": {"covered": plan_coverage, "total": 10},
            "execution_questions": execution,
            "human_open_threads": threads,
            "semantic_passes": passes,
        },
    }
    if variant == "treatment":
        result["trace"] = "traces/treatment.json"
        result["termination_reason"] = "converged_no_surviving_gap"
    return result


def payload(cases):
    return {
        "eval": "autonomous-research-convergence",
        "date": "2026-08-25",
        "model": "model",
        "reasoning": "high",
        "revision": "deadbeef",
        "context_hash": "a" * 64,
        "token_budget": 1000,
        "time_budget_seconds": 60,
        "cases": cases,
    }


class ScoreTest(unittest.TestCase):
    def test_promotes_majority_improvement_without_regression(self):
        cases = []
        for case_id, pair in zip(FIXED_CASE_IDS, ((1, 4), (2, 3), (1, 3), (2, 4), (4, 4))):
            cases.append({
                "id": case_id,
                "runs": [run("baseline", pair[0], 2, 3, 5, 1), run("treatment", pair[1], 1, 2, 4, 2)],
                "blind_human_preference": "treatment",
            })
        got = score(payload(cases))
        self.assertEqual(got["recommendation"], "promote_to_dogfood")
        self.assertEqual(got["coverage_improved_cases"], 4)
        self.assertEqual(got["treatment_semantic_passes"]["max"], 2)

    def test_blocks_unsupported_claim_regression(self):
        cases = []
        for case_id in FIXED_CASE_IDS:
            cases.append({
                "id": case_id,
                "runs": [run("baseline", 2, 0, 2, 3, 1), run("treatment", 4, 1, 1, 2, 2)],
                "blind_human_preference": "treatment",
            })
        got = score(payload(cases))
        self.assertEqual(got["recommendation"], "do_not_promote")
        self.assertFalse(got["guardrails"]["no_unsupported_claim_regression"])

    def test_requires_all_five_fixed_cases(self):
        cases = []
        for case_id in FIXED_CASE_IDS[:4]:
            cases.append({
                "id": case_id,
                "runs": [run("baseline", 2, 0, 2, 3, 1), run("treatment", 4, 0, 1, 2, 2)],
                "blind_human_preference": "treatment",
            })
        got = score(payload(cases))
        self.assertEqual(got["recommendation"], "do_not_promote")
        self.assertFalse(got["guardrails"]["five_fixed_cases"])

    def test_blocks_plan_coverage_regression_separately_from_discovery(self):
        cases = []
        for case_id in FIXED_CASE_IDS:
            cases.append({
                "id": case_id,
                "runs": [
                    run("baseline", 2, 0, 2, 3, 1, plan_coverage=9),
                    run("treatment", 4, 0, 1, 2, 2, plan_coverage=8),
                ],
                "blind_human_preference": "treatment",
            })
        got = score(payload(cases))
        self.assertEqual(got["coverage_improved_cases"], 5)
        self.assertEqual(got["recommendation"], "do_not_promote")
        self.assertFalse(got["guardrails"]["no_plan_research_coverage_regression"])

    def test_requires_paired_variants(self):
        with self.assertRaisesRegex(ValueError, "exactly one baseline"):
            score(payload([{"id": "bad", "runs": [run("baseline", 2, 0, 0, 0, 1)]}]))

    def test_requires_treatment_trace_and_termination(self):
        cases = []
        for case_id in FIXED_CASE_IDS:
            treatment = run("treatment", 4, 0, 1, 2, 2)
            treatment.pop("trace")
            cases.append({
                "id": case_id,
                "runs": [run("baseline", 2, 0, 2, 3, 1), treatment],
                "blind_human_preference": "treatment",
            })
        with self.assertRaisesRegex(ValueError, "trace must be a non-empty artifact path"):
            score(payload(cases))


if __name__ == "__main__":
    unittest.main()
