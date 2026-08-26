package comment

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	md "github.com/rcliao/comments/pkg/markdown"
)

type ContextOptions struct {
	For            string
	IncludeBody    bool
	IncludeThreads bool
}

type ContextReview struct {
	Decision           string        `json:"decision"`
	Blocking           int           `json:"blocking"`
	NonBlocking        int           `json:"non_blocking"`
	PendingSuggestions int           `json:"pending_suggestions"`
	LastReview         *ReviewRecord `json:"last_review,omitempty"`
}

type ContextDocument struct {
	Path        string        `json:"path"`
	Type        string        `json:"type,omitempty"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	Status      string        `json:"status,omitempty"`
	Template    string        `json:"template,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Review      ContextReview `json:"review"`
	Focus       string        `json:"focus,omitempty"`
	Body        string        `json:"body,omitempty"`
	Threads     []CommentView `json:"threads,omitempty"`
	absPath     string
	metadata    DocumentMetadata
}

type ContextRelation struct {
	Relation string          `json:"relation,omitempty"`
	Reason   string          `json:"reason"`
	Document ContextDocument `json:"document"`
}

type DocumentContext struct {
	Mode           string                     `json:"mode"`
	Bundle         string                     `json:"bundle"`
	BundleRoot     string                     `json:"bundle_root"`
	Document       ContextDocument            `json:"document"`
	Implementation *PlanImplementationContext `json:"implementation,omitempty"`
	Related        []ContextRelation          `json:"related"`
	Backlinks      []ContextRelation          `json:"backlinks"`
	Sources        []KnowledgeSource          `json:"sources"`
	Suggestions    []ContextRelation          `json:"suggestions"`
}

// BuildDocumentContext returns a deterministic, explainable neighborhood. It
// does not use embeddings or an LLM; every included concept says why it is
// present.
func BuildDocumentContext(docPath string, options ContextOptions) (*DocumentContext, error) {
	absPath, err := filepath.Abs(docPath)
	if err != nil {
		return nil, err
	}
	bundle, err := FindBundle(absPath)
	if err != nil {
		return nil, err
	}
	mode := strings.TrimSpace(options.For)
	if mode == "" {
		mode = "drafting"
	}
	switch mode {
	case "drafting", "review", "coverage-scout", "evidence-verifier", "human-review", "implementation":
	default:
		return nil, fmt.Errorf("unknown context mode %q", mode)
	}
	if mode == "coverage-scout" {
		options.IncludeBody = false
		options.IncludeThreads = false
	}

	current, err := loadContextDocument(bundle, absPath, options)
	if err != nil {
		return nil, err
	}
	if mode == "coverage-scout" {
		current.Focus = contextSection(contentAt(absPath), "Research Question")
	}
	result := &DocumentContext{
		Mode:        mode,
		Bundle:      bundle.Config.Bundle,
		BundleRoot:  bundle.RootPath,
		Document:    current,
		Sources:     nonNilSources(current.metadata.Sources),
		Related:     []ContextRelation{},
		Backlinks:   []ContextRelation{},
		Suggestions: []ContextRelation{},
	}
	if mode == "implementation" {
		if current.Template != "plan" && !strings.EqualFold(current.Type, "plan") {
			return nil, fmt.Errorf("implementation context requires a plan document")
		}
		doc, _, err := LoadFromSidecar(absPath)
		if err != nil {
			return nil, err
		}
		implementation := BuildPlanImplementationContext(doc)
		result.Implementation = &implementation
	}

	paths, err := bundleConceptPaths(bundle)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{absPath: true}

	for _, related := range current.metadata.Related {
		target, ok := resolveKnowledgePath(bundle, filepath.Dir(absPath), related.Path)
		if !ok || target == absPath {
			continue
		}
		doc, err := loadContextDocument(bundle, target, options)
		if err != nil {
			continue
		}
		result.Related = append(result.Related, ContextRelation{Relation: related.Relation, Reason: "explicit frontmatter relationship", Document: doc})
		known[target] = true
	}

	// Draft-derived links are deliberately absent from the draft-blind scout.
	if mode != "coverage-scout" {
		data, _ := os.ReadFile(absPath)
		for _, ref := range md.ParseReferences(string(data)) {
			if ref.Line != 0 || ref.ThreadID != "" || !strings.HasSuffix(strings.ToLower(ref.Path), ".md") {
				continue
			}
			target, ok := resolveKnowledgePath(bundle, filepath.Dir(absPath), ref.Path)
			if !ok || known[target] {
				continue
			}
			doc, err := loadContextDocument(bundle, target, options)
			if err != nil {
				continue
			}
			result.Related = append(result.Related, ContextRelation{Relation: "links_to", Reason: "Markdown link in document body", Document: doc})
			known[target] = true
		}
	}

	if mode != "coverage-scout" {
		for _, candidate := range paths {
			if candidate == absPath {
				continue
			}
			meta, refs, err := documentEdges(candidate)
			if err != nil {
				continue
			}
			var relation string
			for _, edge := range meta.Related {
				if target, ok := resolveKnowledgePath(bundle, filepath.Dir(candidate), edge.Path); ok && target == absPath {
					relation = edge.Relation
					break
				}
			}
			if relation == "" {
				for _, ref := range refs {
					if target, ok := resolveKnowledgePath(bundle, filepath.Dir(candidate), ref); ok && target == absPath {
						relation = "links_to"
						break
					}
				}
			}
			if relation == "" {
				continue
			}
			doc, err := loadContextDocument(bundle, candidate, options)
			if err == nil {
				result.Backlinks = append(result.Backlinks, ContextRelation{Relation: relation, Reason: "document links to the current concept", Document: doc})
			}
		}
	}

	// Tag matches are useful during drafting but would widen a strict reviewer
	// or evidence verifier's allowlist without an explicit relationship.
	if mode == "drafting" && len(current.Tags) > 0 {
		for _, candidate := range paths {
			if known[candidate] || candidate == absPath || len(result.Suggestions) >= 5 {
				continue
			}
			doc, err := loadContextDocument(bundle, candidate, ContextOptions{})
			if err != nil || !sharesTag(current.Tags, doc.Tags) {
				continue
			}
			result.Suggestions = append(result.Suggestions, ContextRelation{Relation: "shares_tag", Reason: "matching frontmatter tag", Document: doc})
		}
	}

	sortContextRelations(result.Related)
	sortContextRelations(result.Backlinks)
	sortContextRelations(result.Suggestions)
	stripContextInternals(&result.Document)
	for i := range result.Related {
		stripContextInternals(&result.Related[i].Document)
	}
	for i := range result.Backlinks {
		stripContextInternals(&result.Backlinks[i].Document)
	}
	for i := range result.Suggestions {
		stripContextInternals(&result.Suggestions[i].Document)
	}
	return result, nil
}

