package comment

import (
	"fmt"
	"regexp"
	"strings"
)

// Style caps shape prose for scanning rather than reading start-to-finish.
//
// Measured over this repo's shipped RPI docs: sentences run to 86 words and
// paragraphs to 144, and the most recent docs were 91% prose blocks against a
// 32% corpus average. A reviewer facing a 144-word paragraph cannot find the
// claim without reading all of it, which is the opposite of what a review
// artifact is for.
//
// The caps do not mandate bullets directly — a bullet quota invites padding a
// list to satisfy it. Capping paragraph size pushes the same way without
// rewarding the wrong behavior: once a block cannot hold three ideas, they
// separate on their own.

// splitSentences breaks prose on terminal punctuation followed by whitespace
// and an opening token. Written as a scan rather than a regex because Go's RE2
// has no lookaround, and a naive split on ". " would cut "pkg/comment/gate.go:39"
// and "v2.1" in half.
func splitSentences(body string) []string {
	var out []string
	start := 0
	runes := []rune(body)
	for i := 0; i < len(runes)-1; i++ {
		if runes[i] != '.' && runes[i] != '!' && runes[i] != '?' {
			continue
		}
		j := i + 1
		for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
			j++
		}
		if j == i+1 || j >= len(runes) {
			continue // no space after the stop: an abbreviation or a version
		}
		switch c := runes[j]; {
		case c >= 'A' && c <= 'Z', c == '(', c == '`', c == '*', c == '[':
			out = append(out, string(runes[start:i+1]))
			start = j
			i = j - 1
		}
	}
	if start < len(runes) {
		out = append(out, string(runes[start:]))
	}
	return out
}

// codeSpan matches an inline `code` span.
var codeSpan = regexp.MustCompile("`[^`]+`")

// readableWords counts what a reader parses, collapsing each inline code span
// to a single token. A trial agent reported the raw count "penalizes precision":
// `pkg/comment/gate.go:39` is one thing to a reader but three words to Fields,
// so naming the exact symbol cost more budget than writing vaguely.
func readableWords(s string) int {
	return len(strings.Fields(codeSpan.ReplaceAllString(s, "CODE")))
}

// TemplateStyle bounds prose shape. Zero values disable a check.
type TemplateStyle struct {
	MaxSentenceWords  int `yaml:"max_sentence_words"`
	MaxParagraphWords int `yaml:"max_paragraph_words"`
}

// proseBlocks returns the blank-line-separated prose blocks of a document,
// with their starting line numbers. Headings, list items, tables and fenced
// code are excluded: their shape is deliberate and not prose to be split.
func proseBlocks(lines []string) (starts []int, bodies []string) {
	var cur []string
	curStart := 0
	inFence := false

	flush := func() {
		if len(cur) == 0 {
			return
		}
		first := strings.TrimSpace(cur[0])
		isList := strings.HasPrefix(first, "-") || strings.HasPrefix(first, "*") ||
			strings.HasPrefix(first, "+") || strings.HasPrefix(first, ">") ||
			strings.HasPrefix(first, "|") || listNumber.MatchString(first)
		if !strings.HasPrefix(first, "#") && !isList {
			starts = append(starts, curStart)
			bodies = append(bodies, strings.Join(cur, " "))
		}
		cur = nil
	}

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			flush()
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if len(cur) == 0 {
			curStart = i + 1
		}
		cur = append(cur, strings.TrimSpace(line))
	}
	flush()
	return starts, bodies
}

var listNumber = regexp.MustCompile(`^\d+[.)]\s`)

// validateStyle reports prose that is too dense to scan.
func validateStyle(lines []string, style TemplateStyle) []Violation {
	if style.MaxSentenceWords == 0 && style.MaxParagraphWords == 0 {
		return nil
	}
	var violations []Violation
	starts, bodies := proseBlocks(lines)

	for i, body := range bodies {
		line := starts[i]
		// A [NEEDS CLARIFICATION: ...] marker is an annotation whose bracket
		// syntax competed with its own content for the sentence budget.
		if strings.HasPrefix(strings.TrimSpace(body), "[NEEDS CLARIFICATION") {
			continue
		}

		if style.MaxParagraphWords > 0 {
			if n := readableWords(body); n > style.MaxParagraphWords {
				violations = append(violations, Violation{
					Rule:    "long_paragraph",
					Line:    line,
					Message: fmt.Sprintf("line %d: paragraph is %d words (max %d) — split it, one idea per block, or turn the list of things into a list", line, n, style.MaxParagraphWords),
				})
			}
		}

		if style.MaxSentenceWords > 0 {
			for _, s := range splitSentences(body) {
				n := readableWords(s)
				if n <= style.MaxSentenceWords {
					continue
				}
				violations = append(violations, Violation{
					Rule:    "long_sentence",
					Line:    line,
					Message: fmt.Sprintf("line %d: %d-word sentence (max %d) starting %q", line, n, style.MaxSentenceWords, opening(s)),
				})
			}
		}
	}
	return violations
}

// opening quotes the first few words so the writer can find the sentence.
func opening(s string) string {
	words := strings.Fields(s)
	if len(words) > 7 {
		return strings.Join(words[:7], " ") + "…"
	}
	return strings.Join(words, " ")
}
