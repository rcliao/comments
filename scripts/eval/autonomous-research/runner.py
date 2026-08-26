#!/usr/bin/env python3
"""Validate a treatment run and emit its deterministic convergence trace.

Agent runtimes produce the versioned coverage and evidence envelopes. This
runner never calls a model or judges prose; it validates the independent-role
contract, records recursive Qn decisions, and derives convergence or visible
three-pass exhaustion.
"""

import argparse
import json
import re
import sys
from pathlib import Path


RUN_SCHEMA = "convergence-run/v1"
TRACE_SCHEMA = "convergence-trace/v1"
COVERAGE_SCHEMA = "coverage-judgment/v1"
EVIDENCE_SCHEMA = "evidence-judgment/v1"
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")


class ContractError(ValueError):
    """The supplied evaluation envelope violates the frozen protocol."""


def object_at(value, context):
    if not isinstance(value, dict):
        raise ContractError(f"{context} must be an object")
    return value


def array_at(value, context):
    if not isinstance(value, list):
        raise ContractError(f"{context} must be an array")
    return value


def nonempty_string(value, context):
    if not isinstance(value, str) or not value.strip():
        raise ContractError(f"{context} must be a non-empty string")
    return value


def exact_keys(value, required, optional, context):
    missing = sorted(set(required) - set(value))
    extra = sorted(set(value) - set(required) - set(optional))
    if missing:
        raise ContractError(f"{context} missing fields: {missing}")
    if extra:
        raise ContractError(f"{context} has unknown fields: {extra}")


def load_cases(path):
    with path.open(encoding="utf-8") as handle:
        payload = json.load(handle)
    payload = object_at(payload, "cases file")
    if payload.get("schema_version") != "autonomous-research-cases/v1":
        raise ContractError("cases file uses an unsupported schema_version")
    cases = array_at(payload.get("cases"), "cases")
    by_id = {}
    for index, case in enumerate(cases, start=1):
        case = object_at(case, f"cases[{index}]")
        case_id = nonempty_string(case.get("id"), f"cases[{index}].id")
        if case_id in by_id:
            raise ContractError(f"duplicate case id: {case_id}")
        allowlist = array_at(case.get("source_allowlist"), f"{case_id}.source_allowlist")
        if not allowlist or any(not isinstance(path, str) or not path for path in allowlist):
            raise ContractError(f"{case_id}.source_allowlist needs non-empty paths")
        by_id[case_id] = case
    return by_id


def validate_provenance(value):
    value = object_at(value, "provenance")
    required = {
        "model",
        "reasoning",
        "revision",
        "context_hash",
        "token_budget",
        "time_budget_seconds",
    }
    exact_keys(value, required, set(), "provenance")
    for name in ("model", "reasoning", "revision", "context_hash"):
        nonempty_string(value[name], f"provenance.{name}")
    if not SHA256_RE.fullmatch(value["context_hash"]):
        raise ContractError("provenance.context_hash must be a lowercase SHA-256 digest")
    for name in ("token_budget", "time_budget_seconds"):
        if not isinstance(value[name], int) or isinstance(value[name], bool) or value[name] <= 0:
            raise ContractError(f"provenance.{name} must be a positive integer")
    return value


def validate_questions(value):
    questions = array_at(value, "initial_questions")
    if not questions:
        raise ContractError("initial_questions must not be empty")
    for index, question_id in enumerate(questions, start=1):
        if question_id != f"Q{index}":
            raise ContractError("initial_questions must be contiguous from Q1")
    return list(questions)


def validate_judge(value, context):
    value = object_at(value, context)
    exact_keys(value, {"id", "runtime"}, set(), context)
    nonempty_string(value["id"], f"{context}.id")
    nonempty_string(value["runtime"], f"{context}.runtime")
    return value


