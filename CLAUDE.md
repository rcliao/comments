# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`comments` is a CLI tool for collaborative document writing with inline comments, designed for seamless LLM integration. It brings Google Doc-style commenting to terminal-based markdown editing, enabling better collaboration between humans and AI agents.

**Key Philosophy**: Instead of having LLMs rewrite entire documents, add contextual comments at specific lines to guide iteration and discussion.

## Build and Development Commands

### Building
```bash
# Build main binary
go build -o comments ./cmd/comments

# Build test parser utility
go build -o test-parser ./cmd/test-parser
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./pkg/comment/...
go test ./pkg/tui/...

# Run single test function
go test -v -run TestParserBasic ./pkg/comment/
go test -v -run TestThreadBuilding ./pkg/comment/

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

# List with filters
./comments list examples/sample.md --author alice --format json
./comments list examples/sample.md --search "TODO" --sort timestamp
./comments list examples/sample.md --lines 10-30 --resolved false

# Batch operations for LLM agents
./comments batch-add examples/sample.md --json comments.json
./comments batch-reply examples/sample.md --json replies.json
echo '[{"thread":"c123","author":"claude","text":"LGTM"}]' | ./comments batch-reply examples/sample.md --json -

# LLM integration (requires ANTHROPIC_API_KEY)
export ANTHROPIC_API_KEY=your_key
./comments ask examples/sample.md --prompt "Explain this" --line 5
```

## Architecture

### High-Level Data Flow

The system follows a clear separation of concerns across three main layers:

1. **Comment System Layer** (`pkg/comment/`)
   - **Parser**: Transforms markdown with CriticMarkup syntax into structured Comment objects
   - **Serializer**: Converts Comment objects back to CriticMarkup format for file persistence
   - **Thread Manager**: Organizes flat comment arrays into hierarchical thread structures
   - **Types**: Core data structures (Comment, Thread, DocumentWithComments, Position)

2. **TUI Layer** (`pkg/tui/`)
   - **Model**: Bubbletea application state, implements Update/View/Init pattern
   - **Modes**: State machine for different UI modes (FilePicker, Browse, LineSelect, AddComment, ThreadView, Reply, Resolve)
   - **Rendering**: Pure functions that transform model state into terminal output
   - **Styles**: Centralized lipgloss styling definitions

3. **LLM Integration Layer** (`pkg/llm/`)
   - **Provider Interface**: Abstract LLM provider contract
   - **Claude Provider**: Anthropic API implementation with context building and streaming

### Critical Architecture Patterns

#### Comment Format and Threading

Comments use extended CriticMarkup with this structure:
```
{>>[@author:id:threadid:line:timestamp] comment text <<}
```

**Comment Types**: Comments can be categorized with type prefixes:
- `Q` - Question: Requests clarification or poses a question
- `S` - Suggestion: Proposes a change or improvement
- `B` - Bug: Identifies an issue or bug
- `T` - TODO: Marks something to be done
- `E` - Enhancement: Suggests a feature enhancement

Types are stored in the `Type` field and auto-prefixed to text as `[Q]`, `[S]`, etc. during batch operations.

**Threading Model**:
- Root comments: `ThreadID == ID`, `ParentID == ""`
- Replies: `ThreadID` points to root, `ParentID` points to immediate parent
- Position tracking: Line numbers stored in separate `Positions` map (keyed by comment ID)

**Backward Compatibility**: The parser supports both:
- Old format (4 fields): `author:id:line:timestamp`
- New format (5 fields): `author:id:threadid:line:timestamp`

The distinction is made by counting colons in the metadata string (accounting for colons in ISO8601 timestamps).

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
│   ├── types.go      # Data structures
│   ├── parser.go     # CriticMarkup → Comment objects
│   ├── serializer.go # Comment objects → CriticMarkup
│   ├── threads.go    # Thread building and operations
│   └── *_test.go     # Unit tests
├── tui/              # Terminal UI (Bubbletea components)
│   ├── model.go      # Application state and update logic
│   ├── modes.go      # View mode state machine
│   ├── rendering.go  # Pure rendering functions
│   └── styles.go     # Lipgloss styling
└── llm/              # LLM provider integration
    ├── types.go      # Provider interface
    └── claude.go     # Anthropic implementation

