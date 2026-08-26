package markdown

import (
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	content := "---\ntype: Research\ntags: [agents, review]\ncomments:\n  template: research-deep\n---\n# Title\n"
	fm, present, err := ParseFrontmatter(content)
	if err != nil || !present {
		t.Fatalf("ParseFrontmatter() = present %v, err %v", present, err)
	}
	if got := FrontmatterString(fm.Values, "type"); got != "Research" {
		t.Fatalf("type = %q", got)
	}
	comments, ok := fm.Values["comments"].(map[string]any)
	if !ok || FrontmatterString(comments, "template") != "research-deep" {
		t.Fatalf("comments metadata = %#v", fm.Values["comments"])
	}
	if got := strings.Join(FrontmatterStrings(fm.Values, "tags"), ","); got != "agents,review" {
		t.Fatalf("tags = %q", got)
	}
}

func TestMaskFrontmatterPreservesLines(t *testing.T) {
	content := "---\ntype: Research\n---\n# Title\nbody\n"
	masked := MaskFrontmatter(content)
	if strings.Count(masked, "\n") != strings.Count(content, "\n") {
		t.Fatalf("line count changed: %q", masked)
	}
	if !strings.HasPrefix(masked, "\n\n\n# Title") {
		t.Fatalf("frontmatter not masked: %q", masked)
	}
	structure := ParseDocument(content)
	if len(structure.Sections) != 1 || structure.Sections[0].StartLine != 4 {
		t.Fatalf("heading lines changed: %#v", structure.Sections)
	}
}

func TestParseFrontmatterRejectsUnclosedOrInvalidYAML(t *testing.T) {
	for _, content := range []string{"---\ntype: Research\n", "---\ntype: [\n---\n# T\n"} {
		if _, present, err := ParseFrontmatter(content); !present || err == nil {
			t.Fatalf("wanted present frontmatter error for %q, got present=%v err=%v", content, present, err)
		}
	}
}
