package comment

import (
	"regexp"
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
)

// LivingTemplate names the doc the agent keeps current while it builds: no
// verdict, no zones, read whenever the human wants the state.
const LivingTemplate = "living"

// IsLivingDoc reports whether content declares the living template.
func IsLivingDoc(content string) bool {
	meta, err := ParseDocumentMetadata(content)
	return err == nil && meta.Template == LivingTemplate
}

// SaveSeenBaseline records content as what the reader last saw. A living doc
// has no verdict, so closing the view moves the reader's baseline: the next
// open tints only what changed since they last looked.
func SaveSeenBaseline(docPath, reader, content string) error {
	return SaveReviewBaseline(docPath, reader, content)
}

// LivingState is what a glance at a living doc should say: its phase and the
// first line of Now, which is the doc's own recap.
type LivingState struct {
	Phase string `json:"phase,omitempty"`
	Now   string `json:"now,omitempty"`
}

var (
	livingLabel = regexp.MustCompile(`^\*\*[^*]+:\*\*\s*`)
	livingMark  = regexp.MustCompile("\\*\\*|`")
	// listItem matches any list marker: -, *, +, or 1. / 1)
	listItem = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+`)
)

// sectionItems returns the top-level list items of a section as
// with continuation lines folded in, skipping fenced code. Top
// level is the indent of the section's first item, so a list indented as a
// whole still counts and sub-bullets fold into their parent.
// sectionItem is one top-level list item: its 1-based line and its text.
type sectionItem struct {
	Line int
	Text string
}

func sectionItems(lines []string, sec *markdown.Section) []sectionItem {
	end := sec.EndLine
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	fenced := markdown.FencedLines(lines)
	var items []sectionItem
	indent := -1
	for i := sec.StartLine; i < end; i++ {
		if fenced[i] {
			continue
		}
		m := listItem.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		if indent < 0 {
			indent = len(m[1])
		}
		if len(m[1]) > indent {
			continue // a sub-bullet, already folded into its parent
		}
		text := lines[i][len(m[0]):]
		for j := i + 1; j < end && !fenced[j]; j++ {
			next := lines[j]
			if strings.TrimSpace(next) == "" || strings.HasPrefix(next, "#") {
				break
			}
			if n := listItem.FindStringSubmatch(next); n != nil && len(n[1]) <= indent {
				break
			}
			text += "\n" + next
		}
		items = append(items, sectionItem{Line: i + 1, Text: text})
	}
	return items
}

// ReadLivingState returns the phase and Now's first list item with its bold
// label and inline markup stripped, or false when content is not a living doc.
func ReadLivingState(content string) (LivingState, bool) {
	meta, err := ParseDocumentMetadata(content)
	if err != nil || meta.Template != LivingTemplate {
		return LivingState{}, false
	}
	st := LivingState{Phase: meta.Phase}
	body := markdown.MaskFrontmatter(content)
	lines := strings.Split(body, "\n")
	if sec := topSection(markdown.ParseDocument(body), "Now"); sec != nil {
		if items := sectionItems(lines, sec); len(items) > 0 {
			first, _, _ := strings.Cut(items[0].Text, "\n")
			st.Now = strings.TrimSpace(livingMark.ReplaceAllString(livingLabel.ReplaceAllString(first, ""), ""))
		}
	}
	return st, true
}

// topSection finds the shallowest section titled title, so a "### Now" nested
// under another section never wins over the doc's own "## Now".
func topSection(structure *markdown.DocumentStructure, title string) *markdown.Section {
	var best *markdown.Section
	var walk func([]*markdown.Section)
	walk = func(sections []*markdown.Section) {
		for _, s := range sections {
			if strings.EqualFold(strings.TrimSpace(s.Title), title) && (best == nil || s.Level < best.Level) {
				best = s
			}
			walk(s.Children)
		}
	}
	walk(structure.Sections)
	return best
}
