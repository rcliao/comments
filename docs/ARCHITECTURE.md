# Architecture Documentation

**Version:** 2.0
**Last Updated:** 2025-11-03
**Status:** Current

## Overview

`comments` is a CLI tool for collaborative markdown document editing with inline comments and suggestions, designed for seamless LLM integration. Instead of having LLMs rewrite entire documents, you can add contextual comments at specific lines or markdown sections to guide iteration and discussion, or propose multi-line edits as suggestions with preview and accept/reject workflow.

**Key Philosophy:** Clean separation of content and collaboration metadata through JSON sidecar storage.

## Storage Architecture (v2.0)

### Sidecar JSON Storage

Comments and suggestions are stored in `.md.comments.json` sidecar files alongside markdown documents:

**Benefits:**
- Markdown files remain clean and readable
- Structured metadata storage with efficient querying
- Independent version control of content vs. comments
- Document staleness detection via SHA-256 hashing

**Storage Format v2.0:**
```json
{
  "version": "2.0",
  "documentHash": "sha256_hash_of_markdown_content",
  "lastValidated": "2025-11-03T17:39:51Z",
  "threads": [
    {
      "ID": "c123",
      "Author": "alice",
      "Timestamp": "2025-11-03T10:30:00Z",
      "Text": "[Q] What about edge cases?",
      "Type": "Q",
      "Line": 10,
      "SectionID": "s2",
      "SectionPath": "Introduction > Overview",
      "Resolved": false,
      "Replies": [
        {
          "ID": "c124",
          "Author": "bob",
          "Timestamp": "2025-11-03T11:00:00Z",
          "Text": "Good question, let me add tests",
          "Line": 10,
          "Replies": []
        }
      ],
      "IsSuggestion": false
    },
    {
      "ID": "s456",
      "Author": "claude",
      "Text": "Improve clarity",
      "Line": 15,
      "IsSuggestion": true,
      "StartLine": 15,
      "EndLine": 17,
      "OriginalText": "old multi-line text",
      "ProposedText": "new improved text",
      "Accepted": null,
      "Replies": []
    }
  ]
}
```

### Key v2.0 Changes

**Nested Thread Structure:**
- Comments are stored as trees with `Replies` arrays
- Removed flat array with `ThreadID`/`ParentID` linking
- Simplified thread operations (no BuildThreads() needed)

**Document Hashing:**
- SHA-256 hash of markdown content stored in sidecar
- Automatic staleness detection on load
- Stale sidecars archived to `.backup.TIMESTAMP` files

**Simplified Position Tracking:**
- Line-only tracking (removed `Column` and `ByteOffset`)
- Positions stored directly in Comment struct (removed separate map)
- Recalculation only when accepting suggestions

**Simplified Suggestions:**
- Multi-line only (removed line, char-range, diff-hunk types)
- Boolean acceptance state: `*bool` pointer (nil=pending, true=accepted, false=rejected)
- @filename support for reading text from external files

**Helper Functions:**
- `AddReplyToThread()` - Add reply to existing thread
- `ResolveThread()` - Mark thread as resolved
- `AcceptSuggestion()` - Apply suggestion and update document
- `RejectSuggestion()` - Reject suggestion
- Removed `BuildThreads()` - no longer needed with nested structure

## Data Model

### Core Types

#### Comment (pkg/comment/types.go)

```go
type Comment struct {
    // Identity
    ID        string    // Unique timestamp-based ID
    Author    string    // Author name
    Timestamp time.Time // Creation timestamp

    // Content
    Text      string    // Comment text
    Type      string    // Q, S, B, T, E (optional categorization)

    // Position
    Line      int       // Line number in document

    // Section metadata (computed from markdown structure)
    SectionID   string  // Section identifier (e.g., "s1", "s2")
    SectionPath string  // Hierarchical path (e.g., "Intro > Overview")

    // State
    Resolved bool      // Resolution status

    // Threading (v2.0 nested structure)
    Replies []*Comment // Nested replies (empty for leaf comments)

    // Suggestion fields (optional)
    IsSuggestion bool    // True if this is an edit suggestion
    StartLine    int     // Start line for multi-line suggestion
    EndLine      int     // End line for multi-line suggestion
    OriginalText string  // Text being replaced
    ProposedText string  // Proposed replacement
    Accepted     *bool   // nil=pending, true=accepted, false=rejected
}
```

