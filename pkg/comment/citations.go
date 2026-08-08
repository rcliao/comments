package comment

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Citation checking answers the question the review criteria ask but nothing
// enforced: can a reviewer actually open this evidence?
//
// A paired eval across six drafted research docs found up to half the
// references on a single question were unpeekable — not wrong, but ambiguous:
// "gate.go:39" when pkg/comment/ and cmd/comments/ both have a gate.go. The
// reviewer presses f and cannot know which file was meant, so a citation that
// looks rigorous carries no more weight than an assertion.
//
// This needs the filesystem, so it is deliberately separate from
// ValidateTemplate, which stays a pure content check.

// citationRef matches "path/to/file.go:12" and "file.go:12-34". The extension
// list keeps version strings and prose out of the match.
var citationRef = regexp.MustCompile(`([A-Za-z0-9_./-]+\.(?:go|md|ya?ml|json|ts|tsx|js|py|sh)):(\d+)(?:-(\d+))?`)

// FindRepoRoot walks up from startDir looking for a repository marker, so
// citations resolve the way a reader would read them: relative to the project.
func FindRepoRoot(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	for {
		for _, marker := range []string{".git", "go.mod"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// basenameIndex maps each file's base name to every path carrying it, so a
// bare-basename citation can be reported as ambiguous instead of guessed at.
func basenameIndex(root string) map[string][]string {
	index := map[string][]string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtrees hold no citable evidence
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		index[d.Name()] = append(index[d.Name()], path)
		return nil
	})
	return index
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return -1
	}
	defer func() { _ = f.Close() }()
	n := 0
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for s.Scan() {
		n++
	}
	return n
}

// ValidateCitations reports references a reviewer could not follow: files that
// do not exist, line numbers past the end of the file, and bare basenames that
// match more than one file. References inside fenced code blocks are skipped —
// those are illustrations, not evidence.
func ValidateCitations(content, docPath string) []Violation {
	root := FindRepoRoot(filepath.Dir(docPath))
	if root == "" {
		return nil // outside a project there is nothing to resolve against
	}

	var violations []Violation
	var index map[string][]string // built lazily; only bare basenames need it
	inFence := false

	for i, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range citationRef.FindAllStringSubmatch(line, -1) {
			ref, path := m[0], m[1]
			start, _ := strconv.Atoi(m[2])
			last := start
			if m[3] != "" {
				last, _ = strconv.Atoi(m[3])
			}

			var candidates []string
			if strings.Contains(path, string(filepath.Separator)) {
				if abs := filepath.Join(root, path); fileExists(abs) {
					candidates = []string{abs}
				}
			} else {
				if index == nil {
					index = basenameIndex(root)
				}
				candidates = index[path]
			}

			switch {
			case len(candidates) == 0:
				violations = append(violations, Violation{
					Rule:    "unresolvable_citation",
					Line:    i + 1,
					Message: fmt.Sprintf("line %d: %s — no such file in the repository", i+1, ref),
				})
			case len(candidates) > 1:
				violations = append(violations, Violation{
					Rule: "ambiguous_citation",
					Line: i + 1,
					Message: fmt.Sprintf("line %d: %s matches %d files (%s) — cite the path from the repo root so a reviewer can peek it",
						i+1, ref, len(candidates), shortList(root, candidates)),
				})
			default:
				if n := countLines(candidates[0]); n >= 0 && (start < 1 || last > n) {
					violations = append(violations, Violation{
						Rule:    "unresolvable_citation",
						Line:    i + 1,
						Message: fmt.Sprintf("line %d: %s points past the end of the file (%d lines)", i+1, ref, n),
					})
				}
			}
		}
	}
	return violations
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// shortList renders candidate paths relative to the root, capped so the message
// stays readable.
func shortList(root string, paths []string) string {
	const max = 3
	out := make([]string, 0, max)
	for i, p := range paths {
		if i == max {
			out = append(out, fmt.Sprintf("and %d more", len(paths)-max))
			break
		}
		if rel, err := filepath.Rel(root, p); err == nil {
			p = rel
		}
		out = append(out, p)
	}
	return strings.Join(out, ", ")
}
