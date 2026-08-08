package comment

import "fmt"

// LoadDocument is the shared load prelude for every surface (CLI, TUI, MCP):
// read the sidecar, and when the load-time re-anchor cascade changed anything,
// persist the revalidated sidecar so the next read is clean.
//
// It lives here rather than in an adapter so the CLI and the MCP server cannot
// drift apart on load semantics — see the layering note in docs/ARCHITECTURE.md.
func LoadDocument(absPath string) (*DocumentWithComments, *LoadReport, error) {
	doc, report, err := LoadFromSidecar(absPath)
	if err != nil {
		return nil, nil, err
	}
	if report.Dirty {
		if err := SaveToSidecar(absPath, doc); err != nil {
			return nil, nil, fmt.Errorf("failed to persist re-anchored sidecar: %w", err)
		}
	}
	return doc, report, nil
}