**Comment Types:**
- `Q` - Question: Requests clarification
- `S` - Suggestion: Proposes a change
- `B` - Bug: Identifies an issue
- `T` - TODO: Marks something to be done
- `E` - Enhancement: Suggests a feature

#### DocumentWithComments (pkg/comment/types.go)

```go
type DocumentWithComments struct {
    Content   string     // Raw markdown content
    Threads   []*Comment // Root comments with nested replies
    Hash      string     // SHA-256 hash for staleness detection
}
```

#### Section (pkg/markdown/types.go)

```go
type Section struct {
    ID        string     // Unique identifier (e.g., "s1")
    Level     int        // Heading level (1-6)
    Title     string     // Heading text
    StartLine int        // Line number of heading
    EndLine   int        // Line before next same/higher level heading
    ParentID  string     // Parent section ID
    Children  []*Section // Nested sub-sections
}
```

## System Architecture

### High-Level Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                             │
│              (cmd/comments/main.go)                          │
│                                                              │
│  Commands: view, add, reply, suggest, accept, reject,       │
│            list, resolve, batch-add, batch-reply            │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ Calls
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                   Business Logic Layer                       │
│                                                              │
│  ┌──────────────────┐  ┌────────────────┐  ┌──────────────┐│
│  │ Comment System   │  │ Markdown Parser│  │ TUI Engine   ││
│  │ (pkg/comment/)   │  │ (pkg/markdown/)│  │ (pkg/tui/)   ││
│  │                  │  │                │  │              ││
│  │ • Storage        │  │ • Section tree │  │ • Bubbletea  ││
│  │ • Threads        │  │ • Path resolve │  │ • Viewports  ││
│  │ • Suggestions    │  │ • Line mapping │  │ • Modes      ││
│  │ • Positions      │  │                │  │              ││
│  └──────────────────┘  └────────────────┘  └──────────────┘│
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ File I/O
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                      File System                             │
│                                                              │
│  doc.md                    doc.md.comments.json             │
│  ┌────────────────┐        ┌─────────────────────────┐     │
│  │ # Introduction │        │ {                       │     │
│  │                │        │   "version": "2.0",     │     │
│  │ This is...     │        │   "threads": [...],     │     │
│  │                │        │   "hash": "sha256..."   │     │
│  │ ## Background  │        │ }                       │     │
│  └────────────────┘        └─────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow: Adding a Comment

```
1. User runs: ./comments add doc.md --line 10 --author "alice" --text "Review this"

2. CLI handler parses flags
   ↓
3. Load markdown content from doc.md
   ↓
4. Load existing comments from doc.md.comments.json (verify hash)
   ↓
5. Parse markdown to extract section structure
   ↓
6. Create new Comment object with unique ID
   ↓
7. Compute section metadata (SectionID, SectionPath) for line 10
   ↓
8. Append Comment to threads array
   ↓
9. Compute document hash (SHA-256)
   ↓
10. Save updated sidecar to doc.md.comments.json
    ↓
11. Output success message with comment ID
```

### Data Flow: Accepting a Suggestion

```
1. User runs: ./comments accept doc.md --suggestion s456

2. Load document and comments (verify hash)
   ↓
3. Find suggestion by ID
   ↓
4. Validate suggestion is pending (Accepted == nil)
   ↓
5. Apply multi-line replacement:
   - Split content into lines
   - Replace lines[StartLine:EndLine] with ProposedText
   - Join back to string
   ↓
6. Update markdown content in doc.md
   ↓
7. Mark suggestion.Accepted = true
   ↓
8. Recalculate positions for affected comments
   ↓
9. Compute new document hash
   ↓
10. Save updated sidecar with new hash
    ↓
11. Output success message
```

