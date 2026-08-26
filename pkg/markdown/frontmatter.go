package markdown

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter is a parsed YAML block at the start of a Markdown document.
// StartLine and EndLine include the opening and closing delimiters and are
// one-based so callers can preserve source-addressed comments.
type Frontmatter struct {
	StartLine int
	EndLine   int
	Values    map[string]any
}

// ParseFrontmatter parses a leading YAML frontmatter block. A document without
// an opening delimiter is valid Markdown and returns present=false. Once an
// opener is present, a missing closer or invalid YAML is an error rather than
// silently treating metadata as prose.
func ParseFrontmatter(content string) (fm Frontmatter, present bool, err error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSuffix(lines[0], "\r") != "---" {
		return Frontmatter{}, false, nil
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSuffix(lines[i], "\r")
		if line == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return Frontmatter{}, true, fmt.Errorf("frontmatter opened on line 1 but has no closing --- delimiter")
	}

	values := map[string]any{}
	if raw := strings.Join(lines[1:end], "\n"); strings.TrimSpace(raw) != "" {
		if err := yaml.Unmarshal([]byte(raw), &values); err != nil {
			return Frontmatter{}, true, fmt.Errorf("invalid YAML frontmatter: %w", err)
		}
	}
	return Frontmatter{StartLine: 1, EndLine: end + 1, Values: values}, true, nil
}

// MaskFrontmatter replaces a leading frontmatter block with blank lines while
// retaining every newline. Markdown structure, citation locations, and comment
// anchors therefore keep their original line numbers, while body-only logic
// does not count or render metadata as prose.
func MaskFrontmatter(content string) string {
	fm, present, err := ParseFrontmatter(content)
	if !present || err != nil {
		return content
	}
	lines := strings.Split(content, "\n")
	for i := fm.StartLine - 1; i < fm.EndLine && i < len(lines); i++ {
		lines[i] = ""
	}
	return strings.Join(lines, "\n")
}

// FrontmatterString returns a scalar string field without coercing other YAML
// types. Producers should fix malformed metadata rather than have consumers
// guess whether numbers or booleans were intended as strings.
func FrontmatterString(values map[string]any, key string) string {
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

// FrontmatterStrings returns a YAML string list, tolerating a single string as
// a one-item convenience form.
func FrontmatterStrings(values map[string]any, key string) []string {
	switch value := values[key].(type) {
	case string:
		if strings.TrimSpace(value) != "" {
			return []string{strings.TrimSpace(value)}
		}
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	case []string:
		return append([]string(nil), value...)
	}
	return nil
}
