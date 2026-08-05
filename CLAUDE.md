# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`comments` is a CLI tool for collaborative document writing with inline comments and suggestions, designed for seamless LLM integration. It brings Google Doc-style commenting and track-changes functionality to terminal-based markdown editing, enabling better collaboration between humans and AI agents.

**Key Philosophy**: Instead of having LLMs rewrite entire documents, add contextual comments at specific lines to guide iteration and discussion. Suggestions allow proposing edits with preview and accept/reject workflow.

**Storage Model (v2.0)**: Comments and suggestions are stored in sidecar JSON files (`.md.comments.json`) separate from the markdown content, with nested thread structure and automatic staleness detection.

## Build and Development Commands

### Building
```bash
# Build main binary
go build -o comments ./cmd/comments
```

**Important**: After any code change, rebuild the root binary (`go build -o comments ./cmd/comments`) before testing CLI behavior — stale binaries have masqueraded as missing features.

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/comment/...
go test ./pkg/tui/...
go test ./pkg/markdown/...

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Running
```bash
# Interactive TUI
./comments view examples/sample.md

# Non-interactive commands
./comments list examples/sample.md
./comments add examples/sample.md --line 10 --author "user" --text "Test comment" --type Q
./comments reply examples/sample.md --thread c123 --author "user" --text "Reply"
./comments resolve examples/sample.md --thread c123

# Suggestion commands (multi-line only in v2.0)
./comments suggest examples/sample.md --start-line 15 --end-line 17 --author "claude" \
  --text "Improve wording" --original "old text" --proposed "new text"
./comments accept examples/sample.md --suggestion c123 --preview
./comments accept examples/sample.md --suggestion c123
./comments reject examples/sample.md --suggestion c456

# List with filters
./comments list examples/sample.md --author alice --format json
./comments list examples/sample.md --search "TODO" --sort timestamp
./comments list examples/sample.md --lines 10-30 --resolved false

# Batch operations for LLM agents (supports suggestions)
./comments batch-add examples/sample.md --json comments.json
./comments batch-reply examples/sample.md --json replies.json
echo '[{"thread":"c123","author":"claude","text":"LGTM"}]' | ./comments batch-reply examples/sample.md --json -

# Section-based operations
# Add comments by section instead of line number
./comments add examples/sample.md --section "Introduction" --author "alice" --text "Great intro"
./comments add examples/sample.md --section "Implementation > Architecture" --author "bob" --text "Add diagram"

# Filter comments by section (includes nested sub-sections)
./comments list examples/sample.md --section "Introduction"
./comments list examples/sample.md --section "Implementation > Data Model"

# File input support (v2.0 - use @filename for multi-line text)
./comments add doc.md --line 10 --author "claude" --text @comment.txt
./comments suggest doc.md --start-line 5 --end-line 8 --author "claude" \
  --text "Major refactor" --original @original.txt --proposed @proposed.txt

# Review gate (for agent loops and CI)
# Exit codes: 0 = approved, 10 = changes requested
./comments add doc.md --line 10 --author "eric" --text "Fix this first" --blocking
./comments gate doc.md                  # human-readable
./comments gate doc.md --json           # machine-readable decision with context
./comments gate specs/001-feature/      # aggregate over a directory
./comments gate doc.md --strict         # any unresolved comment fails, not just blocking
./comments signoff doc.md               # record a completed review pass (decision derived from gate)
./comments signoff doc.md --decision changes_requested --note "see blocking comments"

# Watch review-state changes (NDJSON event stream, poll-based)
./comments watch specs/                       # runs until killed; exits cleanly if stdout closes (EPIPE)
./comments watch doc.md --until signoff       # exit 0 after emitting a signoff event
./comments watch doc.md --until signoff,gate_changed --interval 500ms

# Doc templates (guardrails for agent-written docs)
./comments template list                       # built-ins: design-doc, adr, rfc (+ project templates in .comments/templates/)
./comments template show design-doc
./comments validate draft.md --template design-doc          # structural check, exit 1 on violations
./comments validate draft.md --template design-doc --json
./comments seed draft.md --template design-doc              # materialize review criteria + NEEDS CLARIFICATION markers as threads
./comments gate draft.md                                    # gate now also validates against the recorded template

# MCP Server (Model Context Protocol for LLM integration)
./comments serve-mcp
```

