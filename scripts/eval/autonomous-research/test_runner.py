import copy
import json
import unittest
from pathlib import Path

from runner import ContractError, build_trace, load_cases


HERE = Path(__file__).resolve().parent
CASES = load_cases(HERE / "cases.json")
CASE_ID = "gate-contract"
CITATION = {"path": "pkg/comment/gate.go", "start_line": 34, "end_line": 64}


def coverage(pass_number, candidates=None):
    return {
        "schema_version": "coverage-judgment/v1",
        "case_id": CASE_ID,
        "pass": pass_number,
        "judge": {"id": f"coverage-{pass_number}", "runtime": "test-runtime"},
        "candidates": candidates or [],
    }


def evidence(pass_number, findings=None):
    return {
        "schema_version": "evidence-judgment/v1",
        "case_id": CASE_ID,
        "pass": pass_number,
        "judge": {"id": f"evidence-{pass_number}", "runtime": "test-runtime"},
        "findings": findings or [],
    }


def semantic_pass(pass_number, *, candidates=None, reconciliation=None, findings=None, ready=True):
    return {
        "number": pass_number,
        "research": {"path": f"research-{pass_number}.md", "sha256": "a" * 64},
        "analysis_ready": ready,
        "coverage": coverage(pass_number, candidates),
        "reconciliation": reconciliation or [],
        "evidence": evidence(pass_number, findings),
    }


def run_with(passes):
    return {
        "schema_version": "convergence-run/v1",
        "run_id": "run-001",
        "case_id": CASE_ID,
        "variant": "treatment",
        "max_passes": 3,
        "provenance": {
            "model": "model",
            "reasoning": "high",
            "revision": "deadbeef",
            "context_hash": "b" * 64,
            "token_budget": 1000,
            "time_budget_seconds": 60,
        },
        "initial_questions": ["Q1"],
        "passes": passes,
    }


def candidate(candidate_id="C1"):
    return {
        "id": candidate_id,
        "question": "What qualifier is missing?",
        "expected_answer": "Strict mode also counts non-blocking work.",
        "citations": [copy.deepcopy(CITATION)],
    }


def accepted(candidate_id="C1", question_id="Q2"):
    return {
        "candidate_id": candidate_id,
        "decision": "accepted",
        "question_id": question_id,
        "reason": "This changes the complete answer.",
        "thread_id": "c-coverage-1",
    }


def blocker(finding_id="E1"):
    return {
        "id": finding_id,
        "claim": "Reviews authorize the current gate.",
        "verdict": "contradicted",
        "blocking": True,
        "reason": "The gate derives current state independently.",
        "thread_id": f"c-{finding_id.lower()}",
        "citations": [copy.deepcopy(CITATION)],
    }


class RunnerTest(unittest.TestCase):
    def test_records_recursive_question_then_clean_convergence(self):
        first = semantic_pass(1, candidates=[candidate()], reconciliation=[accepted()], ready=True)
        second = semantic_pass(2, ready=True)

        trace = build_trace(run_with([first, second]), CASES)

        self.assertEqual(trace["termination_reason"], "converged_no_surviving_gap")
        self.assertEqual(trace["passes"][0]["questions_before"], ["Q1"])
        self.assertEqual(trace["passes"][0]["accepted_questions"], ["Q2"])
        self.assertEqual(trace["passes"][1]["questions_before"], ["Q1", "Q2"])
        self.assertEqual(trace["final_questions"], ["Q1", "Q2"])
        self.assertEqual(trace["metrics"]["semantic_passes"], 2)

    def test_exposes_survivors_at_three_pass_cap(self):
        passes = [
            semantic_pass(index, findings=[blocker(f"E{index}")], ready=False)
            for index in range(1, 4)
        ]

        trace = build_trace(run_with(passes), CASES)

        self.assertEqual(trace["termination_reason"], "three_pass_cap_exhausted")
        self.assertEqual(trace["passes"][-1]["outcome"], "cap_exhausted")
        self.assertEqual(trace["survivors"]["blocking_evidence"], ["E3"])
        self.assertTrue(trace["survivors"]["analysis_not_ready"])

    def test_rejects_run_that_stops_before_a_terminal_condition(self):
        with self.assertRaisesRegex(ContractError, "ended before convergence"):
            build_trace(run_with([semantic_pass(1, ready=False)]), CASES)

    def test_requires_every_candidate_to_be_reconciled(self):
        with self.assertRaisesRegex(ContractError, "must cover every candidate"):
            build_trace(run_with([semantic_pass(1, candidates=[candidate()])]), CASES)

    def test_accepted_questions_must_be_sequential(self):
        bad = semantic_pass(1, candidates=[candidate()], reconciliation=[accepted(question_id="Q9")])
        with self.assertRaisesRegex(ContractError, "must assign next question Q2"):
            build_trace(run_with([bad]), CASES)

    def test_rejects_evidence_outside_fixed_allowlist(self):
        bad_candidate = candidate()
        bad_candidate["citations"][0]["path"] = "README.md"
        bad = semantic_pass(1, candidates=[bad_candidate], reconciliation=[accepted()])
        with self.assertRaisesRegex(ContractError, "outside the fixed source allowlist"):
            build_trace(run_with([bad]), CASES)

    def test_rejects_citation_past_source_end_at_pinned_revision(self):
        bad_candidate = candidate()
        bad_candidate["citations"][0].update({"start_line": 9999, "end_line": 9999})
        bad = semantic_pass(1, candidates=[bad_candidate], reconciliation=[accepted()])
        with self.assertRaisesRegex(ContractError, "past pkg/comment/gate.go"):
            build_trace(run_with([bad]), CASES, HERE.parents[2])

    def test_case_fixture_has_five_source_consistent_cases(self):
        with (HERE / "cases.json").open(encoding="utf-8") as handle:
            cases = json.load(handle)["cases"]
        self.assertEqual(len(cases), 5)
        for case in cases:
            allowlist = set(case["source_allowlist"])
            self.assertEqual(len(case["golden_questions"]), 4)
            for question in case["golden_questions"]:
                self.assertTrue(set(question["evidence"]).issubset(allowlist))


if __name__ == "__main__":
    unittest.main()