cmd/
├── comments/         # Main CLI application
│   └── main.go       # Command routing and handlers
└── test-parser/      # Parser testing utility
    └── main.go
```

## Key Abstractions

### Comment System

**DocumentWithComments**: The central data structure
```go
type DocumentWithComments struct {
    Content   string              // Raw markdown without comments
    Comments  []*Comment          // All comments (flat array)
    Positions map[string]Position // Comment ID → Position mapping
}
```

**Why separate Positions?** Allows efficient position updates during document edits without scanning all comment text.

**Comment**: The core comment structure
```go
type Comment struct {
    ID        string    // Unique identifier
    ThreadID  string    // Root comment ID (same as ID for root)
    ParentID  string    // Parent comment ID (empty for root)
    Author    string    // Author name
    Line      int       // Original line number
    Timestamp time.Time // Creation time
    Text      string    // Comment content
    Type      string    // Optional: Q, S, B, T, E
    Resolved  bool      // Resolution status
}
```

**Type field**: Used for categorizing comments. When set, batch operations auto-prefix text with `[Q]`, `[S]`, etc.

**Thread**: Built on-demand from flat Comment array
```go
type Thread struct {
    ID          string
    RootComment *Comment
    Replies     []*Comment
    Resolved    bool
    Line        int
}
```

Threads are **ephemeral** - rebuilt via `BuildThreads(comments)` whenever needed.

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

### LLM Integration

**Context Building**: The LLM provider receives:
```go
type CompletionRequest struct {
    DocumentContent  string
    Comments         []*Comment
    Prompt           string
    ContextStartLine int
    ContextEndLine   int
    Temperature      float64
    MaxTokens        int
}
```

The provider is responsible for building the actual LLM prompt from this context.

## Testing Strategy

### Unit Tests

Focus on:
1. **Parser correctness**: All CriticMarkup variants, malformed input, edge cases
2. **Thread building**: Parent-child relationships, reply ordering
3. **Serialization round-trips**: Parse → Serialize → Parse should be identity

Example test structure:
```go
func TestParserBasic(t *testing.T) {
    content := `# Title
{>>[@user:c1:c1:5:2025-01-15T10:30:00Z] Comment text <<}
More content`

    parser := NewParser()
    doc, err := parser.Parse(content)

    if err != nil {
        t.Fatalf("Parse failed: %v", err)
    }
    if len(doc.Comments) != 1 {
        t.Errorf("Expected 1 comment, got %d", len(doc.Comments))
    }
}
```

### Manual Testing

Use the test utility:
```bash
# Build test parser
go build -o test-parser ./cmd/test-parser

# Test parsing
./test-parser examples/sample.md
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
- Verification step re-parses to confirm all comments/replies were added
- Better for scripting and automation
- Reduces I/O overhead

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

### Modifying Comment Format

**WARNING**: Comment format changes require backward compatibility!

1. Update parser to handle both old and new formats
2. Use colon counting or other heuristics to detect format version
3. Update serializer to write new format only
4. Add round-trip test: old format → parse → serialize → should be new format
5. Test with existing documents

## Debugging Tips

### TUI Issues

If the TUI hangs or doesn't respond:
- Check that the mode is included in `updateByMode()` switch
- Verify viewport initialization happens in `handleResize()` when `ready = false`
- Look for blocking operations in Update() - all I/O should be async

### Parser Issues

If comments aren't parsed:
- Check colon counting logic (timestamps have colons!)
- Verify regex patterns match actual format
- Use test-parser utility to see parsed output
- Add debug logging in parser.go temporarily

### Thread Building Issues

If threads show wrong structure:
- Verify ThreadID is set correctly on all comments
- Check that root comments have `ParentID == ""`
- Ensure replies have both ThreadID and ParentID
- Print thread structure in test to visualize

## Environment Variables

- `ANTHROPIC_API_KEY`: Required for LLM integration (`ask` command)
- `USER`: Used as default author name for comments (falls back to "user")

## Dependencies

Key external dependencies:
- **Bubbletea** (`charmbracelet/bubbletea`): TUI framework
- **Bubbles** (`charmbracelet/bubbles`): UI components (filepicker, viewport, textarea)
- **Lip Gloss** (`charmbracelet/lipgloss`): Terminal styling
- **Anthropic SDK** (via API): Claude integration

All dependencies are vendored via `go.mod`.
