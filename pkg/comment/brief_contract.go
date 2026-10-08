package comment

import (
	"regexp"
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
)

// BriefTemplate is the tiered artifact: the human owns its zone: human
// sections (Why, What, Shape, Checks), the agent owns the rest (How).
const BriefTemplate = "brief"

// loadBuiltinTemplate reads a template from the embedded set only. Approval
// hashing must not depend on where comments runs or on a project template an
// agent can write, so it never consults .comments/templates.
func loadBuiltinTemplate(name string) (*Template, error) {
	data, err := builtinTemplates.ReadFile("templates/" + name + ".yaml")
	if err != nil {
		return nil, err
	}
	return parseTemplate(data)
}

var (
	// Lines CommonMark renders as a level 1-2 heading that the section parser
	// does not split on: an ATX heading indented 1-3 spaces, and a setext
	// underline (judged with the line before it).
	indentedTopHeading = regexp.MustCompile(`^ {1,3}#{1,2}(\s|$)`)
	setextUnderline    = regexp.MustCompile(`^ {0,3}(=+|-+)\s*$`)
)

// BriefContractHash hashes exactly what the human approves in a brief:
// everything except the bodies of agent-owned sections (How). That covers the
// frontmatter, any text above the first section, every heading of the human
// sections and the agent sections' own heading lines, and the human sections
// in full. Inside an agent body, a line a reader would see as a new top-level
// section still counts, so How cannot fake a second Checks. Blank lines count
// (they change how markdown renders), with runs collapsed to one and trailing
// whitespace ignored.
func BriefContractHash(content string, t *Template) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	structure := markdown.ParseDocument(strings.Join(lines, "\n"))
	agentBody := map[int]bool{}
	for _, ts := range t.Sections {
		if ts.Zone == "human" {
			continue
		}
		if s := findTemplateSection(structure, ts.Heading); s != nil {
			for line := s.StartLine + 1; line <= s.EndLine && line <= len(lines); line++ {
				agentBody[line] = true
			}
		}
	}
	var b strings.Builder
	prevBlank := false
	for i, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		n := i + 1
		if agentBody[n] {
			prevText := i > 0 && strings.TrimSpace(lines[i-1]) != ""
			switch {
			case indentedTopHeading.MatchString(line):
				b.WriteString("A|" + line + "\n")
			case prevText && setextUnderline.MatchString(line):
				b.WriteString("A|" + strings.TrimRight(lines[i-1], " \t") + "\n" + line + "\n")
			}
			continue
		}
		if line == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			prevBlank = false
		}
		b.WriteString(line + "\n")
	}
	return ComputeDocumentHash(b.String())
}
