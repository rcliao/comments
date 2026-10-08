package comment

import "fmt"

// ReplySpec is one response to a thread. Resolve closes the thread after the
// reply lands; Text may be empty only when resolving.
type ReplySpec struct {
	ThreadID string `json:"thread_id" jsonschema:"Thread ID (a reply ID resolves to its thread)"`
	Author   string `json:"author,omitempty" jsonschema:"Author of the reply; required whenever text is given"`
	Text     string `json:"text,omitempty" jsonschema:"Reply text; may be empty only when resolve is true"`
	Resolve  bool   `json:"resolve,omitempty" jsonschema:"Also resolve the thread. Refused for an agent in a template's zone: human section — reply and leave it to the human"`
}

// ReplyResult reports what a reply call did, by thread ID.
type ReplyResult struct {
	Count    int      `json:"count"`
	Replied  []string `json:"replied"`
	Resolved []string `json:"resolved"`
}

// ReplyToThreads is the only respond path: replying and resolving were two
// commands that agents always ran as a pair ("reply with what changed, then
// resolve"), and a standalone resolve let a thread close with no explanation.
//
// It carries the zone guard: an agent may not resolve a thread in a template's
// `zone: human` section. The batch is atomic — every spec is checked, guard
// included, before any thread is touched — so a refused resolve cannot leave
// half a batch applied.
func ReplyToThreads(doc *DocumentWithComments, absPath string, specs []ReplySpec, actor Actor) (*ReplyResult, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("no replies given")
	}
	roots := make([]*Comment, len(specs))
	for i, spec := range specs {
		where := fmt.Sprintf("reply %d", i+1)
		switch {
		case spec.ThreadID == "":
			return nil, fmt.Errorf("%s: thread_id is required", where)
		case spec.Text == "" && !spec.Resolve:
			return nil, fmt.Errorf("%s: text is required (it may be empty only when resolving)", where)
		case spec.Text != "" && spec.Author == "":
			return nil, fmt.Errorf("%s: author is required", where)
		}
		root := FindThreadContaining(doc.Threads, spec.ThreadID)
		if root == nil {
			return nil, fmt.Errorf("%s: thread not found: %s", where, spec.ThreadID)
		}
		if spec.Resolve {
			// Only the human's approval settles a pick (AddReviewRecord); an
			// agent closing its own pick would pass the strict gate unseen.
			if root.Pick != "" && !root.Resolved && actor != ActorHuman {
				return nil, fmt.Errorf("%s: thread %s is a pick; the human's approval settles it — reply without resolving, or file a new pick", where, root.ID)
			}
			if err := GuardZoneResolve(doc, absPath, root.ID, actor); err != nil {
				return nil, fmt.Errorf("%s: %w", where, err)
			}
		}
		roots[i] = root
	}

	result := &ReplyResult{Replied: []string{}, Resolved: []string{}}
	for i, spec := range specs {
		root := roots[i]
		if spec.Text != "" {
			if err := AddReplyToThread(doc.Threads, root.ID, spec.Author, spec.Text); err != nil {
				return nil, err
			}
			result.Replied = append(result.Replied, root.ID)
		}
		if spec.Resolve {
			if err := ResolveThread(doc.Threads, root.ID); err != nil {
				return nil, err
			}
			result.Resolved = append(result.Resolved, root.ID)
		}
	}
	result.Count = len(specs)
	return result, nil
}
