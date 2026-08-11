package comment

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StorageVersion is the current version of the JSON storage format
const StorageVersion = "2.0"

// StorageFormat represents the JSON sidecar file structure (v2.0)
type StorageFormat struct {
	Version       string         `json:"version"`            // Format version ("2.0")
	DocumentHash  string         `json:"documentHash"`       // SHA-256 hash for staleness detection
	LastValidated time.Time      `json:"lastValidated"`      // Last validation timestamp
	Threads       []*Comment     `json:"threads"`            // Root comment threads with nested replies
	Reviews       []ReviewRecord `json:"reviews,omitempty"`  // Completed review passes (signoffs)
	Template      string         `json:"template,omitempty"` // Doc template governing this document
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

// LoadReport describes what LoadFromSidecar observed and changed in memory
// while loading. Loading never prints and never writes files: callers decide
// whether to surface Issues to the user and whether to persist a Dirty
// document back to disk with SaveToSidecar.
type LoadReport struct {
	Stale         bool              // Markdown content changed since the sidecar was last written
	OrphanedCount int               // Comments newly marked orphaned during this load
	Issues        []ValidationIssue // Staleness, orphaning, and re-anchoring details
	Dirty         bool              // In-memory doc differs from the sidecar on disk (persist with SaveToSidecar)
}

// LoadFromSidecar loads comments from the sidecar JSON file (v2.0).
// Returns the DocumentWithComments plus a LoadReport describing staleness,
// re-anchoring, and orphaning discovered during the load. This is a pure read:
// it never prints and never writes files. When report.Dirty is true the caller
// owns the decision to persist the migrated document via SaveToSidecar.
func LoadFromSidecar(mdPath string) (*DocumentWithComments, *LoadReport, error) {
	report := &LoadReport{}

	// Read markdown content
	contentBytes, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read markdown file: %w", err)
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
		return doc, report, nil
	}

	// Read sidecar file
	sidecarBytes, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read sidecar file: %w", err)
	}

	// Parse JSON
	var storage StorageFormat
	if err := json.Unmarshal(sidecarBytes, &storage); err != nil {
		return nil, nil, fmt.Errorf("failed to parse sidecar JSON: %w", err)
	}

	// Validate version (v2.0 only)
	if storage.Version != "2.0" {
		return nil, nil, fmt.Errorf("unsupported storage version: %s (expected 2.0)", storage.Version)
	}

	// Populate document with loaded data
	doc.Threads = storage.Threads
	doc.DocumentHash = storage.DocumentHash
	doc.LastValidated = storage.LastValidated
	doc.Reviews = storage.Reviews
	doc.Template = storage.Template

	// Staleness: has the markdown changed since the sidecar was last written?
	report.Stale = storage.DocumentHash != contentHash

	// Migrate old format comments to new format (adds default values for Status, Priority, etc.)
	doc.MigrateDocument()

	// Validate comments and mark orphaned ones (granular validation, in memory only)
	report.OrphanedCount, report.Issues = ValidateAndUpdateCommentStatus(doc)

	// Update hash and timestamp to current values (in memory)
	doc.DocumentHash = contentHash
	doc.LastValidated = time.Now()

	// The sidecar on disk is out of date when validation changed anything
	// (re-anchored lines, orphaned statuses, refreshed hash after an edit)
	report.Dirty = report.OrphanedCount > 0 || len(report.Issues) > 0

	return doc, report, nil
}

// FormatLoadReport renders the human-facing warnings for a load, matching the
// wording the CLI has always printed. Returns "" when there is nothing to say.
func FormatLoadReport(report *LoadReport) string {
	if report == nil {
		return ""
	}
	var b strings.Builder
	if report.OrphanedCount > 0 {
		fmt.Fprintf(&b, "Warning: %d comment(s) marked as orphaned due to document changes\n", report.OrphanedCount)
		fmt.Fprintf(&b, "%s\n", FormatValidationIssues(report.Issues))
		fmt.Fprintf(&b, "Use './comments list --status orphaned' to view orphaned comments\n")
		return b.String()
	}
	// Report non-orphaning issues (like section moves)
	for _, issue := range report.Issues {
		if issue.Severity == "info" {
			fmt.Fprintf(&b, "Info: %s\n", issue.Message)
		}
	}
	return b.String()
}

