package comment

import (
	"strings"
	"testing"
)

func TestLeadingType(t *testing.T) {
	tests := []struct {
		text     string
		wantType string
		wantOK   bool
	}{
		{"[Q] why?", "Q", true},
		{"[S] consider x", "S", true},
		{"[B] broken", "B", true},
		{"[T] todo", "T", true},
		{"[E] nicer", "E", true},
		{"  [Q] leading space", "Q", true},
		{"[Z] unknown letter", "", false},
		{"no marker at all", "", false},
		{"", "", false},
		{"[Q", "", false},
		{"text with [Q] in the middle", "", false},
	}
	for _, tc := range tests {
		gotType, gotOK := LeadingType(tc.text)
		if gotType != tc.wantType || gotOK != tc.wantOK {
			t.Errorf("LeadingType(%q) = (%q, %v), want (%q, %v)",
				tc.text, gotType, gotOK, tc.wantType, tc.wantOK)
		}
	}
}

func TestDecorateType(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"[Q] why?", "❓ [Q] why?"},
		{"[S] consider", "💡 [S] consider"},
		{"[B] broken", "🐛 [B] broken"},
		{"[T] todo", "📌 [T] todo"},
		{"[E] nicer", "✨ [E] nicer"},
		// Unmarked text is untouched, so replies and plain comments stay clean
		{"just a reply", "just a reply"},
		{"[Z] unknown", "[Z] unknown"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := DecorateType(tc.text); got != tc.want {
			t.Errorf("DecorateType(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

// The marker must survive decoration: --type filters and every existing
// sidecar match on the "[X]" prefix of the stored text.
func TestDecorateTypeKeepsMarker(t *testing.T) {
	decorated := DecorateType("[Q] why?")
	if !strings.Contains(decorated, "[Q]") {
		t.Errorf("decorated text lost its type marker: %q", decorated)
	}
}

// Decoration is idempotent by construction: the emoji displaces the "[", so a
// second pass finds no marker. This is what makes the CommentView layer safe
// even if a caller decorates again by mistake.
func TestDecorateTypeIsIdempotent(t *testing.T) {
	once := DecorateType("[Q] why?")
	if twice := DecorateType(once); twice != once {
		t.Errorf("double decoration changed the text: %q -> %q", once, twice)
	}
}