### Doc Templates (Review Guardrails)

Templates constrain what an agent writes so humans can review it well. A template (YAML, built-in or in `.comments/templates/`) declares:

- **Sections**: required headings (matched by title or path suffix), order, `max_words` caps (attacks LLM padding), `min_subsections` (e.g. "Options Considered" needs >= 2 alternatives).
- **`zone: human`**: human-decision sections. Threads anchored there cannot be resolved by agents over MCP — the agent gets an error telling it to reply instead; only the human resolves (CLI/TUI).
- **`review_criteria`**: per-section self-review prompts for the *agent* — the skill requires the agent to judge its draft against each criterion and post doc-specific callouts (weakest reasoning, assumptions, invented facts) at exact lines, instead of forwarding generic questions. `comments seed` without flags still materializes criteria as generic blocking threads (useful for human-only workflows); agent flows use `seed --markers-only`.
- **Markers**: every `[NEEDS CLARIFICATION: ...]` occurrence is a validation violation and seeds a blocking Q thread at that line — agents must flag ambiguity instead of guessing (Spec Kit convention).

Workflow: agent reads the template (`comments_get_template`) as its writing brief → drafts → `comments_validate` and self-corrects structure → `comments_seed` → human review = resolving seeded/own threads → `comments gate` (validates structure + comment state; `seed` records the template in the sidecar so the gate picks it up automatically) → `signoff`.

### Review Gate and Signoff

The gate turns review state into a machine-readable contract for agent loops and SDD phase boundaries:

- **Blocking comments**: `--blocking` on `add` (or `"blocking": true` in batch/MCP) marks a thread as gate-failing until resolved. Non-blocking comments are reported but don't fail the gate.
- **`comments gate <file-or-dir>`**: exit 0 = approved, exit 10 = changes requested (revdiff/Plannotator convention). `--json` emits `{"decision", "files", "summary"}` with blocking/non-blocking/pending-suggestion lists and document context. `--strict` fails on any unresolved thread or pending suggestion.
- **`comments signoff <file>`**: records a human review pass (`reviews` array in the sidecar) with a decision derived from the gate (or overridden via `--decision`). This is the signal `comments_request_review` waits for.
- **Agent loop**: agent drafts → calls `comments_request_review` (MCP, blocks) → human reviews (`comments view`, `add --blocking`, etc.) → human runs `comments signoff` → agent receives decision + remaining comments → addresses them one at a time (see `skills/review-comments/SKILL.md`) → repeat until gate passes.

### Model Context Protocol (MCP) Integration

**Overview:** The `comments` tool provides a Model Context Protocol (MCP) server for seamless LLM integration. This enables LLMs like Claude to access documents, manage comments, and collaborate on document editing through a standardized protocol.

**Starting the MCP Server:**

```bash
./comments serve-mcp
```

The server runs over stdio and exposes:
- **2 Resources** (subscribable) for accessing documents and threads
- **19 Tools** for all comment operations

**MCP Resources:**

Resources expose document data that LLMs can read and subscribe to:

```
comments://doc/{filepath}
  - Returns: Full document with all comment threads
  - Format: JSON with Content, Threads, DocumentHash, LastValidated
  - Example: comments://doc/examples/sample.md

comments://thread/{filepath}/{thread_id}
  - Returns: Specific comment thread with all nested replies
  - Format: JSON Comment object
  - Example: comments://thread/examples/sample.md/c123
```

**MCP Tools:**

All CLI operations are available as MCP tools with automatic JSON schema validation:

