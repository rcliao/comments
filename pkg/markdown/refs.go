package markdown

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Reference is a citation to another local file found in a document line:
// a markdown link `[text](path#heading)` or a bare `path.ext:line` token.
// StartCol/EndCol are byte offsets of the raw token within the line, for
// span styling.
type Reference struct {
	Raw      string // exact token as it appears in the line
	Path     string // path as written (unresolved; "" for same-doc thread refs)
	Line     int    // cited line number (0 = none)
	Heading  string // cited #heading anchor (markdown links only, "" = none)
	ThreadID string // cited comment thread (thread:c1abc form, "" = none)
	LineNum  int    // 1-based document line the reference appears on
	StartCol int    // byte offset of Raw within the line
	EndCol   int    // byte offset just past Raw
}

// mdLinkRe matches [text](target) — target captured; images (![) excluded.
var mdLinkRe = regexp.MustCompile(`(^|[^!])\[[^\]]+\]\(([^)\s]+)\)`)

// fileLineRe matches bare path.ext:line tokens. The extension must come
// immediately before the colon so `research.md:42` parses as a unit
// (path=research.md, line=42); a leading path with slashes is optional.
// A backtick counts as a leading boundary: `file.go:12` (code-span-wrapped
// citations are the common style in agent-written docs, and were silently
// unpeekable before). Range citations (file.go:11-44) parse as their start.
var fileLineRe = regexp.MustCompile("(?:^|[\\s(`])((?:[\\w.\\-~]+/)*[\\w.\\-]+\\.[A-Za-z][A-Za-z0-9]{0,9}):(\\d{1,6})\\b")

// threadRefRe matches thread citations (docs/plan-compounding-rpi.md Phase 1,
// syntax decided in review thread cmvlt): thread:c1abc cites a thread in this
// document, thread:path.md#c1abc in another. An explicit scheme, so there is
// no collision with heading anchors and no precedence rule to remember.
var threadRefRe = regexp.MustCompile("(?:^|[\\s(`])thread:(?:((?:[\\w.\\-~]+/)*[\\w.\\-]+\\.md)#)?(c[a-z0-9]{4,12})\\b")

// ParseReferences scans document content for local-file references.
// Fenced code blocks are skipped — a path in example code is not a citation.
// URLs (scheme://) and pure anchors (#section) are not references.
func ParseReferences(content string) []Reference {
	var refs []Reference
	lines := strings.Split(content, "\n")

	inFence := false
	var fenceChar byte
	var fenceLen int

	for i, line := range lines {
		if inFence {
			if closesFence(line, fenceChar, fenceLen) {
				inFence = false
				continue
			}
			// Code in a fence is not a citation — but a COMMENT TRAIL in a
			// fence is exactly where schema-notation templates put their
			// file:line evidence (DBML: `max_points int // scorer.go:41`).
			// Mask everything before the comment marker, preserving byte
			// offsets, and parse only the trail.
			if masked, ok := commentTrail(line); ok {
				refs = append(refs, parseLineReferences(masked, i+1)...)
			}
			continue
		}
		if ch, ln, ok := parseFenceOpen(line); ok {
			inFence, fenceChar, fenceLen = true, ch, ln
			continue
		}
		refs = append(refs, parseLineReferences(line, i+1)...)
	}
	return refs
}

// commentTrail masks a fenced-code line up to its first comment marker
// (`//` or `#`), preserving byte offsets so parsed reference columns still
// point into the original line. ok=false when the line has no comment.
func commentTrail(line string) (string, bool) {
	idx := -1
	for _, marker := range []string{"//", "#"} {
		if i := strings.Index(line, marker); i >= 0 && (idx == -1 || i < idx) {
			idx = i
		}
	}
	if idx < 0 {
		return "", false
	}
	return strings.Repeat(" ", idx) + line[idx:], true
}