func loadContextDocument(bundle *Bundle, path string, options ContextOptions) (ContextDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ContextDocument{}, err
	}
	content := string(data)
	meta, err := ParseDocumentMetadata(content)
	if err != nil {
		return ContextDocument{}, err
	}
	templateName, _, err := ResolveTemplateName(path, content, "", "")
	if err != nil {
		return ContextDocument{}, err
	}
	doc, _, err := LoadFromSidecar(path)
	if err != nil {
		return ContextDocument{}, err
	}
	if templateName == "" {
		templateName = doc.Template
	}
	gate := EvaluateGate(doc, false)
	rel, err := filepath.Rel(bundle.RootPath, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		rel, _ = filepath.Rel(bundle.ProjectDir, path)
	}
	out := ContextDocument{
		Path: filepath.ToSlash(rel), Type: meta.Type, Title: meta.Title,
		Description: meta.Description, Status: meta.Status, Template: templateName,
		Tags: nonNilStrings(meta.Tags), absPath: path, metadata: meta,
		Review: ContextReview{
			Decision: gate.Decision, Blocking: len(gate.Blocking), NonBlocking: len(gate.NonBlocking),
			PendingSuggestions: len(gate.PendingSuggestions), LastReview: gate.LastReview,
		},
	}
	if options.IncludeBody {
		out.Body = strings.TrimSpace(md.MaskFrontmatter(content))
	}
	if options.IncludeThreads {
		out.Threads = NewCommentViews(doc.Threads)
	}
	return out, nil
}

func contentAt(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}

func contextSection(content, section string) string {
	lines := strings.Split(content, "\n")
	structure := md.ParseDocument(content)
	for _, candidate := range structure.SectionsByID {
		if candidate.Title != section {
			continue
		}
		start, end := candidate.StartLine, candidate.EndLine
		if start < 1 || end > len(lines) || start >= end {
			return ""
		}
		return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	}
	return ""
}

func documentEdges(path string) (DocumentMetadata, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DocumentMetadata{}, nil, err
	}
	meta, err := ParseDocumentMetadata(string(data))
	if err != nil {
		return DocumentMetadata{}, nil, err
	}
	var refs []string
	for _, ref := range md.ParseReferences(string(data)) {
		if ref.Line == 0 && ref.ThreadID == "" && strings.HasSuffix(strings.ToLower(ref.Path), ".md") {
			refs = append(refs, ref.Path)
		}
	}
	return meta, refs, nil
}

func resolveKnowledgePath(bundle *Bundle, base, value string) (string, bool) {
	if value == "" || strings.Contains(value, "://") {
		return "", false
	}
	value, _, _ = strings.Cut(value, "#")
	var target string
	if strings.HasPrefix(value, "/") {
		target = filepath.Join(bundle.RootPath, filepath.FromSlash(strings.TrimPrefix(value, "/")))
	} else {
		target = filepath.Join(base, filepath.FromSlash(value))
	}
	target = filepath.Clean(target)
	rootRel, err := filepath.Rel(bundle.RootPath, target)
	if err != nil || rootRel == ".." || strings.HasPrefix(rootRel, ".."+string(filepath.Separator)) {
		return "", false
	}
	info, err := os.Stat(target)
	return target, err == nil && !info.IsDir()
}

func bundleConceptPaths(bundle *Bundle) ([]string, error) {
	var paths []string
	for _, collection := range bundle.Config.Collections {
		root := bundle.collectionPath(collection)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") || entry.Name() == "index.md" || entry.Name() == "log.md" {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func sharesTag(left, right []string) bool {
	want := map[string]bool{}
	for _, tag := range left {
		want[strings.ToLower(tag)] = true
	}
	for _, tag := range right {
		if want[strings.ToLower(tag)] {
			return true
		}
	}
	return false
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilSources(values []KnowledgeSource) []KnowledgeSource {
	if values == nil {
		return []KnowledgeSource{}
	}
	return values
}

func sortContextRelations(values []ContextRelation) {
	sort.Slice(values, func(i, j int) bool { return values[i].Document.Path < values[j].Document.Path })
}

func stripContextInternals(doc *ContextDocument) {
	doc.absPath = ""
	doc.metadata = DocumentMetadata{}
}
