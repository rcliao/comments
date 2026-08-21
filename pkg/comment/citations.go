package comment

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rcliao/comments/pkg/markdown"
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
	for _, ref := range markdown.ParseReferences(content) {
		if ref.Line == 0 || ref.ThreadID != "" {
			continue
		}

		var candidates []string
		if strings.Contains(ref.Path, string(filepath.Separator)) {
			if abs, ok := markdown.ResolveReference(filepath.Dir(docPath), ref.Path); ok {
				candidates = []string{abs}
			}
		} else {
			if index == nil {
				index = basenameIndex(root)
			}
			candidates = index[ref.Path]
		}

		switch {
		case len(candidates) == 0:
			violations = append(violations, Violation{
				Rule:    "unresolvable_citation",
				Line:    ref.LineNum,
				Message: fmt.Sprintf("line %d: %s — no such file in the repository", ref.LineNum, ref.Raw),
			})
		case len(candidates) > 1:
			violations = append(violations, Violation{
				Rule: "ambiguous_citation",
				Line: ref.LineNum,
				Message: fmt.Sprintf("line %d: %s matches %d files (%s) — cite the path from the repo root so a reviewer can peek it",
					ref.LineNum, ref.Raw, len(candidates), shortList(root, candidates)),
			})
		default:
			last := ref.EndLine
			if last == 0 {
				last = ref.Line
			}
			if last < ref.Line {
				violations = append(violations, Violation{
					Rule:    "unresolvable_citation",
					Line:    ref.LineNum,
					Message: fmt.Sprintf("line %d: %s has a reversed line range", ref.LineNum, ref.Raw),
				})
				continue
			}
			if n := countLines(candidates[0]); n >= 0 && (ref.Line < 1 || last > n) {
				violations = append(violations, Violation{
					Rule:    "unresolvable_citation",
					Line:    ref.LineNum,
					Message: fmt.Sprintf("line %d: %s points past the end of the file (%d lines)", ref.LineNum, ref.Raw, n),
				})
			}
		}
	}
	return violations
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
