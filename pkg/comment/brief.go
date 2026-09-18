package comment

import (
	"fmt"
	"strings"
)

// BriefSection is one section of a writing brief. Zero-valued constraints are
// omitted: an agent should read the rules that apply, not a wall of defaults.
type BriefSection struct {
	Heading             string   `json:"heading"`
	Required            bool     `json:"required,omitempty"`
	MaxWords            int      `json:"max_words,omitempty"`
	MinSubsections      int      `json:"min_subsections,omitempty"`
	MaxSubsections      int      `json:"max_subsections,omitempty"`
	MaxSubsectionWords  int      `json:"max_subsection_words,omitempty"`
	EnumeratesQuestions bool     `json:"enumerates_questions,omitempty"`
	AnswersQuestions    bool     `json:"answers_questions,omitempty"`
	Tier                int      `json:"tier,omitempty"`
	Zone                string   `json:"zone,omitempty"`
	ReviewCriteria      []string `json:"review_criteria,omitempty"`
}

// TemplateBrief is a template as a writing brief: what to write, how much, who
// owns it, and how it will be judged. It is the one shape both surfaces hand an
// agent — `comments template show` prints Text(), `comments context` embeds the
// struct — so the brief an agent drafts from cannot differ by surface.
type TemplateBrief struct {
	Template          string         `json:"template"`
	Description       string         `json:"description,omitempty"`
	MaxWords          int            `json:"max_words,omitempty"`
	Sections          []BriefSection `json:"sections"`
	ReadingPath       []ReadingTier  `json:"reading_path,omitempty"`
	MaxSentenceWords  int            `json:"max_sentence_words,omitempty"`
	MaxParagraphWords int            `json:"max_paragraph_words,omitempty"`
	MarkerPrefix      string         `json:"marker_prefix"`
	MaxMarkers        int            `json:"max_markers,omitempty"`
	// Guidance is the prose an agent needs that no field carries: how to break
	// lines, what a marker is for. Validate checks the numbers, not these.
	Guidance []string `json:"guidance,omitempty"`
}

// Brief renders the template as a writing brief.
func (t *Template) Brief() *TemplateBrief {
	b := &TemplateBrief{
		Template: t.Name, Description: t.Description, MaxWords: t.Doc.MaxWords,
		Sections: []BriefSection{}, ReadingPath: t.ReadingPath(),
		MaxSentenceWords: t.Doc.Style.MaxSentenceWords, MaxParagraphWords: t.Doc.Style.MaxParagraphWords,
		MarkerPrefix: t.MarkerPrefix(), MaxMarkers: t.Markers.Max,
	}
	for _, s := range t.Sections {
		b.Sections = append(b.Sections, BriefSection{
			Heading: s.Heading, Required: s.Required, MaxWords: s.MaxWords,
			MinSubsections: s.MinSubsections, MaxSubsections: s.MaxSubsections, MaxSubsectionWords: s.MaxSubsectionWords,
			EnumeratesQuestions: s.EnumeratesQuestions, AnswersQuestions: s.AnswersQuestions,
			Tier: s.Tier, Zone: s.Zone, ReviewCriteria: s.ReviewCriteria,
		})
	}
	if b.MaxParagraphWords > 0 || b.MaxSentenceWords > 0 {
		b.Guidance = append(b.Guidance,
			"One purpose per paragraph; use bullet lists where you are listing things. Lead each sentence with the claim.",
			"Break lines at sentence or clause boundaries, never at column width: the tool is line-addressed, so one sentence per line means a comment anchors to a sentence and an edit does not reflow the lines below.")
	}
	if b.MaxMarkers > 0 {
		b.Guidance = append(b.Guidance, fmt.Sprintf(
			"Use %s ...] where you would otherwise guess at the human's intent, and add a blocking comment for each. Stay under the cap: decide the rest yourself and record those as assumptions in the doc.", b.MarkerPrefix))
	}
	return b
}