def validate_citations(value, allowlist, context, source_root=None):
    citations = array_at(value, context)
    if not citations:
        raise ContractError(f"{context} must contain at least one citation")
    for index, citation in enumerate(citations, start=1):
        item_context = f"{context}[{index}]"
        citation = object_at(citation, item_context)
        exact_keys(citation, {"path", "start_line", "end_line"}, set(), item_context)
        path = nonempty_string(citation["path"], f"{item_context}.path")
        if path not in allowlist:
            raise ContractError(f"{item_context}.path is outside the fixed source allowlist: {path}")
        start = citation["start_line"]
        end = citation["end_line"]
        if not isinstance(start, int) or isinstance(start, bool) or start < 1:
            raise ContractError(f"{item_context}.start_line must be a positive integer")
        if not isinstance(end, int) or isinstance(end, bool) or end < start:
            raise ContractError(f"{item_context}.end_line must be at least start_line")
        if source_root is not None:
            source = source_root / path
            if not source.is_file():
                raise ContractError(f"{item_context}.path does not exist at the pinned revision: {path}")
            try:
                with source.open(encoding="utf-8") as handle:
                    line_count = sum(1 for _ in handle)
            except (OSError, UnicodeError) as exc:
                raise ContractError(f"{item_context}.path cannot be read: {path}: {exc}") from exc
            if end > line_count:
                raise ContractError(
                    f"{item_context} ends at line {end}, past {path}'s {line_count} lines"
                )
    return citations


def validate_coverage(value, case_id, pass_number, allowlist, source_root=None):
    value = object_at(value, f"pass {pass_number} coverage")
    exact_keys(value, {"schema_version", "case_id", "pass", "judge", "candidates"}, set(), f"pass {pass_number} coverage")
    if value["schema_version"] != COVERAGE_SCHEMA:
        raise ContractError(f"pass {pass_number} coverage uses unsupported schema_version")
    if value["case_id"] != case_id or value["pass"] != pass_number:
        raise ContractError(f"pass {pass_number} coverage identity does not match its run")
    judge = validate_judge(value["judge"], f"pass {pass_number} coverage.judge")
    candidates = array_at(value["candidates"], f"pass {pass_number} coverage.candidates")
    seen = set()
    for index, candidate in enumerate(candidates, start=1):
        context = f"pass {pass_number} coverage.candidates[{index}]"
        candidate = object_at(candidate, context)
        exact_keys(candidate, {"id", "question", "expected_answer", "citations"}, set(), context)
        candidate_id = nonempty_string(candidate["id"], f"{context}.id")
        if not re.fullmatch(r"C[1-9][0-9]*", candidate_id):
            raise ContractError(f"{context}.id must look like C1")
        if candidate_id in seen:
            raise ContractError(f"pass {pass_number} has duplicate coverage candidate {candidate_id}")
        seen.add(candidate_id)
        nonempty_string(candidate["question"], f"{context}.question")
        nonempty_string(candidate["expected_answer"], f"{context}.expected_answer")
        validate_citations(candidate["citations"], allowlist, f"{context}.citations", source_root)
    return judge, candidates


def validate_reconciliation(value, candidates, questions, pass_number):
    decisions = array_at(value, f"pass {pass_number} reconciliation")
    by_candidate = {}
    for index, decision in enumerate(decisions, start=1):
        context = f"pass {pass_number} reconciliation[{index}]"
        decision = object_at(decision, context)
        exact_keys(
            decision,
            {"candidate_id", "decision", "reason", "thread_id"},
            {"question_id"},
            context,
        )
        candidate_id = nonempty_string(decision["candidate_id"], f"{context}.candidate_id")
        if candidate_id in by_candidate:
            raise ContractError(f"pass {pass_number} reconciles {candidate_id} more than once")
        if decision["decision"] not in {"accepted", "rejected"}:
            raise ContractError(f"{context}.decision must be accepted or rejected")
        nonempty_string(decision["reason"], f"{context}.reason")
        nonempty_string(decision["thread_id"], f"{context}.thread_id")
        by_candidate[candidate_id] = decision

    candidate_ids = [candidate["id"] for candidate in candidates]
    missing = sorted(set(candidate_ids) - set(by_candidate))
    extra = sorted(set(by_candidate) - set(candidate_ids))
    if missing or extra:
        raise ContractError(
            f"pass {pass_number} reconciliation must cover every candidate; missing={missing}, extra={extra}"
        )

    accepted = []
    next_number = len(questions) + 1
    for candidate_id in candidate_ids:
        decision = by_candidate[candidate_id]
        if decision["decision"] == "accepted":
            expected = f"Q{next_number}"
            if decision.get("question_id") != expected:
                raise ContractError(
                    f"pass {pass_number} accepted {candidate_id} must assign next question {expected}"
                )
            accepted.append(expected)
            questions.append(expected)
            next_number += 1
        elif "question_id" in decision:
            raise ContractError(f"pass {pass_number} rejected {candidate_id} must not assign question_id")
    return decisions, accepted


