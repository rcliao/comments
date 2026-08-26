package comment

import (
	"strings"
	"testing"
)

const planLedgerFixture = `---
comments:
  template: plan
type: Plan
---
# Plan
## Implementation Phases
### Phase 1
Stable intent.
#### Status
- 2026-08-25 — **pending**
  - Summary: Waiting.
  - Evidence: thread:one
  - Next: Begin.
- 2026-08-26 — **active**
  - Summary: Building it.
  - Evidence: pkg/comment/planstatus.go
  - Next: Test it.
**Success Criteria**
- automated: parser passes
### Phase 2
More intent.
**Success Criteria**
- manual: reviewer can follow it
`

func TestParsePlanStatusLatestWinsAndMissingWarns(t *testing.T) {
	parsed := ParsePlanStatus(planLedgerFixture)
	if len(parsed.Phases) != 2 {
		t.Fatalf("phases = %#v", parsed.Phases)
	}
	first := parsed.Phases[0]
	if first.State != "active" || first.HistoryCount != 2 || first.Latest.Next != "Test it." {
		t.Fatalf("first phase = %#v", first)
	}
	if !strings.Contains(first.SuccessCriteria, "parser passes") {
		t.Fatalf("success criteria = %q", first.SuccessCriteria)
	}
	if parsed.Phases[1].State != "untracked" || len(parsed.Phases[1].Warnings) != 1 {
		t.Fatalf("historical phase = %#v", parsed.Phases[1])
	}
	if violations := ValidatePlanStatus(planLedgerFixture); len(violations) != 0 {
		t.Fatalf("missing status should be warning-only: %#v", violations)
	}
}

func TestPlanStatusMalformedPresentLogFails(t *testing.T) {
	content := strings.Replace(planLedgerFixture, "  - Next: Test it.", "  - Nope: Test it.", 1)
	violations := ValidatePlanStatus(content)
	if len(violations) == 0 || violations[0].Rule != "invalid_status_log" {
		t.Fatalf("violations = %#v", violations)
	}
}

func TestPlanStatusIgnoresFencedExamplesAndRejectsDuplicateSections(t *testing.T) {
	fenced := strings.Replace(planLedgerFixture, "**Success Criteria**", "```md\n- 2026-08-26 — **done**\n  - Summary: Example only.\n```\n**Success Criteria**", 1)
	if got := ParsePlanStatus(fenced).Phases[0].HistoryCount; got != 2 {
		t.Fatalf("fenced example counted as status: %d", got)
	}
	duplicate := strings.Replace(planLedgerFixture, "**Success Criteria**", "#### Status\n\n**Success Criteria**", 1)
	if violations := ValidatePlanStatus(duplicate); len(violations) == 0 || !strings.Contains(violations[0].Message, "multiple Status") {
		t.Fatalf("duplicate status sections = %#v", violations)
	}
}

func TestPlanStatusEntryCountHasIndependentCap(t *testing.T) {
	var entries strings.Builder
	for i := 0; i <= PlanStatusMaxEntries; i++ {
		entries.WriteString("- 2026-08-26 — **active**\n  - Summary: Working.\n  - Evidence: test.\n  - Next: Continue.\n")
	}
	content := strings.Replace(planLedgerFixture,
		"- 2026-08-25 — **pending**\n  - Summary: Waiting.\n  - Evidence: thread:one\n  - Next: Begin.\n- 2026-08-26 — **active**\n  - Summary: Building it.\n  - Evidence: pkg/comment/planstatus.go\n  - Next: Test it.\n",
		entries.String(), 1)
	violations := ValidatePlanStatus(content)
	found := false
	for _, violation := range violations {
		found = found || violation.Rule == "status_entry_cap"
	}
	if !found {
		t.Fatalf("entry cap missing: %#v", violations)
	}
}

