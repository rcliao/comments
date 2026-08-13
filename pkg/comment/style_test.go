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

// Citations must not compete with content for the style budget either — style
// caps share countWords with every other cap, so the exemption is uniform.
func TestStyleExemptsCitations(t *testing.T) {
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

// A column wrap ends on a word because the width ran out; a semantic break
// ends on punctuation because the thought did.
func TestStyleFlagsHardWrappedLines(t *testing.T) {
	tpl := &Template{Name: "t", Doc: TemplateDocRules{Style: TemplateStyle{MaxParagraphWords: 500}}}

	wrapped := "# D\n\nThe review gate reads the recorded thread state and then decides whether\nthe document may proceed to the next phase of the flow.\n"
	found := false
	for _, v := range ValidateTemplate(wrapped, tpl) {
		if v.Rule == "hard_wrapped_line" {
			found = true
		}
	}
	if !found {
		t.Error("a line broken mid-phrase at column width must be reported")
	}

	// Same prose, broken after the clause: legal.
	semantic := "# D\n\nThe review gate reads the recorded thread state,\nand then decides whether the document may proceed to the next phase.\n"
	for _, v := range ValidateTemplate(semantic, tpl) {
		if v.Rule == "hard_wrapped_line" {
			t.Errorf("a clause-boundary break is not a column wrap: %+v", v)
		}
	}
}

// Regression: "**Bold lead-in:**" starts with * but is prose, not a bullet.
// Matching the bare marker exempted every bold-led paragraph from every style
// check, which hid real violations for as long as the checks existed.
func TestStyleChecksBoldLedParagraphs(t *testing.T) {
	tpl := &Template{Name: "t", Doc: TemplateDocRules{Style: TemplateStyle{MaxParagraphWords: 10}}}
	content := "# D\n\n**Decision:** " + strings.Repeat("word ", 40) + "\n"
	found := false
	for _, v := range ValidateTemplate(content, tpl) {
		if v.Rule == "long_paragraph" {
			found = true
		}
	}
	if !found {
		t.Error("a bold-led paragraph is prose and must be checked")
	}

	// A real bullet still is one.
	bullet := "# D\n\n- " + strings.Repeat("word ", 40) + "\n"
	for _, v := range ValidateTemplate(bullet, tpl) {
		if v.Rule == "long_paragraph" {
			t.Errorf("a genuine list item must stay exempt: %+v", v)
		}
	}
}
