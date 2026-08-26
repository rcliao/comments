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

func TestCreateBundleDocumentBootstrapsDefaultAtRepoRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "work", "agent")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := CreateBundleDocument(NewDocumentOptions{Name: "default-path", Template: "plan", StartDir: nested})
	if err != nil {
		t.Fatal(err)
	}
	if !result.BundleCreated {
		t.Fatal("first comments new should report that it initialized the default bundle")
	}
	wantConfig := filepath.Join(root, ProjectBundleFile)
	if result.BundleConfig != wantConfig {
		t.Fatalf("bundle config = %s, want %s", result.BundleConfig, wantConfig)
	}
	wantDoc := filepath.Join(root, "docs", "artifacts", "plans", "default-path.md")
	if result.Path != wantDoc {
		t.Fatalf("document path = %s, want %s", result.Path, wantDoc)
	}
	config, err := os.ReadFile(wantConfig)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`okf_version: "0.2"`, "root: docs/artifacts", "research-deep", "design-doc"} {
		if !strings.Contains(string(config), want) {
			t.Errorf("default bundle missing %q:\n%s", want, config)
		}
	}

	second, err := CreateBundleDocument(NewDocumentOptions{Name: "second", Template: "mini", StartDir: nested})
	if err != nil {
		t.Fatal(err)
	}
	if second.BundleCreated {
		t.Fatal("existing default bundle must be reused, not rewritten")
	}
}

func TestEnsureBundleDoesNotReplaceInvalidConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, ProjectBundleFile)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	broken := []byte("bundle: [not valid\n")
	if err := os.WriteFile(configPath, broken, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := EnsureBundle(root); err == nil {
		t.Fatal("invalid explicit bundle should be reported, not replaced")
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(broken) {
		t.Fatalf("invalid bundle was overwritten: %q", after)
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