*Read Operations:*
- **comments_list** - List/filter comments (all filtering options: author, type, section, search, status, priority, line range)
- **comments_get** - Get specific comment with context
- **comments_status** - Get document status (pending suggestions, orphaned comments, staleness)

*Write Operations:*
- **comments_add** - Add root comment (by line or section)
- **comments_reply** - Reply to thread
- **comments_resolve** - Mark thread resolved/unresolved

*Suggestion Operations:*
- **comments_suggest** - Create multi-line edit suggestion
- **comments_accept** - Accept suggestion (with preview option, applies changes to document)
- **comments_reject** - Reject suggestion

*Batch Operations:*
- **comments_batch_add** - Bulk comment addition
- **comments_batch_reply** - Bulk reply addition

*Review Gate Operations:*
- **comments_gate** - Evaluate review gate for a file or directory (approved / changes_requested)
- **comments_request_review** - Request human review. Default: block until the human records a signoff (`comments signoff`), then return the decision and remaining comments. With `blocking: false`: return immediately with `{status: "requested", since: <RFC3339>}` — a durable handle for later polling that survives agent restarts
- **comments_check_review** - Poll a review handle: `{filepath, since}` returns `{status: "pending"}` until a signoff newer than `since` exists, then `{status: "review_completed", review, gate_decision, files}`

*Agent Inbox:*
- **comments_inbox** - One-call attention view for a file or directory: unresolved threads with replies newer than `since` (or any replies when `since` is empty) plus all unresolved blocking threads, each with its last reply (author, text, timestamp)

*Template Operations:*
- **comments_get_template** - Read a template as a writing brief (or list templates)
- **comments_validate** - Check document structure against a template before requesting review
- **comments_seed** - Materialize template review criteria and ambiguity markers as anchored threads

*Anchor Migration:*
- **comments_reanchor** - After editing a commented document, migrate the anchors your edits displaced (batch comment_id → new line/section); required post-edit step for agents

### Content Anchoring (v2.1)

Comments carry a content `Anchor` (target line text + one line of context each side), captured automatically at creation. On document change, each comment re-anchors via a cascade: exact position → exact text search → normalized (whitespace/case-insensitive) search, labeled `anchor_confidence: fuzzy` → section-path fallback (`section-level`) → orphan. `add --line N` stores N exactly — comments no longer snap to section headings. Old sidecars get anchors backfilled lazily on load when the document hash matches. Comment IDs are short random base36 (`c7f3k`); existing long IDs stay valid, and `SaveToSidecar` guarantees uniqueness.

**Example MCP Tool Usage:**

```json
// comments_list with filters
{
  "filepath": "examples/sample.md",
  "author": "alice",
  "type": "Q",
  "resolved": false,
  "with_context": true
}

// comments_add
{
  "filepath": "examples/sample.md",
  "author": "claude",
  "text": "Consider adding error handling here",
  "line": 42,
  "type": "S",
  "priority": "high"
}

// comments_suggest
{
  "filepath": "examples/sample.md",
  "author": "claude",
  "text": "Improve error message clarity",
  "start_line": 15,
  "end_line": 17,
  "original_text": "Error occurred",
  "proposed_text": "Failed to process request: connection timeout"
}

// comments_batch_add
{
  "filepath": "examples/sample.md",
  "comments": [
    {"author": "claude", "text": "Great point!", "line": 10, "type": "Q"},
    {"author": "claude", "text": "Add tests", "line": 25, "type": "T"}
  ]
}
```

**Integration with LLM Clients:**

To use the MCP server with Claude or other LLM clients:

1. Configure your MCP client to launch: `./comments serve-mcp`
2. The client will communicate via stdio
3. LLMs can now:
   - Read documents with `comments://doc/{path}` resources
   - Add comments and suggestions via tools
   - Manage threads (reply, resolve)
   - Accept/reject suggestions with automatic document updates

