package comment

import (
	"fmt"
	"strings"
)

// Serializer handles converting comments back to CriticMarkup format
type Serializer struct{}

// NewSerializer creates a new comment serializer
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serialize inserts comments back into the document content
func (s *Serializer) Serialize(content string, comments []*Comment, positions map[string]Position) (string, error) {
	if len(comments) == 0 {
		return content, nil
	}

	lines := strings.Split(content, "\n")

	// Group comments by line
	commentsByLine := make(map[int][]*Comment)
	for _, c := range comments {
		pos, ok := positions[c.ID]
		if !ok {
			// Fallback to comment's line field
			pos = Position{Line: c.Line}
		}
		commentsByLine[pos.Line] = append(commentsByLine[pos.Line], c)
	}

	// Insert comments into each line (process from end to start to preserve positions)
	result := make([]string, len(lines))
	for i, line := range lines {
		lineNum := i + 1
		lineComments := commentsByLine[lineNum]

		if len(lineComments) == 0 {
			result[i] = line
			continue
		}

		// Serialize all comments for this line
		commentMarkup := make([]string, 0, len(lineComments))
		for _, c := range lineComments {
			markup := s.serializeComment(c)
			commentMarkup = append(commentMarkup, markup)
		}

		// Append comments to the end of the line
		result[i] = line + " " + strings.Join(commentMarkup, " ")
	}

	return strings.Join(result, "\n"), nil
}

// serializeComment converts a single comment to CriticMarkup format
func (s *Serializer) serializeComment(c *Comment) string {
	// Format: {>>[@author:id:threadid:line:timestamp] text <<}
	metadata := fmt.Sprintf("@%s:%s:%s:%d:%s",
		c.Author,
		c.ID,
		c.ThreadID,
		c.Line,
		c.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	)

	return fmt.Sprintf("{>>[%s] %s <<}", metadata, c.Text)
}

// SerializeComment exports a comment to CriticMarkup format (for testing/debugging)
func SerializeComment(c *Comment) string {
	s := NewSerializer()
	return s.serializeComment(c)
}