func TestPlanStatusDoesNotConsumeNormalWordBudget(t *testing.T) {
	template, err := LoadTemplate("plan")
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("progress ", 1100)
	content := strings.Replace(planLedgerFixture, "Building it.", long, 1)
	report := SectionWordReport(content, template)
	if report[0].Words >= 1000 {
		t.Fatalf("status leaked into document cap: %#v", report[0])
	}
	violations := ValidateTemplate(content, template)
	foundStatusCap := false
	for _, violation := range violations {
		if violation.Rule == "doc_over_length" || violation.Rule == "over_length" || violation.Rule == "subsection_over_length" {
			t.Fatalf("status leaked into normal cap: %#v", violation)
		}
		if violation.Rule == "status_entry_over_length" {
			foundStatusCap = true
		}
	}
	if !foundStatusCap {
		t.Fatalf("separate status cap missing: %#v", violations)
	}
}

func TestPlanIntentHashIgnoresStatusOnlyChanges(t *testing.T) {
	changedStatus := strings.Replace(planLedgerFixture, "Building it.", "Still building it.", 1)
	if PlanIntentHash(planLedgerFixture) != PlanIntentHash(changedStatus) {
		t.Fatal("status-only edit changed plan intent hash")
	}
	changedIntent := strings.Replace(planLedgerFixture, "Stable intent.", "Different stable intent.", 1)
	if PlanIntentHash(planLedgerFixture) == PlanIntentHash(changedIntent) {
		t.Fatal("stable intent edit did not change plan intent hash")
	}
	if len(strings.Split(MaskPlanStatusLog(planLedgerFixture), "\n")) != len(strings.Split(planLedgerFixture, "\n")) {
		t.Fatal("mask changed line count")
	}
}

func TestPlanImplementationApprovalAndAttention(t *testing.T) {
	doc := &DocumentWithComments{Content: planLedgerFixture, Threads: []*Comment{{
		ID: "guardrail", Line: 12, Priority: PriorityHigh, Status: StatusActive,
	}}}
	if got := BuildPlanImplementationContext(doc).OverallStatus; got != "unapproved" {
		t.Fatalf("without signoff = %q", got)
	}
	AddReviewRecord(doc, "human", DecisionApproved, "ship it", false)
	context := BuildPlanImplementationContext(doc)
	if context.Approval.Freshness != "current" || context.OverallStatus != "attention-needed" {
		t.Fatalf("context = %#v", context)
	}

	doc.Threads[0].Resolved = true
	doc.Content = strings.Replace(doc.Content, "Building it.", "Still building it.", 1)
	context = BuildPlanImplementationContext(doc)
	if context.Approval.Freshness != "current" || !context.Approval.FullDocumentChanged {
		t.Fatalf("status edit freshness = %#v", context.Approval)
	}

	doc.Content = strings.Replace(doc.Content, "Stable intent.", "Changed intent.", 1)
	if got := BuildPlanImplementationContext(doc); got.Approval.Freshness != "stale" || got.OverallStatus != "stale" {
		t.Fatalf("intent edit = %#v", got)
	}

	legacy := &DocumentWithComments{Content: planLedgerFixture, Reviews: []ReviewRecord{{Decision: DecisionApproved}}}
	if got := PlanApprovalState(legacy).Freshness; got != "unknown" {
		t.Fatalf("legacy freshness = %q", got)
	}
}

func TestPlanImplementationAlignedAndComplete(t *testing.T) {
	tracked := strings.Replace(planLedgerFixture, "More intent.\n**Success Criteria**", `More intent.
#### Status
- 2026-08-26 — **pending**
  - Summary: Waiting.
  - Evidence: test.
  - Next: Begin.
**Success Criteria**`, 1)
	doc := &DocumentWithComments{Content: tracked}
	AddReviewRecord(doc, "human", DecisionApproved, "approved", false)
	if got := BuildPlanImplementationContext(doc).OverallStatus; got != "aligned" {
		t.Fatalf("active approved plan = %q", got)
	}
	doc.Content = strings.Replace(doc.Content, "**active**", "**done**", 1)
	doc.Content = strings.Replace(doc.Content, "**pending**\n  - Summary: Waiting.\n  - Evidence: test.", "**done**\n  - Summary: Finished.\n  - Evidence: test.", 1)
	if got := BuildPlanImplementationContext(doc); got.Approval.Freshness != "current" || got.OverallStatus != "complete" {
		t.Fatalf("completed plan = %#v", got)
	}
}
