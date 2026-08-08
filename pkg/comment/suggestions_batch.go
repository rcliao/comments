package comment

// AcceptResult reports the outcome of accepting one suggestion in a batch.
// Failures are per-suggestion so one bad ID cannot discard the rest.
type AcceptResult struct {
	ID       string `json:"id"`
	Accepted bool   `json:"accepted"`
	Error    string `json:"error,omitempty"`
}

// SelectPendingSuggestions returns the IDs of pending (undecided) suggestions
// matching the given author and/or type letter. Empty filters match everything,
// so at least one filter should be supplied by the caller.
func SelectPendingSuggestions(doc *DocumentWithComments, author, typeLetter string) []string {
	var ids []string
	for _, c := range doc.GetAllComments() {
		if !c.IsSuggestion || c.Accepted != nil {
			continue
		}
		if author != "" && c.Author != author {
			continue
		}
		if typeLetter != "" {
			if t, ok := LeadingType(c.Text); !ok || t != typeLetter {
				continue
			}
		}
		ids = append(ids, c.ID)
	}
	return ids
}

// AcceptSuggestions applies and accepts each suggestion in order. Each accept
// shifts the lines of comments the edit displaced, so the ranges of the
// remaining suggestions stay valid as the batch proceeds.
//
// The caller is responsible for persisting both the markdown and the sidecar.
func AcceptSuggestions(doc *DocumentWithComments, ids []string) []AcceptResult {
	results := make([]AcceptResult, 0, len(ids))
	for _, id := range ids {
		if _, err := ApplyAndAcceptSuggestion(doc, id); err != nil {
			results = append(results, AcceptResult{ID: id, Error: err.Error()})
			continue
		}
		results = append(results, AcceptResult{ID: id, Accepted: true})
	}
	return results
}