func parseLineReferences(line string, lineNum int) []Reference {
	var refs []Reference
	claimed := make([]bool, len(line)) // avoid double-reporting overlapping matches

	for _, m := range mdLinkRe.FindAllStringSubmatchIndex(line, -1) {
		targetStart, targetEnd := m[4], m[5]
		target := line[targetStart:targetEnd]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") ||
			strings.HasPrefix(target, "mailto:") {
			continue
		}
		path, heading, _ := strings.Cut(target, "#")
		if path == "" {
			continue
		}
		// The styled span is the whole [text](target) link. m[2],m[3] bound
		// the non-! prefix group; the link starts at the bracket.
		linkStart := strings.Index(line[m[0]:m[1]], "[") + m[0]
		ref := Reference{
			Raw:      line[linkStart:m[1]],
			Path:     path,
			Heading:  heading,
			LineNum:  lineNum,
			StartCol: linkStart,
			EndCol:   m[1],
		}
		refs = append(refs, ref)
		for c := linkStart; c < m[1]; c++ {
			claimed[c] = true
		}
	}

	for _, m := range threadRefRe.FindAllStringSubmatchIndex(line, -1) {
		// group 1 = optional path, group 2 = thread id
		idStart, idEnd := m[4], m[5]
		tokStart := strings.Index(line[m[0]:m[1]], "thread:") + m[0]
		if claimed[tokStart] {
			continue
		}
		path := ""
		if m[2] >= 0 {
			path = line[m[2]:m[3]]
		}
		refs = append(refs, Reference{
			Raw:      line[tokStart:idEnd],
			Path:     path,
			ThreadID: line[idStart:idEnd],
			LineNum:  lineNum,
			StartCol: tokStart,
			EndCol:   idEnd,
		})
		for c := tokStart; c < idEnd; c++ {
			claimed[c] = true
		}
	}

	for _, m := range fileLineRe.FindAllStringSubmatchIndex(line, -1) {
		pathStart, pathEnd := m[2], m[3]
		lineStart, lineEnd := m[4], m[5]
		if claimed[pathStart] {
			continue // inside a markdown link already reported
		}
		n, err := strconv.Atoi(line[lineStart:lineEnd])
		if err != nil || n == 0 {
			continue
		}
		refs = append(refs, Reference{
			Raw:      line[pathStart:lineEnd],
			Path:     line[pathStart:pathEnd],
			Line:     n,
			LineNum:  lineNum,
			StartCol: pathStart,
			EndCol:   lineEnd,
		})
	}
	return refs
}

// ResolveReference turns a reference path into an absolute path on disk.
// Tries docDir-relative first, then walks parent directories up to the
// filesystem root (mirrors template discovery). Returns ok=false when no
// existing file is found.
func ResolveReference(docDir, refPath string) (string, bool) {
	if filepath.IsAbs(refPath) {
		if fileExists(refPath) {
			return refPath, true
		}
		return "", false
	}
	dir := docDir
	for {
		candidate := filepath.Join(dir, refPath)
		if fileExists(candidate) {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// HeadingLine returns the 1-based line of the section matching a markdown
// anchor slug (e.g. "goals--non-goals" or a raw heading title) in content,
// or 0 when not found. Matching is case-insensitive on both the slugified
// and raw forms.
func HeadingLine(content, anchor string) int {
	if anchor == "" {
		return 0
	}
	want := strings.ToLower(anchor)
	structure := ParseDocument(content)
	var found int
	var walk func(sections []*Section)
	walk = func(sections []*Section) {
		for _, s := range sections {
			if found != 0 {
				return
			}
			title := strings.ToLower(s.Title)
			if title == want || slugify(s.Title) == want {
				found = s.StartLine
				return
			}
			walk(s.Children)
		}
	}
	walk(structure.Sections)
	return found
}

// slugify approximates GitHub's heading-anchor slugs: lowercase, spaces to
// hyphens, punctuation dropped.
func slugify(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteByte('-')
		}
	}
	return b.String()
}
