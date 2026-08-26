package comment

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectBundleFile is the project-local, versioned bundle configuration.
const ProjectBundleFile = ".comments/bundle.yaml"

// BundleConfig maps reusable document templates into a project folder shape.
// The config lives under .comments, but unlike review baselines and view state
// it is intended to be committed.
type BundleConfig struct {
	Bundle      string                      `yaml:"bundle" json:"bundle"`
	Version     int                         `yaml:"version" json:"version"`
	OKFVersion  string                      `yaml:"okf_version" json:"okf_version"`
	Root        string                      `yaml:"root" json:"root"`
	Collections map[string]BundleCollection `yaml:"collections" json:"collections"`
}

type BundleCollection struct {
	Path        string   `yaml:"path" json:"path"`
	Type        string   `yaml:"type" json:"type"`
	Templates   []string `yaml:"templates" json:"templates"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

// Bundle is a resolved config with absolute project and knowledge roots.
type Bundle struct {
	Config     BundleConfig
	ConfigPath string
	ProjectDir string
	RootPath   string
}

// FindBundle walks upward from a document or directory for .comments/bundle.yaml.
func FindBundle(start string) (*Bundle, error) {
	if start == "" {
		start = "."
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Stat(abs); statErr == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, ProjectBundleFile)
		if data, readErr := os.ReadFile(path); readErr == nil {
			var config BundleConfig
			if err := yaml.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("invalid bundle config %s: %w", path, err)
			}
			bundle := &Bundle{Config: config, ConfigPath: path, ProjectDir: dir}
			if err := bundle.resolveAndValidate(); err != nil {
				return nil, err
			}
			return bundle, nil
		} else if !os.IsNotExist(readErr) {
			return nil, readErr
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, fmt.Errorf("no %s found above %s", ProjectBundleFile, start)
}

func (b *Bundle) resolveAndValidate() error {
	if b.Config.Version == 0 {
		b.Config.Version = 1
	}
	if b.Config.Version != 1 {
		return fmt.Errorf("unsupported bundle config version %d", b.Config.Version)
	}
	if strings.TrimSpace(b.Config.Root) == "" {
		return fmt.Errorf("bundle config %s is missing root", b.ConfigPath)
	}
	root, err := safeRelativeJoin(b.ProjectDir, b.Config.Root)
	if err != nil {
		return fmt.Errorf("invalid bundle root: %w", err)
	}
	b.RootPath = root
	if len(b.Config.Collections) == 0 {
		return fmt.Errorf("bundle config %s has no collections", b.ConfigPath)
	}
	for name, collection := range b.Config.Collections {
		if strings.TrimSpace(collection.Path) == "" || strings.TrimSpace(collection.Type) == "" || len(collection.Templates) == 0 {
			return fmt.Errorf("bundle collection %q requires path, type, and at least one template", name)
		}
		if _, err := safeRelativeJoin(root, collection.Path); err != nil {
			return fmt.Errorf("invalid path for bundle collection %q: %w", name, err)
		}
	}
	return nil
}

func safeRelativeJoin(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("path must be relative: %s", relative)
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes its root: %s", relative)
	}
	return filepath.Join(root, clean), nil
}

func (b *Bundle) collectionPath(collection BundleCollection) string {
	path, _ := safeRelativeJoin(b.RootPath, collection.Path)
	return path
}

// CollectionForTemplate finds the unique collection assigned to a template.
func (b *Bundle) CollectionForTemplate(template string) (string, BundleCollection, error) {
	var foundName string
	var found BundleCollection
	for name, collection := range b.Config.Collections {
		for _, candidate := range collection.Templates {
			if candidate != template {
				continue
			}
			if foundName != "" {
				return "", BundleCollection{}, fmt.Errorf("template %q belongs to multiple bundle collections (%s, %s)", template, foundName, name)
			}
			foundName, found = name, collection
		}
	}
	if foundName == "" {
		return "", BundleCollection{}, fmt.Errorf("template %q is not assigned to a bundle collection", template)
	}
	return foundName, found, nil
}

// CollectionForDocument identifies a document by containment in a configured
// collection. The longest collection path wins if projects deliberately nest.
func (b *Bundle) CollectionForDocument(docPath string) (string, BundleCollection, bool) {
	abs, err := filepath.Abs(docPath)
	if err != nil {
		return "", BundleCollection{}, false
	}
	type match struct {
		name string
		path string
		col  BundleCollection
	}
	var matches []match
	for name, collection := range b.Config.Collections {
		root := b.collectionPath(collection)
		rel, err := filepath.Rel(root, abs)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			matches = append(matches, match{name: name, path: root, col: collection})
		}
	}
	if len(matches) == 0 {
		return "", BundleCollection{}, false
	}
	sort.Slice(matches, func(i, j int) bool { return len(matches[i].path) > len(matches[j].path) })
	return matches[0].name, matches[0].col, true
}

// ResolveTemplateName centralizes the durable template association used by
// validation, gates, analysis, context, and zone guards.
func ResolveTemplateName(docPath, content, explicit, legacy string) (name, source string, err error) {
	if explicit != "" {
		return explicit, "explicit", nil
	}
	meta, err := ParseDocumentMetadata(content)
	if err != nil {
		return "", "", err
	}
	if meta.Template != "" {
		return meta.Template, "frontmatter", nil
	}
	if legacy != "" {
		return legacy, "sidecar", nil
	}
	if bundle, bundleErr := FindBundle(docPath); bundleErr == nil {
		if _, collection, ok := bundle.CollectionForDocument(docPath); ok {
			if len(collection.Templates) == 1 {
				return collection.Templates[0], "bundle", nil
			}
		}
	}
	return "", "", nil
}

// ResolveTemplateForDocument resolves and loads a document template.
func ResolveTemplateForDocument(docPath, content, explicit, legacy string) (*Template, string, error) {
	name, source, err := ResolveTemplateName(docPath, content, explicit, legacy)
	if err != nil || name == "" {
		return nil, source, err
	}
	template, err := LoadTemplateForDoc(name, docPath)
	return template, source, err
}

// ValidateManagedDocument applies template and citation rules everywhere, then
// adds the OKF metadata floor when the document belongs to a bundle collection.
func ValidateManagedDocument(content, docPath string, template *Template) []Violation {
	violations := ValidateDocument(content, docPath, template)
	bundle, err := FindBundle(docPath)
	if err != nil {
		return violations
	}
	_, collection, ok := bundle.CollectionForDocument(docPath)
	if !ok {
		return violations
	}
	violations = append(violations, ValidateOKFMetadata(content)...)
	meta, metaErr := ParseDocumentMetadata(content)
	if metaErr == nil && meta.Type != "" && meta.Type != collection.Type {
		violations = append(violations, Violation{
			Rule: "collection_type_mismatch", Line: 1,
			Message: fmt.Sprintf("OKF type %q does not match bundle collection type %q", meta.Type, collection.Type),
		})
	}
	return violations
}

var bundleSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type NewDocumentOptions struct {
	Name        string
	Template    string
	Title       string
	Description string
	From        string
	StartDir    string
}

type NewDocumentResult struct {
	Path       string `json:"path"`
	Sidecar    string `json:"sidecar"`
	Collection string `json:"collection"`
	Template   string `json:"template"`
	Type       string `json:"type"`
}

// CreateBundleDocument creates the Markdown skeleton and empty review sidecar,
// then refreshes generated indexes.
func CreateBundleDocument(options NewDocumentOptions) (NewDocumentResult, error) {
	name := strings.TrimSuffix(strings.TrimSpace(options.Name), ".md")
	if !bundleSlug.MatchString(name) {
		return NewDocumentResult{}, fmt.Errorf("document name %q must be a lowercase slug (letters, numbers, dot, underscore, hyphen)", options.Name)
	}
	if options.Template == "" {
		return NewDocumentResult{}, fmt.Errorf("template is required")
	}
	bundle, err := FindBundle(options.StartDir)
	if err != nil {
		return NewDocumentResult{}, err
	}
	collectionName, collection, err := bundle.CollectionForTemplate(options.Template)
	if err != nil {
		return NewDocumentResult{}, err
	}
	template, err := LoadTemplateForDoc(options.Template, bundle.RootPath)
	if err != nil {
		return NewDocumentResult{}, err
	}
	collectionRoot := bundle.collectionPath(collection)
	path := filepath.Join(collectionRoot, name+".md")
	if _, err := os.Stat(path); err == nil {
		return NewDocumentResult{}, fmt.Errorf("document already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return NewDocumentResult{}, err
	}
	if err := os.MkdirAll(collectionRoot, 0o755); err != nil {
		return NewDocumentResult{}, err
	}
	title := strings.TrimSpace(options.Title)
	if title == "" {
		title = humanizeSlug(name)
	}
	content, err := newDocumentContent(bundle, path, collection, template, title, options.Description, options.From)
	if err != nil {
		return NewDocumentResult{}, err
	}
	if err := writeFileAtomic(path, []byte(content), 0o644); err != nil {
		return NewDocumentResult{}, err
	}
	doc := &DocumentWithComments{Content: content, Threads: []*Comment{}, Template: template.Name}
	if err := SaveToSidecar(path, doc); err != nil {
		return NewDocumentResult{}, err
	}
	if err := WriteBundleIndexes(bundle); err != nil {
		return NewDocumentResult{}, err
	}
	return NewDocumentResult{Path: path, Sidecar: GetSidecarPath(path), Collection: collectionName, Template: template.Name, Type: collection.Type}, nil
}

func humanizeSlug(slug string) string {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(slug))
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func newDocumentContent(bundle *Bundle, path string, collection BundleCollection, template *Template, title, description, from string) (string, error) {
	frontmatter := map[string]any{
		"type":     collection.Type,
		"title":    title,
		"status":   "draft",
		"comments": map[string]any{"template": template.Name},
	}
	if strings.TrimSpace(description) != "" {
		frontmatter["description"] = strings.TrimSpace(description)
	}
	if from != "" {
		fromAbs := filepath.Clean(from)
		if !filepath.IsAbs(fromAbs) {
			// Bundle relationships are project-relative so agents get the same
			// edge regardless of the subdirectory they launched from.
			fromAbs = filepath.Join(bundle.ProjectDir, fromAbs)
		}
		if _, err := os.Stat(fromAbs); err != nil {
			return "", fmt.Errorf("related source does not exist: %w", err)
		}
		rel, err := filepath.Rel(filepath.Dir(path), fromAbs)
		if err != nil {
			return "", err
		}
		frontmatter["related"] = []map[string]any{{"path": filepath.ToSlash(rel), "relation": "informed_by"}}
	}
	encoded, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	body.WriteString("---\n")
	body.Write(encoded)
	body.WriteString("---\n\n# ")
	body.WriteString(title)
	body.WriteString("\n")
	for _, section := range template.Sections {
		if !section.Required {
			continue
		}
		body.WriteString("\n## ")
		body.WriteString(section.Heading)
		body.WriteString("\n")
	}
	return body.String(), nil
}

type indexedConcept struct {
	path string
	meta DocumentMetadata
}

// WriteBundleIndexes regenerates the root and per-collection OKF indexes.
func WriteBundleIndexes(bundle *Bundle) error {
	if err := os.MkdirAll(bundle.RootPath, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(bundle.Config.Collections))
	for name := range bundle.Config.Collections {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		collection := bundle.Config.Collections[name]
		root := bundle.collectionPath(collection)
		if err := os.MkdirAll(root, 0o755); err != nil {
			return err
		}
		concepts, err := readCollectionConcepts(root)
		if err != nil {
			return err
		}
		var index strings.Builder
		index.WriteString("<!-- Generated by comments bundle index; edits will be replaced. -->\n\n# ")
		index.WriteString(humanizeSlug(name))
		index.WriteString("\n")
		for _, concept := range concepts {
			rel, _ := filepath.Rel(root, concept.path)
			title := concept.meta.Title
			if title == "" {
				title = humanizeSlug(strings.TrimSuffix(filepath.Base(concept.path), ".md"))
			}
			fmt.Fprintf(&index, "\n- [%s](%s)", title, filepath.ToSlash(rel))
			if concept.meta.Description != "" {
				fmt.Fprintf(&index, " — %s", concept.meta.Description)
			}
			index.WriteString("\n")
		}
		if err := writeFileAtomic(filepath.Join(root, "index.md"), []byte(index.String()), 0o644); err != nil {
			return err
		}
	}

	version := bundle.Config.OKFVersion
	if version == "" {
		version = "0.2"
	}
	var root strings.Builder
	fmt.Fprintf(&root, "---\nokf_version: %q\n---\n\n# %s\n", version, bundle.Config.Bundle)
	for _, name := range names {
		collection := bundle.Config.Collections[name]
		fmt.Fprintf(&root, "\n- [%s](%s/) ", humanizeSlug(name), filepath.ToSlash(collection.Path))
		if collection.Description != "" {
			root.WriteString("— " + collection.Description)
		}
		root.WriteString("\n")
	}
	return writeFileAtomic(filepath.Join(bundle.RootPath, "index.md"), []byte(root.String()), 0o644)
}

func readCollectionConcepts(root string) ([]indexedConcept, error) {
	var concepts []indexedConcept
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") || entry.Name() == "index.md" || entry.Name() == "log.md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		meta, err := ParseDocumentMetadata(string(data))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if meta.Type == "" {
			return fmt.Errorf("%s: OKF concept is missing type", path)
		}
		concepts = append(concepts, indexedConcept{path: path, meta: meta})
		return nil
	})
	sort.Slice(concepts, func(i, j int) bool { return concepts[i].path < concepts[j].path })
	return concepts, err
}
