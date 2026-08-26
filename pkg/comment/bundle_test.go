package comment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestBundle(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".comments"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := `bundle: Test Knowledge
version: 1
okf_version: "0.2"
root: docs
collections:
  research:
    path: research
    type: Research
    templates: [research, research-deep]
  plans:
    path: plans
    type: Implementation Plan
    templates: [plan]
`
	if err := os.WriteFile(filepath.Join(root, ProjectBundleFile), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCreateBundleDocumentAndIndexes(t *testing.T) {
	root := writeTestBundle(t)
	result, err := CreateBundleDocument(NewDocumentOptions{Name: "agent-surface", Template: "research-deep", StartDir: root})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{"type: Research", "template: research-deep", "status: draft", "## Research Question"} {
		if !strings.Contains(content, want) {
			t.Errorf("document missing %q:\n%s", want, content)
		}
	}
	if !SidecarExists(result.Path) {
		t.Fatal("new document has no review sidecar")
	}
	index, err := os.ReadFile(filepath.Join(root, "docs", "research", "index.md"))
	if err != nil || !strings.Contains(string(index), "[Agent Surface](agent-surface.md)") {
		t.Fatalf("collection index = %q, err %v", index, err)
	}
	rootIndex, err := os.ReadFile(filepath.Join(root, "docs", "index.md"))
	if err != nil || !strings.Contains(string(rootIndex), `okf_version: "0.2"`) {
		t.Fatalf("root index = %q, err %v", rootIndex, err)
	}
}

func TestResolveTemplateNameOrder(t *testing.T) {
	root := writeTestBundle(t)
	docPath := filepath.Join(root, "docs", "research", "one.md")
	content := "---\ntype: Research\ncomments:\n  template: research-deep\n---\n# T\n"
	if got, source, err := ResolveTemplateName(docPath, content, "plan", "research"); err != nil || got != "plan" || source != "explicit" {
		t.Fatalf("explicit resolution = %q %q %v", got, source, err)
	}
	if got, source, err := ResolveTemplateName(docPath, content, "", "research"); err != nil || got != "research-deep" || source != "frontmatter" {
		t.Fatalf("frontmatter resolution = %q %q %v", got, source, err)
	}
	withoutFM := "# T\n"
	if got, source, err := ResolveTemplateName(docPath, withoutFM, "", "research"); err != nil || got != "research" || source != "sidecar" {
		t.Fatalf("sidecar resolution = %q %q %v", got, source, err)
	}
}