**Benefits:**
- **Structured Collaboration**: LLMs can add inline feedback instead of rewriting documents
- **Track Changes**: Suggestions provide preview-before-apply workflow
- **Context-Rich**: Section paths and surrounding lines provide better context
- **Batch Efficient**: Bulk operations reduce round-trips
- **Real-time**: Resource subscriptions notify of changes

### Section-Based Operations

**Overview:** Comments can be added and filtered by markdown section (heading) instead of just line numbers. This is useful for organizing comments by document structure rather than physical line positions.

**Key Concepts:**
- **Section Path**: Hierarchical path using " > " separator (e.g., "Introduction > Background > History")
- **Section vs. Line**: Use `--section` OR `--line` (mutually exclusive)
- **Nested Sections**: Section filters include all descendant sections (tree behavior)
- **Storage**: Section metadata (SectionID and SectionPath) stored in comments (v2.0 format)

**Usage Examples:**

```bash
# Add comment to a section (attaches to heading line)
./comments add doc.md --section "Introduction" --author "alice" --text "Needs expansion"

# Add comment to nested section (full path required for disambiguation)
./comments add doc.md --section "Intro > Overview > Key Points" --author "bob" --text "Add examples"

# List all comments in a section (includes nested sub-sections)
./comments list doc.md --section "Implementation"
# Shows comments in "Implementation", "Implementation > Architecture", "Implementation > Data Model", etc.

# Combine section filter with other filters
./comments list doc.md --section "Introduction" --author "alice" --type "Q"

# Batch add with sections
cat > comments.json << 'EOF'
[
  {"section": "Introduction", "author": "alice", "text": "Great start", "type": "Q"},
  {"section": "Implementation > Architecture", "author": "bob", "text": "Add diagram", "type": "S"},
  {"line": 50, "author": "charlie", "text": "Typo here"}
]
EOF
./comments batch-add doc.md --json comments.json
```

**Display Format:**
All comment listings now show section paths when available:
```
[1] Introduction > Background > History (Line 9) • @alice • 2025-01-15 10:30
    Type: Root | Thread ID: c123 | Comment ID: c123
    [Q] Consider adding more historical context
```

**Error Handling:**
If a section path is invalid, you'll see available sections:
```
Error: section 'Overvieww' not found
Available sections:
  - Introduction
  - Introduction > Background
  - Implementation
  - Implementation > Architecture
  - Conclusion
```

## Architecture

### High-Level Data Flow

The system follows a clear separation of concerns across three main layers:

1. **Comment System Layer** (`pkg/comment/`)
   - **Storage**: JSON sidecar file I/O with v2.0 format (nested threads, document hashing)
   - **Validation**: SHA-256 document hashing and staleness detection
   - **Applier**: Multi-line suggestion application engine
   - **Positions**: Line-only position tracking and conflict detection
   - **Helpers**: Thread manipulation functions (AddReplyToThread, ResolveThread, etc.)
   - **Types**: Core data structures (Comment with nested Replies, DocumentWithComments)

2. **TUI Layer** (`pkg/tui/`)
   - **Model**: Bubbletea application state, implements Update/View/Init pattern
   - **Modes**: State machine for different UI modes
   - **Rendering**: Pure functions that transform model state into terminal output
   - **Styles**: Centralized lipgloss styling definitions

### Critical Architecture Patterns

#### Storage Format and Data Model (v2.0)

Comments and suggestions are stored in JSON sidecar files (`{filename}.comments.json`) with the following structure:

```json
{
  "version": "2.0",
  "documentHash": "sha256_hash_of_markdown_content",
  "lastValidated": "2025-01-15T10:30:00Z",
  "threads": [{
    "ID": "c123",
    "Author": "alice",
    "Timestamp": "2025-01-15T10:30:00Z",
    "Text": "Comment text",
    "Type": "Q",
    "Line": 10,
    "SectionID": "s2",
    "SectionPath": "Introduction > Overview",
    "Resolved": false,
    "Replies": [{
      "ID": "c124",
      "Author": "bob",
      "Timestamp": "2025-01-15T11:00:00Z",
      "Text": "Reply to comment",
      "Line": 10,
      "Resolved": false,
      "Replies": []
    }],
    "IsSuggestion": false,
    "StartLine": 0,
    "EndLine": 0,
    "OriginalText": "",
    "ProposedText": "",
    "Accepted": null
  }]
}
```

