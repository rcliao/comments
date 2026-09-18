package comment

import (
	"fmt"
	"strings"
)

// SuggestionSpec proposes replacing a range of lines. The range comes from
// exactly one of: StartLine (with optional EndLine), Section, or Anchor — the
// quoted first target line, whose range is as long as OriginalText.
type SuggestionSpec struct {
	Author       string
	Text         string
	StartLine    int
	EndLine      int
	Section      string
	Anchor       string
	OriginalText string
	ProposedText string
}

// SuggestResult reports the suggestion a call created.
type SuggestResult struct {
	SuggestionID string `json:"suggestion_id"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	SectionPath  string `json:"section_path,omitempty"`
}

// AddSuggestion validates, locates and appends a suggestion. It is the only
// path that creates one: the two adapters used to build suggestions
// separately, and the MCP one skipped the section metadata, so a suggestion
// made over MCP was located differently from one made on the CLI after the
// document changed.
func AddSuggestion(doc *DocumentWithComments, spec SuggestionSpec) (*SuggestResult, error) {
	switch {
	case spec.Author == "":
		return nil, fmt.Errorf("author is required")
	case spec.Text == "":
		return nil, fmt.Errorf("text is required")
	case spec.ProposedText == "":
		return nil, fmt.Errorf("proposed text is required")
	}
	given := 0
	for _, ok := range []bool{spec.StartLine != 0, spec.Section != "", spec.Anchor != ""} {
		if ok {
			given++
		}
	}
	if given == 0 {
		return nil, fmt.Errorf("one of a start line, a section or an anchor is required")
	}
	if given > 1 {
		return nil, fmt.Errorf("start line, section and anchor are mutually exclusive")
	}

	start, end := spec.StartLine, spec.EndLine
	if spec.Section != "" {
		if err := ValidateSectionPath(doc.Content, spec.Section); err != nil {
			return nil, err
		}
		var err error
		if start, end, err = ResolveSectionToLines(doc.Content, spec.Section, false); err != nil {
			return nil, err
		}
	}
	if spec.Anchor != "" {
		resolved, err := ResolveAnchorText(doc.Content, spec.Anchor)
		if err != nil {
			return nil, err
		}
		// The original text's line count defines the range; one line otherwise.
		start, end = resolved, resolved
		if spec.OriginalText != "" {
			end = resolved + strings.Count(strings.TrimRight(spec.OriginalText, "\n"), "\n")
		}
	}
	if end == 0 {
		end = start
	}
	if start > end {
		return nil, fmt.Errorf("start line (%d) must be <= end line (%d)", start, end)
	}

	suggestion := NewSuggestion(spec.Author, start, end, spec.Text, spec.OriginalText, spec.ProposedText)
	UpdateCommentSection(suggestion, doc.Content)
	doc.Threads = append(doc.Threads, suggestion)
	return &SuggestResult{SuggestionID: suggestion.ID, StartLine: start, EndLine: end, SectionPath: suggestion.SectionPath}, nil
}