## Component Details

### Comment System (pkg/comment/)

**Responsibilities:**
- Load/save comments from JSON sidecar files
- Manage thread operations (add reply, resolve)
- Apply suggestions to markdown content
- Track comment positions
- Compute section metadata

**Key Files:**
- `types.go` - Core data structures
- `storage.go` - JSON serialization and hash validation
- `threads.go` - Thread helper functions
- `applier.go` - Suggestion application engine
- `positions.go` - Position recalculation after edits
- `sections.go` - Section metadata computation

**Helper Functions:**
```go
AddReplyToThread(threads []*Comment, threadID, author, text string) error
ResolveThread(threads []*Comment, threadID string) error
AcceptSuggestion(content string, suggestion *Comment) (string, error)
RejectSuggestion(threads []*Comment, suggestionID string) error
RecalculateCommentLines(comments []*Comment, editStartLine, editEndLine, newLineCount int)
```

### Markdown Parser (pkg/markdown/)

**Responsibilities:**
- Parse ATX-style markdown headings (# through ######)
- Build hierarchical section tree
- Map line numbers to section paths
- Resolve section paths to line ranges

**Key Files:**
- `types.go` - Section data structures
- `parser.go` - Document structure parser

**Key Functions:**
```go
ParseDocument(content string) (*DocumentStructure, error)
ResolveSectionPath(structure *DocumentStructure, path string) (*Section, error)
GetSectionAtLine(structure *DocumentStructure, line int) *Section
```

### TUI Engine (pkg/tui/)

**Responsibilities:**
- Interactive terminal UI with Bubbletea
- Split-pane layout (document + comments)
- Mode-based state machine
- Keyboard navigation and input handling

**Key Files:**
- `model.go` - Application state and update logic
- `modes.go` - View mode state machine
- `rendering.go` - Pure rendering functions
- `styles.go` - Lipgloss styling

**View Modes:**
- `FilePicker` - Select markdown file
- `Browse` - View document and comment list
- `LineSelect` - Select line for new comment
- `AddComment` - Input new comment text
- `ThreadView` - Expanded thread view
- `Reply` - Reply to thread
- `Resolve` - Confirm thread resolution
- `ReviewSuggestion` - Preview and accept/reject suggestion

## Key Design Decisions

### 1. Nested Thread Storage (v2.0)
**Decision:** Store threads as nested structures with `Replies` arrays
**Rationale:** Simpler data model, no need for dynamic thread building
**Trade-off:** Slightly more complex serialization, but better performance

### 2. Document Hashing (v2.0)
**Decision:** Store SHA-256 hash of markdown content
**Rationale:** Automatic staleness detection prevents data corruption
**Trade-off:** Extra computation on save, but worth it for safety

### 3. Line-Only Position Tracking (v2.0)
**Decision:** Track line numbers only, removed column and byte offset
**Rationale:** Simpler implementation, sufficient for line-based suggestions
**Trade-off:** Less precise positioning, but adequate for use case

### 4. Multi-Line Only Suggestions (v2.0)
**Decision:** Support only multi-line suggestions (removed other 3 types)
**Rationale:** Simpler implementation, covers 90% of use cases
**Trade-off:** Less granular edits, but clearer UX

### 5. JSON Sidecar Storage
**Decision:** Store comments separately from markdown
**Rationale:** Clean markdown, structured metadata, independent versioning
**Trade-off:** Two files to manage, but better separation of concerns

### 6. Section-Based Addressing
**Decision:** Support both line numbers and section paths
**Rationale:** Natural for hierarchical documents
**Trade-off:** Must recompute when headings change

### 7. @filename Syntax (v2.0)
**Decision:** Support reading text from files with @filename syntax
**Rationale:** Better for long comments/suggestions, esp. for LLM agents
**Trade-off:** Extra file I/O, but improves usability

## CLI Commands

### Core Commands

- `view [file]` - Interactive TUI mode
- `add <file>` - Add root comment (--line or --section)
- `reply <file>` - Reply to thread (--thread)
- `list <file>` - List comments with filters
- `resolve <file>` - Mark thread resolved (--thread)

### Suggestion Commands

- `suggest <file>` - Create multi-line suggestion
- `accept <file>` - Accept suggestion (--suggestion, optional --preview)
- `reject <file>` - Reject suggestion (--suggestion)

### Batch Commands (for LLM agents)

- `batch-add <file>` - Bulk add comments from JSON (--json)
- `batch-reply <file>` - Bulk reply to threads from JSON (--json)

## Testing Strategy

### Unit Tests

**Focus Areas:**
- Comment storage (load/save, hash validation, staleness detection)
- Thread operations (add reply, resolve, find by ID)
- Suggestion application (multi-line replacement)
- Position recalculation (shift lines after edits)
- Markdown parsing (section tree, path resolution)

**Coverage:**
- `pkg/comment/`: 61.5%
- `pkg/markdown/`: 92.0%

### Integration Tests

**Tested Workflows:**
- Add comment by line number
- Add comment by section path
- Create multi-line suggestion with @filename
- Accept suggestion and verify document update
- Batch operations (add, reply)
- List with filters (author, section, type)

## Performance Characteristics

### Current Performance

- **Document Loading:** O(n) where n = file size
- **Section Parsing:** O(n) where n = number of lines
- **Comment Storage:** O(m) where m = number of comments
- **Thread Lookup:** O(m) linear search (acceptable for typical usage)
- **Suggestion Application:** O(n) document rewrite

### Scalability Considerations

- Tested with documents up to 1000 lines
- Comment count typically < 100 per document
- Section parsing cached during single operation
- TUI viewport rendering optimized with Bubbletea

## Future Enhancements

### Potential Improvements

1. **Thread Search:** Indexed search across comment text
2. **Edit History:** Track comment edits over time
3. **Mentions:** @username notifications
4. **Tags:** #topic-based organization
5. **Export:** HTML output with inline comments
6. **Sync:** Multi-user collaboration support

### Architecture Evolution

- Consider B-tree indexing for large comment counts
- Add incremental markdown parsing
- Implement suggestion conflict detection
- Add undo/redo for comment operations

## Project Structure

```
comments/
├── cmd/
│   └── comments/          # Main CLI application
│       ├── main.go        # Command routing
│       ├── batch_add.go   # Batch add handler
│       └── batch_reply.go # Batch reply handler
├── pkg/
│   ├── comment/           # Comment system
│   │   ├── types.go       # Data structures
│   │   ├── storage.go     # JSON I/O + hashing
│   │   ├── threads.go     # Thread helpers
│   │   ├── applier.go     # Suggestion application
│   │   ├── positions.go   # Position tracking
│   │   └── sections.go    # Section computation
│   ├── markdown/          # Markdown parser
│   │   ├── types.go       # Section structures
│   │   └── parser.go      # ATX heading parser
│   └── tui/               # Terminal UI
│       ├── model.go       # Bubbletea model
│       ├── modes.go       # View modes
│       ├── rendering.go   # Pure render functions
│       └── styles.go      # Lipgloss styles
├── examples/              # Test documents
├── docs/                  # Architecture docs
├── CLAUDE.md              # Development guide
├── README.md              # User documentation
├── USAGE.md               # Usage guide
└── go.mod                 # Dependencies
```

## Dependencies

**Core:**
- Go 1.21+
- Standard library (encoding/json, crypto/sha256, os, etc.)

**TUI:**
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - UI components
- `github.com/charmbracelet/lipgloss` - Terminal styling

**Future:**
- LLM integration (Anthropic SDK or similar)

## References

- **Development Guide:** `/CLAUDE.md` - Detailed implementation notes
- **User Documentation:** `/README.md` - Installation and quick start
- **Usage Guide:** `/USAGE.md` - Complete command reference
- **Test Suite:** `pkg/*/test.go` - Unit and integration tests
