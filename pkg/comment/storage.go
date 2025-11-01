package comment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// StorageVersion is the current version of the JSON storage format
const StorageVersion = "1.0"

// StorageFormat represents the JSON sidecar file structure
type StorageFormat struct {
	Version   string               `json:"version"`   // Format version for future migrations
	Comments  []*Comment           `json:"comments"`  // All comments
	Positions map[string]Position  `json:"positions"` // Comment ID to position mapping
}

// GetSidecarPath returns the sidecar JSON path for a given markdown file
func GetSidecarPath(mdPath string) string {
	return mdPath + ".comments.json"
}

// LoadFromSidecar loads comments from the sidecar JSON file
// Returns DocumentWithComments with the markdown content and parsed comments
func LoadFromSidecar(mdPath string) (*DocumentWithComments, error) {
	// Read markdown content
	contentBytes, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Read sidecar JSON file
	sidecarPath := GetSidecarPath(mdPath)
	doc := &DocumentWithComments{
		Content:   string(contentBytes),
		Comments:  []*Comment{},
		Positions: make(map[string]Position),
	}

	// Check if sidecar exists
	if _, err := os.Stat(sidecarPath); os.IsNotExist(err) {
		// No sidecar file exists - return empty document
		return doc, nil
	}

	// Read sidecar file
	sidecarBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar file: %w", err)
	}

	// Parse JSON
	var storage StorageFormat
	if err := json.Unmarshal(sidecarBytes, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse sidecar JSON: %w", err)
	}

	// Validate version
	if storage.Version != StorageVersion {
		return nil, fmt.Errorf("unsupported storage version: %s (expected %s)", storage.Version, StorageVersion)
	}

	// Populate document
	doc.Comments = storage.Comments
	if storage.Positions != nil {
		doc.Positions = storage.Positions
	}

	return doc, nil
}

// SaveToSidecar saves comments to the sidecar JSON file
// Also writes the clean markdown content (without comment markup)
func SaveToSidecar(mdPath string, doc *DocumentWithComments) error {
	// Write markdown content
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	// Prepare storage format
	storage := StorageFormat{
		Version:   StorageVersion,
		Comments:  doc.Comments,
		Positions: doc.Positions,
	}

	// Marshal to JSON with indentation for readability
	jsonBytes, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comments to JSON: %w", err)
	}

	// Write sidecar file
	sidecarPath := GetSidecarPath(mdPath)
	if err := os.WriteFile(sidecarPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar file: %w", err)
	}

	return nil
}

// DeleteSidecar removes the sidecar JSON file if it exists
func DeleteSidecar(mdPath string) error {
	sidecarPath := GetSidecarPath(mdPath)
	if _, err := os.Stat(sidecarPath); os.IsNotExist(err) {
		return nil // Nothing to delete
	}
	return os.Remove(sidecarPath)
}

// SidecarExists checks if a sidecar file exists for the given markdown file
func SidecarExists(mdPath string) bool {
	sidecarPath := GetSidecarPath(mdPath)
	_, err := os.Stat(sidecarPath)
	return err == nil
}

// ListSidecars finds all sidecar files in a directory
func ListSidecars(dir string) ([]string, error) {
	var sidecars []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".json" && len(name) > len(".comments.json") {
			// Check if it ends with .comments.json
			if name[len(name)-len(".comments.json"):] == ".comments.json" {
				sidecars = append(sidecars, filepath.Join(dir, name))
			}
		}
	}

	return sidecars, nil
}
