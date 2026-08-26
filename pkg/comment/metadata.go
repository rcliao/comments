package comment

import (
	"fmt"
	"strings"

	md "github.com/rcliao/comments/pkg/markdown"
)

// RelatedDocument is a producer extension layered on OKF. Generic OKF
// consumers preserve it; Comments uses it for deterministic context routing.
type RelatedDocument struct {
	Path     string `json:"path" yaml:"path"`
	Relation string `json:"relation,omitempty" yaml:"relation,omitempty"`
}

// KnowledgeSource is the subset of an OKF source needed for navigation.
type KnowledgeSource struct {
	ID       string `json:"id,omitempty"`
	Resource string `json:"resource"`
	Title    string `json:"title,omitempty"`
}

// DocumentMetadata is the queryable frontmatter surface shared by bundle,
// context, and template resolution.
type DocumentMetadata struct {
	Type        string            `json:"type,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status,omitempty"`
	Template    string            `json:"template,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Related     []RelatedDocument `json:"related,omitempty"`
	Sources     []KnowledgeSource `json:"sources,omitempty"`
	HasMetadata bool              `json:"-"`
}

// ParseDocumentMetadata parses OKF-compatible frontmatter plus the namespaced
// Comments template extension.
func ParseDocumentMetadata(content string) (DocumentMetadata, error) {
	fm, present, err := md.ParseFrontmatter(content)
	if err != nil {
		return DocumentMetadata{}, err
	}
	if !present {
		return DocumentMetadata{}, nil
	}
	meta := DocumentMetadata{
		Type:        md.FrontmatterString(fm.Values, "type"),
		Title:       md.FrontmatterString(fm.Values, "title"),
		Description: md.FrontmatterString(fm.Values, "description"),
		Status:      md.FrontmatterString(fm.Values, "status"),
		Tags:        md.FrontmatterStrings(fm.Values, "tags"),
		HasMetadata: true,
	}
	if comments, ok := stringMap(fm.Values["comments"]); ok {
		meta.Template = md.FrontmatterString(comments, "template")
	}
	meta.Related = parseRelated(fm.Values["related"])
	meta.Sources = parseSources(fm.Values["sources"])
	return meta, nil
}

func stringMap(value any) (map[string]any, bool) {
	switch mapped := value.(type) {
	case map[string]any:
		return mapped, true
	case map[any]any:
		out := make(map[string]any, len(mapped))
		for key, item := range mapped {
			text, ok := key.(string)
			if !ok {
				return nil, false
			}
			out[text] = item
		}
		return out, true
	default:
		return nil, false
	}
}

func listItems(value any) []any {
	switch list := value.(type) {
	case []any:
		return list
	case nil:
		return nil
	default:
		return []any{value}
	}
}

func parseRelated(value any) []RelatedDocument {
	var out []RelatedDocument
	for _, item := range listItems(value) {
		switch typed := item.(type) {
		case string:
			if path := strings.TrimSpace(typed); path != "" {
				out = append(out, RelatedDocument{Path: path})
			}
		default:
			if mapped, ok := stringMap(typed); ok {
				path := md.FrontmatterString(mapped, "path")
				if path != "" {
					out = append(out, RelatedDocument{Path: path, Relation: md.FrontmatterString(mapped, "relation")})
				}
			}
		}
	}
	return out
}

func parseSources(value any) []KnowledgeSource {
	var out []KnowledgeSource
	for _, item := range listItems(value) {
		mapped, ok := stringMap(item)
		if !ok {
			continue
		}
		resource := md.FrontmatterString(mapped, "resource")
		if resource == "" {
			continue
		}
		out = append(out, KnowledgeSource{
			ID:       md.FrontmatterString(mapped, "id"),
			Resource: resource,
			Title:    md.FrontmatterString(mapped, "title"),
		})
	}
	return out
}

// ValidateOKFMetadata applies the intentionally small OKF conformance floor
// for a concept document. Bundle index and log files are validated separately.
func ValidateOKFMetadata(content string) []Violation {
	meta, err := ParseDocumentMetadata(content)
	if err != nil {
		return []Violation{{Rule: "invalid_frontmatter", Line: 1, Message: err.Error()}}
	}
	if !meta.HasMetadata {
		return []Violation{{Rule: "missing_frontmatter", Line: 1, Message: "OKF concept is missing YAML frontmatter"}}
	}
	if meta.Type == "" {
		return []Violation{{Rule: "missing_type", Line: 1, Message: "OKF frontmatter requires a non-empty type"}}
	}
	if meta.Status != "" && meta.Status != "draft" && meta.Status != "stable" && meta.Status != "deprecated" {
		return []Violation{{Rule: "invalid_status", Line: 1, Message: fmt.Sprintf("OKF status %q must be draft, stable, or deprecated", meta.Status)}}
	}
	return nil
}
