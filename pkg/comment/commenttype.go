package comment

import "strings"

// Comment types. The type is stored as a leading "[X]" marker inside the
// comment text (see the --type flag), not as a separate field on the wire, so
// display decorates the text rather than rewriting it — sidecar payloads and
// the --type filters keep working untouched.
const (
	TypeQuestion    = "Q"
	TypeSuggestion  = "S"
	TypeBug         = "B"
	TypeTODO        = "T"
	TypeEnhancement = "E"
)

// TypeOrder is the canonical listing order for help text and validation errors.
var TypeOrder = []string{TypeQuestion, TypeSuggestion, TypeBug, TypeTODO, TypeEnhancement}

var typeEmoji = map[string]string{
	TypeQuestion:    "❓",
	TypeSuggestion:  "💡",
	TypeBug:         "🐛",
	TypeTODO:        "📌",
	TypeEnhancement: "✨",
}

var typeName = map[string]string{
	TypeQuestion:    "Question",
	TypeSuggestion:  "Suggestion",
	TypeBug:         "Bug",
	TypeTODO:        "TODO",
	TypeEnhancement: "Enhancement",
}

// IsValidType reports whether t is one of the five comment types.
func IsValidType(t string) bool {
	_, ok := typeEmoji[t]
	return ok
}

// TypeEmoji returns the display emoji for a comment type, or "" if the type is
// not recognized.
func TypeEmoji(t string) string { return typeEmoji[t] }

// TypeName returns the human-readable name for a comment type, or "" if the
// type is not recognized.
func TypeName(t string) string { return typeName[t] }

// TypesHelp renders the type list for CLI help and error messages, e.g.
// "Q (Question), S (Suggestion), ...".
func TypesHelp() string {
	parts := make([]string, 0, len(TypeOrder))
	for _, t := range TypeOrder {
		parts = append(parts, t+" ("+typeName[t]+")")
	}
	return strings.Join(parts, ", ")
}

// PrefixType prepends the "[T] " marker to text, normalizing rather than
// stacking: text that already leads with the same marker is returned as-is
// (agents habitually write "[Q] ..." AND pass type=Q — stored text was
// doubling to "[Q] [Q] ..."). A DIFFERENT leading marker is left in place and
// the explicit type wins in front of it, which keeps both signals visible for
// the human to untangle rather than silently dropping one.
func PrefixType(text, t string) string {
	if t == "" {
		return text
	}
	if lead, ok := LeadingType(text); ok && lead == t {
		return text
	}
	return "[" + t + "] " + text
}

// DecorateType prefixes text that opens with a "[X]" type marker with the
// matching emoji, e.g. "[Q] Why?" becomes "❓ [Q] Why?". Text whose marker is
// absent or unrecognized is returned unchanged, and the marker itself is kept
// so --type filtering and existing sidecars stay valid.
func DecorateType(text string) string {
	t, ok := LeadingType(text)
	if !ok {
		return text
	}
	return typeEmoji[t] + " " + text
}

// LeadingType extracts the type letter from a "[X]" marker at the start of
// text, reporting false when there is no recognized marker.
func LeadingType(text string) (string, bool) {
	trimmed := strings.TrimLeft(text, " \t")
	if len(trimmed) < 3 || trimmed[0] != '[' || trimmed[2] != ']' {
		return "", false
	}
	t := string(trimmed[1])
	if !IsValidType(t) {
		return "", false
	}
	return t, true
}
