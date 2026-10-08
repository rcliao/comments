package comment

// LivingTemplate names the doc the agent keeps current while it builds: no
// verdict, no zones, read whenever the human wants the state.
const LivingTemplate = "living"

// IsLivingDoc reports whether content declares the living template.
func IsLivingDoc(content string) bool {
	meta, err := ParseDocumentMetadata(content)
	return err == nil && meta.Template == LivingTemplate
}

// SaveSeenBaseline records content as what the reader last saw. A living doc
// has no verdict, so closing the view moves the reader's baseline: the next
// open tints only what changed since they last looked.
func SaveSeenBaseline(docPath, reader, content string) error {
	return SaveReviewBaseline(docPath, reader, content)
}
