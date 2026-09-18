package comment

import (
	"fmt"
	"strings"
)

// ThreadList is every thread in a document — the history view. The worklist is
// the inbox; this is for looking something up, resolved threads included.
type ThreadList struct {
	File     string        `json:"filepath"`
	Total    int           `json:"total"`
	Comments []CommentView `json:"comments"`
}

// ThreadDetail is one comment with the document lines around it.
type ThreadDetail struct {
	Comment      CommentView `json:"comment"`
	SectionPath  string      `json:"section_path,omitempty"`
	ContextLines []string    `json:"context_lines,omitempty"`
	IsOrphaned   bool        `json:"is_orphaned,omitempty"`
}

// ListThreads returns the document's thread roots with replies nested.
// Resolved threads are included unless unresolvedOnly is set: this is the
// lookup path, and what a lookup most often wants is a thread already closed.
func ListThreads(doc *DocumentWithComments, file string, unresolvedOnly bool) *ThreadList {
	list := &ThreadList{File: file, Comments: []CommentView{}}
	for _, thread := range doc.Threads {
		if unresolvedOnly && thread.Resolved {
			continue
		}
		list.Comments = append(list.Comments, NewCommentView(thread))
	}
	list.Total = len(list.Comments)
	return list
}

// GetThread returns one comment (thread root or reply) with contextSize
// document lines on each side.
func GetThread(doc *DocumentWithComments, id string, contextSize int) (*ThreadDetail, error) {
	found := doc.FindCommentByID(id)
	if found == nil {
		return nil, fmt.Errorf("comment not found: %s", id)
	}
	return &ThreadDetail{
		Comment:      NewCommentView(found),
		SectionPath:  found.SectionPath,
		ContextLines: contextLines(doc.Content, found.Line, contextSize),
		IsOrphaned:   found.Status == "orphaned",
	}, nil
}

func contextLines(content string, targetLine, contextSize int) []string {
	lines := strings.Split(content, "\n")
	start := max(0, targetLine-contextSize-1)
	end := min(len(lines), targetLine+contextSize)
	out := []string{}
	for i := start; i < end; i++ {
		out = append(out, lines[i])
	}
	return out
}
