package comment

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Parser handles parsing comments from markdown text
type Parser struct {
	commentRegex *regexp.Regexp
}

// NewParser creates a new comment parser
func NewParser() *Parser {
	// Regex to match: {>>[@author:id:line:timestamp] text <<}
	// Group 1: metadata (author:id:line:timestamp)
	// Group 2: comment text
	// Note: Use .*? (any character) instead of [^<]*? to allow < in comment text
	// The <<} closing marker is unambiguous enough
	pattern := `\{>>(?:\[@([^\]]+)\]\s*)?(.*?)<<\}`

	return &Parser{
		commentRegex: regexp.MustCompile(pattern),
	}
}

// Parse extracts comments from markdown content
// Returns the clean content (without comment markup) and extracted comments
func (p *Parser) Parse(content string) (*DocumentWithComments, error) {
	doc := &DocumentWithComments{
		Comments:  make([]*Comment, 0),
		Positions: make(map[string]Position),
	}

	lines := strings.Split(content, "\n")
	cleanLines := make([]string, len(lines))

	for lineNum, line := range lines {
		cleanLine, comments, err := p.parseLine(line, lineNum+1)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d: %w", lineNum+1, err)
		}

		cleanLines[lineNum] = cleanLine

		// Add comments and track their positions
		for _, c := range comments {
			doc.Comments = append(doc.Comments, c)
			doc.Positions[c.ID] = Position{
				Line: lineNum + 1,
				Column: strings.Index(line, "{>>"),
			}
		}
	}

	doc.Content = strings.Join(cleanLines, "\n")
	return doc, nil
}

// parseLine extracts comments from a single line
func (p *Parser) parseLine(line string, lineNum int) (string, []*Comment, error) {
	matches := p.commentRegex.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return line, nil, nil
	}

	comments := make([]*Comment, 0, len(matches))
	cleanLine := line

	for _, match := range matches {
		metadata := match[1]
		text := strings.TrimSpace(match[2])

		comment, err := p.parseComment(metadata, text, lineNum)
		if err != nil {
			return "", nil, err
		}

		comments = append(comments, comment)

		// Remove the comment markup from the line
		cleanLine = strings.Replace(cleanLine, match[0], "", 1)
	}

	// Normalize whitespace (collapse multiple spaces into one)
	cleanLine = regexp.MustCompile(`\s+`).ReplaceAllString(cleanLine, " ")
	return strings.TrimSpace(cleanLine), comments, nil
}

// parseComment creates a Comment from metadata and text
func (p *Parser) parseComment(metadata, text string, lineNum int) (*Comment, error) {
	comment := &Comment{
		Text: text,
		Line: lineNum,
	}

	// If no metadata, generate defaults
	if metadata == "" {
		id := generateID()
		comment.ID = id
		comment.ThreadID = id
		comment.ParentID = ""
		comment.Author = "unknown"
		comment.Timestamp = time.Now()
		return comment, nil
	}

	// Parse metadata with backward compatibility:
	// Check for resolved flag at the end (":true" or ":false")
	resolved := false
	if strings.HasSuffix(metadata, ":true") {
		resolved = true
		metadata = strings.TrimSuffix(metadata, ":true")
	} else if strings.HasSuffix(metadata, ":false") {
		metadata = strings.TrimSuffix(metadata, ":false")
	}

	colonCount := strings.Count(metadata, ":")

	// Format evolution:
	// V3 (current): author:id:threadid:line:timestamp:type:resolved (8+ colons with timestamp)
	// V2: author:id:threadid:line:timestamp:resolved (7+ colons)
	// V1: author:id:threadid:line:timestamp (6+ colons)
	// V0 (legacy): author:id:line:timestamp (5 colons)
	//
	// Since timestamps contain 2-3 colons (depending on timezone format), we count total colons

	if colonCount >= 8 {
		// V3 format: author:id:threadid:line:timestamp:type:resolved
		// We need to extract from the right because timestamp has variable colons
		//
		// Strategy: Split from the LEFT for fixed fields, then parse from RIGHT for variable fields
		// Format: author:id:threadid:line:<timestamp>:type:resolved
		//         0      1   2        3    <variable> N-1  N

		// First, split off the fixed left fields (author:id:threadid:line)
		parts := strings.SplitN(metadata, ":", 5) // Split into: author, id, threadid, line, rest
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid V3 format: expected at least 5 parts, got %d", len(parts))
		}

		comment.Author = parts[0]
		comment.ID = parts[1]
		comment.ThreadID = parts[2]
		// parts[3] is the line number (we use lineNum parameter instead)

		// If ThreadID is empty or equals ID, this is a root comment
		if comment.ThreadID == "" || comment.ThreadID == comment.ID {
			comment.ThreadID = comment.ID
			comment.ParentID = ""
		} else {
			// This is a reply
			comment.ParentID = comment.ThreadID
		}

		// Now parse from the right: parts[4] = "timestamp:type:resolved"
		remaining := parts[4]

		// Split from the right to get type and timestamp
		// Find the last colon (before :resolved)
		// Note: resolved was already stripped earlier, so we have "timestamp:type"
		lastColon := strings.LastIndex(remaining, ":")
		if lastColon == -1 {
			return nil, fmt.Errorf("invalid V3 format: missing type separator")
		}

		timestampStr := remaining[:lastColon]
		comment.Type = remaining[lastColon+1:]

		// Parse the timestamp
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp '%s': %w", timestampStr, err)
		}
		comment.Timestamp = timestamp
		comment.Resolved = resolved

	} else if colonCount >= 6 {
		// V2/V1 format: author:id:threadid:line:timestamp (possibly with :resolved)
		// Split into at most 6 parts to preserve timestamp colons
		parts := strings.SplitN(metadata, ":", 6)
		if len(parts) < 6 {
			return nil, fmt.Errorf("invalid V2 format metadata: expected 6 parts, got %d", len(parts))
		}

		comment.Author = parts[0]
		comment.ID = parts[1]
		comment.ThreadID = parts[2]
		// parts[3] is the line number (we use lineNum parameter instead)

		// If ThreadID is empty or equals ID, this is a root comment
		if comment.ThreadID == "" || comment.ThreadID == comment.ID {
			comment.ThreadID = comment.ID
			comment.ParentID = ""
		} else {
			// This is a reply
			comment.ParentID = comment.ThreadID
		}

		// parts[4] contains the date part, parts[5] contains time with colons
		// Reconstruct the full timestamp
		fullTimestamp := parts[4] + ":" + parts[5]

		timestamp, err := time.Parse(time.RFC3339, fullTimestamp)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp: %w", err)
		}
		comment.Timestamp = timestamp
		comment.Resolved = resolved
		comment.Type = "" // V2 doesn't have type

	} else if colonCount == 5 {
		// Old format (backward compatibility): Split into 4 parts
		parts := strings.SplitN(metadata, ":", 4)
		comment.Author = parts[0]
		comment.ID = parts[1]
		comment.ThreadID = parts[1] // Default: root comment (threadID = ID)
		comment.ParentID = ""
		// parts[2] is the line number (we use lineNum parameter instead)

		// Parse timestamp (ISO 8601 format)
		timestamp, err := time.Parse(time.RFC3339, parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp: %w", err)
		}
		comment.Timestamp = timestamp
		comment.Resolved = resolved

	} else {
		return nil, fmt.Errorf("invalid metadata format: expected author:id:threadid:line:timestamp or author:id:line:timestamp, got %s (found %d colons)", metadata, colonCount)
	}

	return comment, nil
}

// generateID creates a unique comment ID
func generateID() string {
	return fmt.Sprintf("c%d", time.Now().UnixNano())
}