// ResolveThreadCitation parses a thread citation exactly as it appears in a
// document — thread:c1abc or thread:path.md#c1abc — into (docPath, threadID).
// Relative citation paths resolve against fromDoc's directory (citations are
// written relative to the CITING doc; resolving them against an agent's cwd
// silently reads the wrong sidecar). A bare same-doc citation returns fromDoc.
func ResolveThreadCitation(cite, fromDoc string) (string, string, error) {
	rest, ok := strings.CutPrefix(cite, "thread:")
	if !ok {
		return "", "", fmt.Errorf("not a thread citation: %q (want thread:c1abc or thread:path.md#c1abc)", cite)
	}
	path, id, hasPath := strings.Cut(rest, "#")
	if !hasPath {
		if fromDoc == "" {
			return "", "", fmt.Errorf("same-doc citation %q needs the citing document (--from)", cite)
		}
		return fromDoc, path, nil
	}
	if !filepath.IsAbs(path) && fromDoc != "" {
		path = filepath.Join(filepath.Dir(fromDoc), path)
	}
	return path, id, nil
}

// ReadThread fetches one thread from a document's sidecar by ID, RAW — no
// re-anchoring, no validation side effects. Built for thread citations
// (thread:c1abc peek): a read path that must work from any doc's context.
func ReadThread(mdPath, id string) (*Comment, error) {
	data, err := os.ReadFile(GetSidecarPath(mdPath))
	if err != nil {
		return nil, fmt.Errorf("no sidecar for %s", mdPath)
	}
	var storage StorageFormat
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("unreadable sidecar for %s", mdPath)
	}
	if c := findCommentByID(storage.Threads, id); c != nil {
		// Return the containing ROOT so the peek shows the whole debate even
		// when the citation names a reply
		if root := FindThreadContaining(storage.Threads, id); root != nil {
			return root, nil
		}
		return c, nil
	}
	return nil, fmt.Errorf("thread %s not found in %s", id, mdPath)
}

// SaveDocumentContent writes doc.Content to the markdown file atomically.
// Call it ONLY from paths that legitimately changed the content (suggestion
// accepts). It is deliberately separate from SaveToSidecar: a signoff or a
// reply from a TUI session holding stale content must never overwrite edits
// made on disk since that session loaded (live lost-update, found in
// dogfooding when a signoff reverted an agent's edits).
func SaveDocumentContent(mdPath string, doc *DocumentWithComments) error {
	current, readErr := os.ReadFile(mdPath)
	if readErr == nil && string(current) == doc.Content {
		return nil
	}
	if err := writeFileAtomic(mdPath, []byte(doc.Content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}
	return nil
}

// SaveToSidecar saves comment threads to the sidecar JSON file (v2.0). It
// writes ONLY the sidecar — the markdown file belongs to SaveDocumentContent.
func SaveToSidecar(mdPath string, doc *DocumentWithComments) error {
	// Recompute document hash
	doc.DocumentHash = ComputeDocumentHash(doc.Content)
	doc.LastValidated = time.Now()

	// Guarantee ID uniqueness across all write paths
	EnsureUniqueIDs(doc)

	// Prepare storage format
	storage := StorageFormat{
		Version:       StorageVersion,
		DocumentHash:  doc.DocumentHash,
		LastValidated: doc.LastValidated,
		Threads:       doc.Threads,
		Reviews:       doc.Reviews,
		Template:      doc.Template,
	}

	// Marshal to JSON with indentation for readability
	jsonBytes, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comments to JSON: %w", err)
	}

	// Write sidecar file
	sidecarPath := GetSidecarPath(mdPath)
	if err := writeFileAtomic(sidecarPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write sidecar file: %w", err)
	}

	return nil
}

// writeFileAtomic writes data to path via a temp file in the same directory
// followed by os.Rename, so readers never observe a partially written file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// SidecarExists checks if a sidecar file exists for the given markdown file
func SidecarExists(mdPath string) bool {
	sidecarPath := GetSidecarPath(mdPath)
	_, err := os.Stat(sidecarPath)
	return err == nil
}
