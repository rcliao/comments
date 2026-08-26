package comment

import "testing"

func TestParseDocumentMetadata(t *testing.T) {
	content := `---
type: Implementation Plan
title: Agent surface
description: Implements the reviewed agent-surface research.
tags: [agents, review]
status: draft
comments:
  template: plan
related:
  - path: ../research/agent-surface.md
    relation: informed_by
sources:
  - id: architecture
    resource: ../ARCHITECTURE.md
    title: Architecture
---
# Agent surface
`
	meta, err := ParseDocumentMetadata(content)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Type != "Implementation Plan" || meta.Template != "plan" || meta.Status != "draft" {
		t.Fatalf("metadata = %#v", meta)
	}
	if len(meta.Related) != 1 || meta.Related[0].Relation != "informed_by" {
		t.Fatalf("related = %#v", meta.Related)
	}
	if len(meta.Sources) != 1 || meta.Sources[0].ID != "architecture" {
		t.Fatalf("sources = %#v", meta.Sources)
	}
}

func TestValidateOKFMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
		rule string
	}{
		{"clean", "---\ntype: Research\nstatus: draft\n---\n# T\n", ""},
		{"missing", "# T\n", "missing_frontmatter"},
		{"type", "---\nstatus: draft\n---\n# T\n", "missing_type"},
		{"status", "---\ntype: Research\nstatus: done\n---\n# T\n", "invalid_status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			violations := ValidateOKFMetadata(tc.doc)
			if tc.rule == "" && len(violations) != 0 {
				t.Fatalf("unexpected violations: %#v", violations)
			}
			if tc.rule != "" && (len(violations) != 1 || violations[0].Rule != tc.rule) {
				t.Fatalf("violations = %#v, want %s", violations, tc.rule)
			}
		})
	}
}