**Key v2.0 Changes**:
- **Nested Thread Structure**: `Replies` array directly in Comment (no more flat array with ThreadID/ParentID)
- **Document Hashing**: SHA-256 hash for staleness detection
- **Validation Timestamp**: `lastValidated` tracks when sidecar was last checked
- **Simplified Position Tracking**: Only `Line` field (removed Column, ByteOffset)
- **Simplified Suggestions**: Only multi-line type (removed line, char-range, diff-hunk types)
- **Boolean Acceptance**: `Accepted *bool` (nil=pending, true=accepted, false=rejected)

**Comment Types**: Comments can be categorized with type prefixes:
- `Q` - Question: Requests clarification or poses a question
- `S` - Suggestion: Proposes a change or improvement (note: edit suggestions use IsSuggestion field)
- `B` - Bug: Identifies an issue or bug
- `T` - TODO: Marks something to be done
- `E` - Enhancement: Suggests a feature enhancement

**Threading Model (v2.0)**:
- Root comments: Stored directly in `threads` array
- Replies: Stored in `Replies` array of parent comment (recursively nested)
- Position tracking: `Line` field in each Comment (no separate Positions map)

**Suggestion Model (v2.0)**: Simplified to multi-line only
- `IsSuggestion`: true if this is an edit suggestion
- `StartLine` / `EndLine`: Line range for the edit
- `OriginalText`: Text being replaced (optional, used for verification)
- `ProposedText`: Proposed replacement text
- `Accepted`: nil (pending), true (accepted), false (rejected)

**Staleness Detection**:
- On load, tool computes SHA-256 hash of current markdown content
- If hash doesn't match stored `documentHash`, sidecar is considered stale
- Stale sidecars are automatically archived to `.comments.json.backup.TIMESTAMP`
- User gets warning with option to start fresh or restore from backup

#### Bubbletea State Machine

The TUI uses a mode-based state machine. Key insight: **Mode transitions happen in key handlers, not in Update()**:

```go
// Good: State transition in key handler
case "c":
    m.mode = ModeLineSelect
    m.selectedLine = 1
    m.documentViewport.SetContent(m.renderDocumentWithCursor())
    return m, nil

// Bad: Don't put transitions in updateByMode
```

**Critical**: When adding new modes, update THREE places:
1. `modes.go`: Add to ViewMode enum
2. `model.go`: Add case in `handleKeyPress()`
3. `model.go`: Add case in `updateByMode()` for viewport updates

#### Viewport Management

Viewports must be initialized when `ready = false`:
```go
if !m.ready {
    m.documentViewport = viewport.New(docWidth, m.height-3)
    // Set initial content
    m.documentViewport.SetContent(m.renderDocument())
    m.ready = true
}
```

**Common bug**: Forgetting to call `handleResize()` after loading a file from the file picker. Always check if dimensions exist before initializing viewports.