def validate_evidence(value, case_id, pass_number, allowlist, source_root=None):
    value = object_at(value, f"pass {pass_number} evidence")
    exact_keys(value, {"schema_version", "case_id", "pass", "judge", "findings"}, set(), f"pass {pass_number} evidence")
    if value["schema_version"] != EVIDENCE_SCHEMA:
        raise ContractError(f"pass {pass_number} evidence uses unsupported schema_version")
    if value["case_id"] != case_id or value["pass"] != pass_number:
        raise ContractError(f"pass {pass_number} evidence identity does not match its run")
    judge = validate_judge(value["judge"], f"pass {pass_number} evidence.judge")
    findings = array_at(value["findings"], f"pass {pass_number} evidence.findings")
    seen = set()
    blockers = []
    for index, finding in enumerate(findings, start=1):
        context = f"pass {pass_number} evidence.findings[{index}]"
        finding = object_at(finding, context)
        exact_keys(
            finding,
            {"id", "claim", "verdict", "blocking", "reason", "citations"},
            {"thread_id"},
            context,
        )
        finding_id = nonempty_string(finding["id"], f"{context}.id")
        if not re.fullmatch(r"E[1-9][0-9]*", finding_id):
            raise ContractError(f"{context}.id must look like E1")
        if finding_id in seen:
            raise ContractError(f"pass {pass_number} has duplicate evidence finding {finding_id}")
        seen.add(finding_id)
        nonempty_string(finding["claim"], f"{context}.claim")
        if finding["verdict"] not in {"supported", "contradicted", "overstated"}:
            raise ContractError(f"{context}.verdict is invalid")
        if not isinstance(finding["blocking"], bool):
            raise ContractError(f"{context}.blocking must be a boolean")
        if finding["verdict"] == "supported" and finding["blocking"]:
            raise ContractError(f"{context} cannot mark a supported claim blocking")
        if finding["verdict"] != "supported":
            nonempty_string(finding.get("thread_id"), f"{context}.thread_id")
        nonempty_string(finding["reason"], f"{context}.reason")
        validate_citations(finding["citations"], allowlist, f"{context}.citations", source_root)
        if finding["blocking"]:
            blockers.append(finding_id)
    return judge, findings, blockers


def validate_artifact(value, context):
    value = object_at(value, context)
    exact_keys(value, {"path", "sha256"}, set(), context)
    nonempty_string(value["path"], f"{context}.path")
    digest = nonempty_string(value["sha256"], f"{context}.sha256")
    if not SHA256_RE.fullmatch(digest):
        raise ContractError(f"{context}.sha256 must be a lowercase SHA-256 digest")
    return value


