package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDocumentContext(t *testing.T) {
	root := writeTestBundle(t)
	research, err := CreateBundleDocument(NewDocumentOptions{Name: "agent-surface", Template: "research-deep", StartDir: root})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := CreateBundleDocument(NewDocumentOptions{Name: "agent-surface", Template: "plan", StartDir: root, From: research.Path})
	if err != nil {
		t.Fatal(err)
	}
	context, err := BuildDocumentContext(plan.Path, ContextOptions{For: "drafting", IncludeBody: true})
	if err != nil {
		t.Fatal(err)
	}
	if context.Document.Template != "plan" || len(context.Related) != 1 {
		t.Fatalf("context = %#v", context)
	}
	if got := context.Related[0]; got.Relation != "informed_by" || got.Document.Path != "research/agent-surface.md" {
		t.Fatalf("related = %#v", got)
	}

	backlinks, err := BuildDocumentContext(research.Path, ContextOptions{For: "review"})
	if err != nil {
		t.Fatal(err)
	}
	if len(backlinks.Backlinks) != 1 || backlinks.Backlinks[0].Document.Path != "plans/agent-surface.md" {
		t.Fatalf("backlinks = %#v", backlinks.Backlinks)
	}
}

func TestCoverageScoutContextNeverIncludesBody(t *testing.T) {
	root := writeTestBundle(t)
	created, err := CreateBundleDocument(NewDocumentOptions{Name: "blind", Template: "research-deep", StartDir: root})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(created.Path)
	content := strings.Replace(string(data), "## Research Question\n", "## Research Question\n\nQ1. What is missing?\n", 1)
	if err := os.WriteFile(created.Path, []byte(content+"SECRET DRAFT BODY\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	context, err := BuildDocumentContext(created.Path, ContextOptions{For: "coverage-scout", IncludeBody: true, IncludeThreads: true})
	if err != nil {
		t.Fatal(err)
	}
	if context.Document.Body != "" || len(context.Document.Threads) != 0 {
		t.Fatalf("draft-blind context leaked body or threads: %#v", context.Document)
	}
	if context.Document.Focus != "Q1. What is missing?" || strings.Contains(context.Document.Focus, "SECRET") {
		t.Fatalf("coverage focus = %q", context.Document.Focus)
	}
	if filepath.Base(context.Document.Path) != "blind.md" {
		t.Fatalf("unexpected path: %s", context.Document.Path)
	}
}