**Recent enhancements**:
- Markdown syntax highlighting for headers (# H1, ## H2, etc.) using lipgloss styling
- Auto-scroll to center selected comments when navigating in comment viewport
- Full-width textarea for comment input (better UX for longer comments)

#### Rendering Pattern

All rendering functions are **pure functions** on the Model:
```go
func (m *Model) renderDocument() string
func (m *Model) renderComments() string
func (m *Model) renderThread() string
```

They:
- Take Model state as input
- Return strings for viewport content
- Have NO side effects
- Don't modify Model state

### File Organization

```
pkg/
├── comment/           # Core comment system (pure logic, no UI)
│   ├── types.go      # Data structures (Comment with Replies, DocumentWithComments)
│   ├── storage.go    # JSON sidecar I/O (v2.0 format)
│   ├── validation.go # Document hashing and staleness detection
│   ├── helpers.go    # Thread manipulation (AddReplyToThread, ResolveThread, etc.)
│   ├── applier.go    # Multi-line suggestion application
│   ├── positions.go  # Line-only position tracking and conflict detection
│   ├── sections.go   # Section-based addressing
│   └── *_test.go     # Unit tests
├── tui/              # Terminal UI (Bubbletea components)
│   ├── model.go      # Application state and update logic
│   ├── modes.go      # View mode state machine
│   ├── rendering.go  # Pure rendering functions
│   └── styles.go     # Lipgloss styling
└── markdown/         # Markdown parsing
    └── parser.go     # ATX heading parser for section addressing

cmd/
├── comments/         # Main CLI application
│   ├── main.go       # Command routing and handlers
│   ├── batch_add.go  # Batch comment addition
│   ├── batch_reply.go # Batch reply addition
│   └── list_filters.go # Filtering and sorting logic
```

## Key Abstractions

### Comment System (v2.0)

**DocumentWithComments**: The central data structure
```go
type DocumentWithComments struct {
    Content       string     // Raw markdown without comments
    Threads       []*Comment // Root comment threads (each may contain nested replies)
    DocumentHash  string     // SHA-256 hash for staleness detection
    LastValidated time.Time  // Last validation timestamp
}
```

**Comment**: The core comment structure (v2.0 simplified)
```go
type Comment struct {
    // Identity
    ID        string    // Unique identifier
    Author    string    // Author of the comment
    Timestamp time.Time // When the comment was created

    // Content
    Text string // Comment content
    Type string // Optional: Q, S, B, T, E

    // Position
    Line int // Line number where comment is attached

    // Section metadata (computed from document structure)
    SectionID   string // ID of the section this comment belongs to
    SectionPath string // Full hierarchical path (e.g., "Intro > Overview")

    // State
    Resolved bool // Whether the comment/thread has been resolved

    // Thread structure (nested replies)
    Replies []*Comment // Nested replies to this comment

    // Suggestion fields (for edit suggestions)
    IsSuggestion bool   // True if this is an edit suggestion
    StartLine    int    // Start line for suggestion (0 if not a suggestion)
    EndLine      int    // End line for suggestion (0 if not a suggestion)
    OriginalText string // Original text being replaced
    ProposedText string // Proposed replacement text
    Accepted     *bool  // nil=pending, true=accepted, false=rejected
}
```

**Key v2.0 Simplifications**:
- Removed `ThreadID`, `ParentID` (use nested `Replies` array instead)
- Removed `Column`, `ByteOffset` (line-only tracking)
- Removed `Selection` struct (use `StartLine`/`EndLine` directly)
- Removed `SuggestionType` enum (multi-line only)
- Removed `AcceptanceState` enum (use `*bool` pointer)

**Helper Functions** (v2.0 - replace dynamic thread building):
```go
// Thread manipulation
func AddReplyToThread(threads []*Comment, threadID, author, text string) error
func ResolveThread(threads []*Comment, threadID string) error
func UnresolveThread(threads []*Comment, threadID string) error

// Suggestion operations
func NewSuggestion(author string, startLine, endLine int, text, originalText, proposedText string) *Comment
func AcceptSuggestion(threads []*Comment, suggestionID string) error
func RejectSuggestion(threads []*Comment, suggestionID string) error
func GetPendingSuggestions(threads []*Comment) []*Comment
func GetSuggestionsByAuthor(threads []*Comment, author string) []*Comment

// Utility
func GetVisibleComments(threads []*Comment, showResolved bool) []*Comment
func GroupCommentsByLine(threads []*Comment) map[int][]*Comment
```

### TUI Model

**The Elm Architecture**: Bubbletea follows this pattern:
- **Model**: All application state in one struct
- **Update**: Model + Message → (Model, Command)
- **View**: Model → String

**State Synchronization**: The model maintains multiple viewports that need to stay in sync:
- `documentViewport`: Shows markdown content (60% width)
- `commentViewport`: Shows comment list (40% width)
- `threadViewport`: Shows expanded thread (full width)
- `commentInput`: Textarea for new comments/replies

When mode changes, update the **active** viewport's content, not all viewports.

### Recommended Review Flow (the tool's core loop)

1. **Agent produces doc** under a template: read the brief (`comments_get_template`), draft, `comments validate` until structure is clean.
2. **Seed** review threads: `comments seed` (template criteria + NEEDS CLARIFICATION markers become blocking threads; template recorded in sidecar).
3. **Human reviews** in the TUI: `comments view <doc>` — walk threads, reply/resolve, add comments (`--blocking` for must-fix), then `comments signoff`.
4. **Agent processes feedback** one comment at a time (see `skills/review-comments/SKILL.md`): reply/resolve/suggest, `comments_reanchor` after edits, re-request review.
5. **Iterate until the gate unblocks**: `comments gate <doc>` exit 0 → implement.

## Testing Strategy

### Unit Tests

Focus on:
1. **Storage round-trips**: v2.0 format save/load with nested threads
2. **Validation**: Document hashing and staleness detection
3. **Helper functions**: AddReplyToThread, ResolveThread, AcceptSuggestion, etc.
4. **Suggestion application**: Multi-line replacements, edge cases
5. **Position tracking**: Line recalculation after edits
6. **Conflict detection**: Overlapping suggestions

Example test structure:
```go
func TestSaveAndLoadRoundTrip(t *testing.T) {
    content := "# Test Document\n\nSome content here.\n"
    doc := &DocumentWithComments{
        Content: content,
        Threads: []*Comment{
            {
                ID:     "c1",
                Author: "alice",
                Line:   3,
                Text:   "This is a question",
                Replies: []*Comment{
                    {
                        ID:     "c2",
                        Author: "bob",
                        Line:   3,
                        Text:   "Here's an answer",
                        Replies: []*Comment{},
                    },
                },
            },
        },
    }

    // Save and load
    SaveToSidecar(mdPath, doc)
    loaded, err := LoadFromSidecar(mdPath)

    // Verify nested structure preserved
    if len(loaded.Threads[0].Replies) != 1 {
        t.Errorf("Expected 1 reply")
    }
}
```

### Manual Testing

Run tests:
```bash
go test ./... -v
go test ./pkg/comment/... -cover
```

Or use the TUI directly on sample documents in `examples/`.

## Common Workflows

### Adding a New CLI Command

1. Add case to switch in `cmd/comments/main.go`
2. Implement handler function (follow pattern of `addCommand`, `replyCommand`)
3. Update `printUsage()` with new command
4. Test with example document

### Adding a New View Mode

1. Add to `ViewMode` enum in `pkg/tui/modes.go`
2. Add `String()` case
3. Create `handle<Mode>Keys()` in `pkg/tui/model.go`
4. Add case to `handleKeyPress()` switch
5. Add case to `updateByMode()` for viewport updates
6. Create `view<Mode>()` rendering function in `pkg/tui/model.go`
7. Add case to `View()` switch

### Using Batch Operations (for LLM Agents)

Batch operations are optimized for LLM agents like Claude Code to perform multiple operations efficiently:

**Batch-add** (add multiple root comments):
```bash
# Create JSON file with comments (type is optional but recommended)
cat > comments.json << 'EOF'
[
  {"line": 10, "author": "claude", "text": "Consider edge cases", "type": "Q"},
  {"line": 25, "author": "claude", "text": "Add unit tests", "type": "S"},
  {"line": 40, "author": "alice", "text": "Check memory usage"}
]
EOF

./comments batch-add document.md --json comments.json

# Or use stdin for single-command workflow
echo '[{"line":10,"author":"claude","text":"Good idea","type":"Q"}]' | \
  ./comments batch-add document.md --json -
```

**Important**: All fields are required in batch-add:
- `line`: Line number (integer, must be > 0)
- `author`: Author name (string, required)
- `text`: Comment text (string, required)
- `type`: Comment type (string, optional: Q, S, B, T, E)

**Batch-reply** (reply to multiple threads):
```bash
# First, list threads to get IDs
./comments list document.md

# Create JSON with replies
cat > replies.json << 'EOF'
[
  {"thread": "c123", "author": "claude", "text": "Good point about scalability"},
  {"thread": "c456", "author": "claude", "text": "I agree with this approach"}
]
EOF

./comments batch-reply document.md --json replies.json

# Or use stdin for single-command workflow
echo '[{"thread":"c123","author":"claude","text":"LGTM"}]' | \
  ./comments batch-reply document.md --json -
```

**Important**: All fields are required in batch-reply:
- `thread`: Thread ID (string, must exist)
- `author`: Author name (string, required)
- `text`: Reply text (string, required)

The tool validates all thread IDs exist before adding any replies and will show available threads if validation fails.

**Benefits:**
- Single file write operation (more efficient than multiple commands)
- Atomic operation (all succeed or all fail)
- Better for scripting and automation
- Reduces I/O overhead

### File Input Support (@filename syntax)

v2.0 adds support for reading text content from files using `@filename` syntax:

```bash
# Read comment text from file
./comments add doc.md --line 10 --author "claude" --text @comment.txt

# Read multi-line suggestion content from files
./comments suggest doc.md --start-line 5 --end-line 8 --author "claude" \
  --text "Refactor section" \
  --original @original-text.txt \
  --proposed @new-text.txt
```

This is especially useful for:
- Multi-line comments that would be awkward to escape on command line
- Large suggestion blocks
- Programmatically generated content

### Filtering and Querying Comments

The `list` command supports powerful filtering and output options:

**Filtering Options:**
```bash
# Filter by author
./comments list doc.md --author alice

# Text search (case-insensitive)
./comments list doc.md --search "TODO"

# Line range
./comments list doc.md --lines 10-30

# Resolved/unresolved only
./comments list doc.md --resolved true
./comments list doc.md --resolved false
```

**Sorting:**
```bash
# Sort by line number (default)
./comments list doc.md --sort line

# Sort by timestamp
./comments list doc.md --sort timestamp

# Sort by author
./comments list doc.md --sort author
```

**Output Formats:**
```bash
# Table format (default) - shows root comments only
./comments list doc.md --format table

# JSON format - includes all comments with full metadata
./comments list doc.md --format json
```

**Combining Filters:**
All filters can be combined:
```bash
./comments list doc.md --author claude --lines 10-50 --search "performance" --format json
```

## Debugging Tips

### TUI Issues

If the TUI hangs or doesn't respond:
- Check that the mode is included in `updateByMode()` switch
- Verify viewport initialization happens in `handleResize()` when `ready = false`
- Look for blocking operations in Update() - all I/O should be async

### Storage Issues

If sidecars aren't loading:
- Check for validation warnings - sidecar may be stale
- Look in same directory for `.backup.TIMESTAMP` files (archived stale sidecars)
- Verify JSON format is v2.0 (has `threads` array, not `comments`)
- Check `documentHash` matches current content

### Thread Structure Issues

If threads show wrong structure in v2.0:
- Verify `Replies` array is populated correctly
- Check that helpers like `AddReplyToThread()` are used (don't manipulate Replies directly)
- Use `doc.GetAllComments()` to flatten threads for searching

## Environment Variables

- `USER`: Used as default author name for comments (falls back to "user")

## Dependencies

Key external dependencies:
- **Bubbletea** (`charmbracelet/bubbletea`): TUI framework
- **Bubbles** (`charmbracelet/bubbles`): UI components (filepicker, viewport, textarea)
- **Lip Gloss** (`charmbracelet/lipgloss`): Terminal styling
- **Anthropic SDK** (via API): Claude integration

All dependencies are vendored via `go.mod`.
