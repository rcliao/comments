package tui

// Position persistence: reopening a document resumes the review where it
// left off. State lives in .comments/view-state.json next to the document,
// keyed by filename — the sidecar directory pattern, no pkg/comment changes.

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// viewState is the per-document reading position persisted across sessions
type viewState struct {
	SelectedLine    int  `json:"selectedLine"`
	YOffset         int  `json:"yOffset"`
	HideLineNumbers bool `json:"hideLineNumbers,omitempty"`
}

// viewStatePath returns the state file path for a document: a shared
// .comments/view-state.json in the document's directory
func viewStatePath(docFilename string) string {
	return filepath.Join(filepath.Dir(docFilename), ".comments", "view-state.json")
}

// loadViewState reads the saved position for a document, if any
func loadViewState(docFilename string) (viewState, bool) {
	data, err := os.ReadFile(viewStatePath(docFilename))
	if err != nil {
		return viewState{}, false
	}
	all := map[string]viewState{}
	if err := json.Unmarshal(data, &all); err != nil {
		return viewState{}, false
	}
	st, ok := all[filepath.Base(docFilename)]
	return st, ok
}

// saveViewState upserts the position for a document, creating the
// .comments directory and state file as needed
func saveViewState(docFilename string, st viewState) error {
	path := viewStatePath(docFilename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	all := map[string]viewState{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &all) // corrupt state file: start over
	}
	all[filepath.Base(docFilename)] = st
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveViewStateNow persists the current reading position; best-effort on quit
func (m *Model) saveViewStateNow() {
	if m.doc == nil || m.filename == "" {
		return
	}
	_ = saveViewState(m.filename, viewState{
		SelectedLine:    m.selectedLine,
		HideLineNumbers: m.hideLineNumbers,
		YOffset:         m.documentViewport.YOffset(),
	})
}