// Text renders the brief for a terminal.
func (b *TemplateBrief) Text() string {
	var out strings.Builder
	fmt.Fprintf(&out, "Template: %s\n%s\n\n", b.Template, b.Description)
	if b.MaxWords > 0 {
		fmt.Fprintf(&out, "Document cap: %d words\n\n", b.MaxWords)
	}
	out.WriteString("Sections:\n")
	for _, s := range b.Sections {
		flags := []string{}
		if s.Required {
			flags = append(flags, "required")
		}
		if s.MaxWords > 0 {
			flags = append(flags, fmt.Sprintf("max %d words", s.MaxWords))
		}
		// Subsection bounds belong in the brief, not only in the validator:
		// an agent cannot follow a cap it is never shown, and a rule that
		// only surfaces as a late violation gets satisfied by trimming
		// rather than by writing tighter in the first place.
		switch {
		case s.MinSubsections > 0 && s.MaxSubsections > 0:
			flags = append(flags, fmt.Sprintf("%d-%d subsections", s.MinSubsections, s.MaxSubsections))
		case s.MinSubsections > 0:
			flags = append(flags, fmt.Sprintf(">=%d subsections", s.MinSubsections))
		case s.MaxSubsections > 0:
			flags = append(flags, fmt.Sprintf("<=%d subsections", s.MaxSubsections))
		}
		if s.MaxSubsectionWords > 0 {
			flags = append(flags, fmt.Sprintf("max %d words each", s.MaxSubsectionWords))
		}
		if s.EnumeratesQuestions {
			flags = append(flags, "enumerate sub-questions Q1., Q2., ...")
		}
		if s.AnswersQuestions {
			flags = append(flags, "tag each subsection [Q1]")
		}
		if s.Zone != "" {
			flags = append(flags, "zone: "+s.Zone)
		}
		if s.Tier > 0 {
			flags = append(flags, fmt.Sprintf("tier %d", s.Tier))
		}
		fmt.Fprintf(&out, "  ## %s", s.Heading)
		if len(flags) > 0 {
			fmt.Fprintf(&out, "  [%s]", strings.Join(flags, ", "))
		}
		out.WriteString("\n")
		for _, c := range s.ReviewCriteria {
			fmt.Fprintf(&out, "     ✓ %s\n", c)
		}
	}

	// The per-section `tier N` flag says where a section sits; the path says
	// how far a reader with limited time gets, which is why tiers exist.
	if len(b.ReadingPath) > 0 {
		out.WriteString("\nReading path (stop after any tier; each adds to the ones before):\n")
		for _, rt := range b.ReadingPath {
			budget := fmt.Sprintf("<=%d words so far", rt.CumulativeMaxWords)
			if rt.Uncapped {
				budget = fmt.Sprintf("%d+ words so far, some sections uncapped", rt.CumulativeMaxWords)
			}
			fmt.Fprintf(&out, "  tier %d: %s  [%s]\n", rt.Tier, strings.Join(rt.Sections, ", "), budget)
		}
	}

	// Style caps shape how the doc reads; an agent that only meets word
	// budgets will write walls unless told the shape too.
	if b.MaxParagraphWords > 0 || b.MaxSentenceWords > 0 {
		out.WriteString("\nWriting style (checked by validate):\n")
		if b.MaxParagraphWords > 0 {
			fmt.Fprintf(&out, "  Paragraphs: at most %d words.\n", b.MaxParagraphWords)
		}
		if b.MaxSentenceWords > 0 {
			fmt.Fprintf(&out, "  Sentences: at most %d words.\n", b.MaxSentenceWords)
		}
	}
	if b.MaxMarkers > 0 {
		fmt.Fprintf(&out, "\nAmbiguity markers: at most %d %s ...] per document (each is reported by validate).\n", b.MaxMarkers, b.MarkerPrefix)
	}
	if len(b.Guidance) > 0 {
		out.WriteString("\nGuidance:\n")
		for _, g := range b.Guidance {
			fmt.Fprintf(&out, "  - %s\n", g)
		}
	}
	return out.String()
}
