package comment

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StorageVersion is the current version of the JSON storage format
const StorageVersion = "2.0"

// StorageFormat represents the JSON sidecar file structure (v2.0)
type StorageFormat struct {
	Version       string     `json:"version"`        // Format version ("2.0")
	DocumentHash  string     `json:"documentHash"`   // SHA-256 hash for staleness detection
	LastValidated time.Time  `json:"lastValidated"`  // Last validation timestamp
	Threads       []*Comment `json:"threads"`        // Root comment threads with nested replies
}

// GetSidecarPath returns the sidecar JSON path for a given markdown file
func GetSidecarPath(mdPath string) string {
	return mdPath + ".comments.json"
}

// ComputeDocumentHash computes SHA-256 hash of markdown content
func ComputeDocumentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// LoadFromSidecar loads comments from the sidecar JSON file (v2.0)
// Returns DocumentWithComments with the markdown content and parsed threads
func LoadFromSidecar(mdPath string) (*DocumentWithComments, error) {
	// Read markdown content
	contentBytes, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown file: %w", err)
	}

	content := string(contentBytes)
	contentHash := ComputeDocumentHash(content)

	// Initialize empty document
	doc := &DocumentWithComments{
		Content:       content,
		Threads:       []*Comment{},
		DocumentHash:  contentHash,
		LastValidated: time.Now(),
	}

	// Read sidecar JSON file
	sidecarPath := GetSidecarPath(mdPath)

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

	// Validate version (v2.0 only)
	if storage.Version != "2.0" {
		return nil, fmt.Errorf("unsupported storage version: %s (expected 2.0)", storage.Version)
	}

	// Populate document with loaded data
	doc.Threads = storage.Threads
	doc.DocumentHash = storage.DocumentHash
	doc.LastValidated = storage.LastValidated

	// Validate the sidecar against the current document
	isValid, issues, err := ValidateSidecar(doc)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// If validation failed, archive stale sidecar and return empty document
	if !isValid {
		// Archive the stale sidecar
		if err := ArchiveStaleSidecar(mdPath); err != nil {
			// Log warning but continue with empty document
			fmt.Fprintf(os.Stderr, "Warning: Failed to archive stale sidecar: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Sidecar file is stale and has been archived\n")
			fmt.Fprintf(os.Stderr, "%s\n", FormatValidationIssues(issues))
		}

		// Return empty document with current hash
		return &DocumentWithComments{
			Content:       content,
			Threads:       []*Comment{},
			DocumentHash:  contentHash,
			LastValidated: time.Now(),
		}, nil
	}

	// Update hash and timestamp to current values
	doc.DocumentHash = contentHash
	doc.LastValidated = time.Now()

	return doc, nil
}

// SaveToSidecar saves comment threads to the sidecar JSON file (v2.0)
// Also writes the clean markdown content (without comment markup)
func SaveToSidecar(mdPath string, doc *DocumentWithComments) error {
	// Write markdown content
	if err := os.WriteFile(mdPath, []byte(doc.Content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	// Recompute document hash
	doc.DocumentHash = ComputeDocumentHash(doc.Content)
	doc.LastValidated = time.Now()

	// Prepare storage format
	storage := StorageFormat{
		Version:       StorageVersion,
		DocumentHash:  doc.DocumentHash,
		LastValidated: doc.LastValidated,
		Threads:       doc.Threads,
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

// ArchiveStaleSidecar renames a stale sidecar to .backup with timestamp
func ArchiveStaleSidecar(mdPath string) error {
	sidecarPath := GetSidecarPath(mdPath)
	if _, err := os.Stat(sidecarPath); os.IsNotExist(err) {
		return nil // Nothing to archive
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", sidecarPath, timestamp)

	// Rename the file
	if err := os.Rename(sidecarPath, backupPath); err != nil {
		return fmt.Errorf("failed to archive sidecar: %w", err)
	}

	return nil
}
