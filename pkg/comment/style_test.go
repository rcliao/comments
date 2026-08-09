package comment

import (
	"strings"
	"testing"
)

func styleTpl() *Template {
	return &Template{
		Name: "t",
		Doc:  TemplateDocRules{Style: TemplateStyle{MaxSentenceWords: 10, MaxParagraphWords: 25}},
	}
}

func styleRules(v []Violation) []string {
	out := []string{}
	for _, x := range v {
		out = append(out, x.Rule)
	}
	return out
}

func TestStyleFlagsLongParagraphAndSentence(t *testing.T) {
	long := "# D\n\n" + strings.Repeat("word ", 30) + "\n"
	got := styleRules(ValidateTemplate(long, styleTpl()))
	if len(got) == 0 {
		t.Fatal("a 30-word paragraph must trip a 25-word cap")
	}
}

func TestStyleSkipsListsHeadingsAndFences(t *testing.T) {
	content := "# A heading that is quite long but headings are not prose blocks here\n\n" +
		"- " + strings.Repeat("word ", 30) + "\n\n" +
		"```\n" + strings.Repeat("word ", 30) + "\n```\n"
	if v := ValidateTemplate(content, styleTpl()); len(v) != 0 {
		t.Errorf("headings, lists and fences are not prose: %v", styleRules(v))
	}
}

// Naming the exact symbol should not cost more budget than writing vaguely.
func TestStyleCountsCodeSpanAsOneToken(t *testing.T) {
	// Nine plain words plus one long citation: 10 readable tokens, at the cap.
	content := "# D\n\nThe gate reads state and stops here at `pkg/comment/gate.go:39-71`.\n"
	if v := ValidateTemplate(content, styleTpl()); len(v) != 0 {
		t.Errorf("inline code spans must count as one token: %v", styleRules(v))
	}
}

// The marker's bracket syntax should not compete with its own content.
func TestStyleExemptsClarificationMarkers(t *testing.T) {
	content := "# D\n\n[NEEDS CLARIFICATION: " + strings.Repeat("word ", 30) + "]\n"
	for _, v := range ValidateTemplate(content, styleTpl()) {
		if v.Rule == "long_sentence" || v.Rule == "long_paragraph" {
			t.Errorf("markers are annotations, not prose: %+v", v)
		}
	}
}

func TestStyleOptOut(t *testing.T) {
	content := "# D\n\n" + strings.Repeat("word ", 200) + "\n"
	for _, v := range ValidateTemplate(content, &Template{Name: "t"}) {
		if strings.Contains(v.Rule, "long_") {
			t.Errorf("unset style caps must not fire: %+v", v)
		}
	}
}
