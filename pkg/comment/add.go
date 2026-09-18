package comment

import "fmt"

// NewCommentSpec locates and describes one comment to add. Exactly one of
// Line, Section or Anchor places it.
type NewCommentSpec struct {
	Line     int    `json:"line,omitempty" jsonschema:"Line number (one of line, section or anchor)"`
	Section  string `json:"section,omitempty" jsonschema:"Section path, e.g. 'Doc Title > Proposed Design'"`
	Anchor   string `json:"anchor,omitempty" jsonschema:"Quote of the target line or a unique substring of it (preferred: no line numbers to look up)"`
	Author   string `json:"author" jsonschema:"Author of the comment"`
	Text     string `json:"text" jsonschema:"Comment text"`
	Type     string `json:"type,omitempty" jsonschema:"Q question, S suggestion, B bug, T todo, E enhancement"`
	Priority string `json:"priority,omitempty" jsonschema:"low, medium (default) or high"`
	Blocking bool   `json:"blocking,omitempty" jsonschema:"Must be resolved before the gate passes"`
}

// AddResult reports the comments an add created, in the order they were given.
type AddResult struct {
	Count int           `json:"count"`
	Added []CommentView `json:"added"`
}

var validCommentTypes = map[string]bool{"Q": true, "S": true, "B": true, "T": true, "E": true}

// AddComments adds one or many comments to doc. It is the only add path: one
// comment is a batch of one, so both surfaces share validation, anchoring and
// error wording. The batch is atomic — every spec is validated and located
// before any is appended, and an error names the item that caused it.
func AddComments(doc *DocumentWithComments, specs []NewCommentSpec) (*AddResult, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("no comments given")
	}
	lines := make([]int, len(specs))
	for i, spec := range specs {
		where := fmt.Sprintf("comment %d", i+1)
		given := 0
		for _, ok := range []bool{spec.Line != 0, spec.Section != "", spec.Anchor != ""} {
			if ok {
				given++
			}
		}
		switch {
		case given == 0:
			return nil, fmt.Errorf("%s: one of line, section or anchor is required", where)
		case given > 1:
			return nil, fmt.Errorf("%s: line, section and anchor are mutually exclusive", where)
		case spec.Text == "":
			return nil, fmt.Errorf("%s: text is required", where)
		case spec.Author == "":
			return nil, fmt.Errorf("%s: author is required", where)
		case spec.Type != "" && !validCommentTypes[spec.Type]:
			return nil, fmt.Errorf("%s: invalid type %q (valid: Q, S, B, T, E)", where, spec.Type)
		}

		lines[i] = spec.Line
		if spec.Section != "" {
			if err := ValidateSectionPath(doc.Content, spec.Section); err != nil {
				return nil, fmt.Errorf("%s: %w", where, err)
			}
			start, _, err := ResolveSectionToLines(doc.Content, spec.Section, false)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", where, err)
			}
			lines[i] = start
		}
		if spec.Anchor != "" {
			resolved, err := ResolveAnchorText(doc.Content, spec.Anchor)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", where, err)
			}
			lines[i] = resolved
		}
	}

	result := &AddResult{Added: []CommentView{}}
	for i, spec := range specs {
		// PrefixType is normalized: it never doubles a marker the author wrote.
		text := PrefixType(spec.Text, spec.Type)
		var c *Comment
		if spec.Type != "" {
			c = NewCommentWithType(spec.Author, lines[i], text, spec.Type)
		} else {
			c = NewComment(spec.Author, lines[i], text)
		}
		c.Priority = "medium"
		if spec.Priority != "" {
			c.Priority = spec.Priority
		}
		c.Status = "active"
		c.Blocking = spec.Blocking
		UpdateCommentSection(c, doc.Content)
		doc.Threads = append(doc.Threads, c)
		result.Added = append(result.Added, NewCommentView(c))
	}
	result.Count = len(result.Added)
	return result, nil
}
