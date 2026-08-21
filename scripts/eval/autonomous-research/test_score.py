import unittest

from score import score


def run(variant, coverage, unsupported, execution, threads, passes):
    return {
        "variant": variant,
        "metrics": {
            "source_question_coverage": {"covered": coverage, "total": 10},
            "unsupported_claims": unsupported,
            "plan_research_coverage": {"covered": coverage, "total": 10},
            "execution_questions": execution,
            "human_open_threads": threads,
            "semantic_passes": passes,
        },
    }


class ScoreTest(unittest.TestCase):
    def test_promotes_majority_improvement_without_regression(self):
        cases = []
        for index, pair in enumerate(((6, 9), (7, 8), (8, 8)), start=1):
            cases.append({
                "id": f"c{index}",
                "runs": [run("baseline", pair[0], 2, 3, 5, 1), run("treatment", pair[1], 1, 2, 4, 2)],
                "blind_human_preference": "treatment",
            })
        got = score({"cases": cases})
        self.assertEqual(got["recommendation"], "promote_to_dogfood")
        self.assertEqual(got["coverage_improved_cases"], 2)
        self.assertEqual(got["treatment_semantic_passes"]["max"], 2)

    def test_blocks_unsupported_claim_regression(self):
        cases = []
        for index in range(3):
            cases.append({
                "id": f"c{index}",
                "runs": [run("baseline", 6, 0, 2, 3, 1), run("treatment", 9, 1, 1, 2, 2)],
                "blind_human_preference": "treatment",
            })
        got = score({"cases": cases})
        self.assertEqual(got["recommendation"], "do_not_promote")
        self.assertFalse(got["guardrails"]["no_unsupported_claim_regression"])

    def test_requires_paired_variants(self):
        with self.assertRaisesRegex(ValueError, "exactly one baseline"):
            score({"cases": [{"id": "bad", "runs": [run("baseline", 5, 0, 0, 0, 1)]}]})


if __name__ == "__main__":
    unittest.main()