def build_trace(run, cases_by_id, source_root=None):
    run = object_at(run, "run")
    exact_keys(
        run,
        {"schema_version", "run_id", "case_id", "variant", "max_passes", "provenance", "initial_questions", "passes"},
        set(),
        "run",
    )
    if run["schema_version"] != RUN_SCHEMA:
        raise ContractError(f"unsupported run schema_version: {run['schema_version']!r}")
    run_id = nonempty_string(run["run_id"], "run_id")
    case_id = nonempty_string(run["case_id"], "case_id")
    if case_id not in cases_by_id:
        raise ContractError(f"unknown case_id: {case_id}")
    if run["variant"] != "treatment":
        raise ContractError("convergence traces are required only for the treatment variant")
    max_passes = run["max_passes"]
    if max_passes != 3:
        raise ContractError("max_passes must remain frozen at 3")
    provenance = validate_provenance(run["provenance"])
    questions = validate_questions(run["initial_questions"])
    initial_questions = list(questions)
    passes = array_at(run["passes"], "passes")
    if not passes:
        raise ContractError("passes must not be empty")
    if len(passes) > max_passes:
        raise ContractError("run contains more than the three-pass cap")

    allowlist = set(cases_by_id[case_id]["source_allowlist"])
    trace_passes = []
    termination = None
    total_accepted = 0
    total_evidence_issues = 0

    for expected_number, semantic_pass in enumerate(passes, start=1):
        if termination is not None:
            raise ContractError(f"run includes pass {expected_number} after {termination}")
        context = f"pass {expected_number}"
        semantic_pass = object_at(semantic_pass, context)
        exact_keys(
            semantic_pass,
            {"number", "research", "analysis_ready", "coverage", "reconciliation", "evidence"},
            set(),
            context,
        )
        if isinstance(semantic_pass["number"], bool) or semantic_pass["number"] != expected_number:
            raise ContractError(f"{context}.number must be {expected_number}")
        research = validate_artifact(semantic_pass["research"], f"{context}.research")
        if not isinstance(semantic_pass["analysis_ready"], bool):
            raise ContractError(f"{context}.analysis_ready must be a boolean")
        questions_before = list(questions)
        coverage_judge, candidates = validate_coverage(
            semantic_pass["coverage"], case_id, expected_number, allowlist, source_root
        )
        decisions, accepted = validate_reconciliation(
            semantic_pass["reconciliation"], candidates, questions, expected_number
        )
        evidence_judge, findings, blockers = validate_evidence(
            semantic_pass["evidence"], case_id, expected_number, allowlist, source_root
        )
        if coverage_judge["id"] == evidence_judge["id"]:
            raise ContractError(f"{context} must use independent coverage and evidence judges")

        reasons = []
        if accepted:
            reasons.append("accepted_questions")
        if blockers:
            reasons.append("blocking_evidence")
        if not semantic_pass["analysis_ready"]:
            reasons.append("analysis_not_ready")
        clean = not reasons
        if clean:
            outcome = "converged"
            termination = "converged_no_surviving_gap"
        elif expected_number == max_passes:
            outcome = "cap_exhausted"
            termination = "three_pass_cap_exhausted"
        else:
            outcome = "continue"

        total_accepted += len(accepted)
        total_evidence_issues += sum(1 for finding in findings if finding["verdict"] != "supported")
        trace_passes.append({
            "number": expected_number,
            "research": research,
            "analysis_ready": semantic_pass["analysis_ready"],
            "questions_before": questions_before,
            "coverage_judge": coverage_judge,
            "coverage_candidates": candidates,
            "reconciliation": decisions,
            "accepted_questions": accepted,
            "evidence_judge": evidence_judge,
            "evidence_findings": findings,
            "blocking_evidence": blockers,
            "questions_after": list(questions),
            "continue_reasons": reasons,
            "outcome": outcome,
        })

    if termination is None:
        raise ContractError("run ended before convergence or the three-pass cap")

    final = trace_passes[-1]
    return {
        "schema_version": TRACE_SCHEMA,
        "run_id": run_id,
        "case_id": case_id,
        "variant": "treatment",
        "provenance": provenance,
        "source_allowlist": sorted(allowlist),
        "initial_questions": initial_questions,
        "passes": trace_passes,
        "final_questions": list(questions),
        "termination_reason": termination,
        "survivors": {
            "accepted_questions": final["accepted_questions"] if termination != "converged_no_surviving_gap" else [],
            "blocking_evidence": final["blocking_evidence"],
            "analysis_not_ready": not final["analysis_ready"],
        },
        "metrics": {
            "semantic_passes": len(trace_passes),
            "accepted_question_count": total_accepted,
            "evidence_issue_count": total_evidence_issues,
        },
    }


def main():
    here = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("run", type=Path, help="completed convergence-run/v1 JSON")
    parser.add_argument("--cases", type=Path, default=here / "cases.json", help="frozen case definitions")
    parser.add_argument("--output", type=Path, help="new immutable trace path (must not already exist)")
    args = parser.parse_args()
    try:
        with args.run.open(encoding="utf-8") as handle:
            run = json.load(handle)
        trace = build_trace(run, load_cases(args.cases), args.cases.resolve().parents[3])
        rendered = json.dumps(trace, indent=2) + "\n"
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            with args.output.open("x", encoding="utf-8") as handle:
                handle.write(rendered)
        else:
            sys.stdout.write(rendered)
    except (OSError, json.JSONDecodeError, ContractError) as exc:
        print(f"runner error: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
