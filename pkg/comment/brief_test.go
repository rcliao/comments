package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const briefFixture = `---
comments:
    template: brief
type: Brief
---

# Cache Policy

## Why

Reads are slow.

## What

### Outcome

Reads take under 50ms.

### Not doing

- No write path changes.

### Invariants

- Only a human records a verdict.

## Shape

- Data model: an index on reads.

## Checks

- ` + "`go test ./...`" + ` passes.

## How

### Premises

- Reads dominate. pkg/x.go:1

### Status

- 2026-10-07 — **pending**
`

// approveBrief records a human verdict on briefFixture the way the TUI does.
func approveBrief(t *testing.T) (string, *DocumentWithComments) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "brief.md")
	if err := os.WriteFile(path, []byte(briefFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := &DocumentWithComments{Content: briefFixture}
	if _, err := RecordVerdict(path, doc, "rcliao", DecisionApproved, ""); err != nil {
		t.Fatal(err)
	}
	if got := doc.Reviews[0].Template; got != "brief" {
		t.Fatalf("verdict template = %q, want brief", got)
	}
	return path, doc
}

func freshnessAfter(t *testing.T, edit func(string) string) string {
	t.Helper()
	path, doc := approveBrief(t)
	doc.Content = edit(briefFixture)
	_ = path
	return PlanApprovalState(doc).Freshness
}

// Every way an agent could change what the human approved reads stale.
func TestBriefApprovalHumanSectionEditsGoStale(t *testing.T) {
	cases := map[string]func(string) string{
		"in place": func(s string) string { return strings.Replace(s, "under 50ms", "under 500ms", 1) },
		"deletion": func(s string) string { return strings.Replace(s, "- No write path changes.\n", "", 1) },
		"whole section": func(s string) string {
			return strings.Replace(s, "### Not doing\n\n- No write path changes.\n\n", "", 1)
		},
		"heading rename": func(s string) string { return strings.Replace(s, "### Not doing", "### Extras", 1) },
		"frontmatter":    func(s string) string { return strings.Replace(s, "template: brief", "template: mini", 1) },
		"status-looking": func(s string) string {
			return strings.Replace(s, "- Only a human records a verdict.\n", "- Only a human records a verdict.\n- 2026-10-07 — **done**\n  - Summary: drop the auth layer.\n", 1)
		},
		"shape":  func(s string) string { return strings.Replace(s, "an index on reads", "a new table", 1) },
		"checks": func(s string) string { return strings.Replace(s, "` passes.", "` passes, or is skipped.", 1) },
		"why": func(s string) string {
			return strings.Replace(s, "Reads are slow.", "Reads are slow and writes too.", 1)
		},
		"new heading":      func(s string) string { return strings.Replace(s, "## How\n", "## Checks\n\n- none.\n\n## How\n", 1) },
		"how hides a zone": func(s string) string { return strings.Replace(s, "### Premises", "## What\n\n### Premises", 1) },
		// Headings CommonMark renders that the section parser does not see.
		"indented h2 in how": func(s string) string { return s + "\n   ## Checks\n\n- skip all checks\n" },
		"setext h2 in how":   func(s string) string { return s + "\nChecks\n------\n\n- skip all checks\n" },
		// A blank line decides how markdown renders; removing one is an edit.
		"blank lines collapsed": func(s string) string {
			return strings.Replace(s, "### Not doing\n\n- No write path changes.", "### Not doing\n- No write path changes.", 1)
		},
		"preamble": func(s string) string {
			return strings.Replace(s, "# Cache Policy\n", "# Cache Policy\n\n> Scope also includes deleting templates.\n", 1)
		},
	}
	for name, edit := range cases {
		if got := freshnessAfter(t, edit); got != "stale" {
			t.Errorf("%s: freshness = %q, want stale", name, got)
		}
	}
}

// The agent's own section is free: How edits, Status entries and blank lines
// never stale the approval.
func TestBriefApprovalHowEditsStayCurrent(t *testing.T) {
	cases := map[string]func(string) string{
		"premise": func(s string) string {
			return strings.Replace(s, "- Reads dominate.", "- Reads dominate, measured.", 1)
		},
		"status": func(s string) string {
			return s + "- 2026-10-08 — **active**\n  - Summary: Building.\n  - Evidence: tests\n  - Next: more.\n"
		},
		"new how subsection": func(s string) string {
			return strings.Replace(s, "### Status", "### Changes\n\n- pkg/x.go\n\n### Status", 1)
		},
		"rule after a blank": func(s string) string { return s + "\n---\n\nMore notes.\n" },
		"extra blank lines":  func(s string) string { return strings.Replace(s, "## Checks\n\n", "## Checks\n\n\n\n", 1) },
	}
	for name, edit := range cases {
		if got := freshnessAfter(t, edit); got != "current" {
			t.Errorf("%s: freshness = %q, want current", name, got)
		}
	}
}

// The zones are the built-in template's. A project template planted where
// comments runs must not change what a verdict covers, nor flip freshness.
func TestBriefApprovalIgnoresProjectTemplates(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".comments", "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	builtin, err := builtinTemplates.ReadFile("templates/brief.yaml")
	if err != nil {
		t.Fatal(err)
	}
	planted := strings.Replace(string(builtin), "  - heading: \"Checks\"\n    required: true\n    tier: 3\n    zone: human", "  - heading: \"Checks\"\n    required: true\n    tier: 3", 1)
	if planted == string(builtin) {
		t.Fatal("fixture did not remove the Checks zone")
	}
	if err := os.WriteFile(filepath.Join(dir, ".comments", "templates", "brief.yaml"), []byte(planted), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	_, doc := approveBrief(t)
	doc.Content = strings.Replace(briefFixture, "` passes.", "` passes, or is skipped.", 1)
	if got := PlanApprovalState(doc).Freshness; got != "stale" {
		t.Fatalf("checks edit under a planted template = %q, want stale", got)
	}
}
