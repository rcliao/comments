# Comments CLI - Usage Guide (v2.0)

A terminal-based collaborative document commenting tool designed for seamless LLM integration.

## Features

### ✅ Core Functionality
- **Interactive TUI** - Browse, create, and manage comments in a split-pane interface
- **Threading** - Reply to comments to form conversation threads (nested structure)
- **Resolution** - Mark comment threads as resolved
- **Section-Based Addressing** - Add comments by markdown section paths
- **Suggestions** - Propose multi-line edits with preview and accept/reject workflow
- **Batch Operations** - Efficient JSON-based bulk operations for LLM agents
- **@filename Support** - Read text content from external files
- **Document Staleness Detection** - Automatic hash-based validation

## Commands

### 1. View Mode (Interactive TUI)

```bash
# Open file picker to select a file
./comments view

# Open a specific file directly
./comments view document.md
```

**Keyboard Shortcuts:**

#### Browse Mode
- `j/k` or `↓/↑` - Navigate through comments
- `c` - Enter line selection mode to add a comment
- `Enter` - Expand selected comment to view full thread
- `R` - Toggle showing/hiding resolved comments
- `q` - Return to file picker
- `Ctrl+C` - Quit application

#### Line Selection Mode
- `j/k` or `↓/↑` - Move cursor to select line
- `c` or `Enter` - Open comment input modal
- `Esc` - Cancel and return to browse mode

#### Add Comment Mode
- Type your comment in the textarea
- `Ctrl+S` - Save comment
- `Esc` - Cancel

#### Thread View Mode
- `r` - Reply to the thread
- `x` - Resolve the thread
- `Esc` - Return to browse mode
- `q` - Return to file picker

#### Reply Mode
- Type your reply in the textarea
- `Ctrl+S` - Save reply
- `Esc` - Cancel

#### Resolve Mode
- `y` or `Enter` - Confirm resolution
- `n` or `Esc` - Cancel

### 2. Add Command

Add a comment to a document:

```bash
# Add by line number
./comments add document.md --line 10 --author "alice" --text "Review this" --type Q

# Add by section path
./comments add document.md --section "Introduction > Overview" --author "bob" --text "Expand this"

# Read text from file
./comments add document.md --line 25 --author "claude" --text @comment.txt
```

**Flags:**
- `--line <N>` - Line number (mutually exclusive with --section)
- `--section <path>` - Section path like "Title > Subtitle" (mutually exclusive with --line)
- `--author <name>` - Author name (required)
- `--text <text|@file>` - Comment text or @filename to read from file (required)
- `--type <Q|S|B|T|E>` - Comment type: Question, Suggestion, Bug, TODO, Enhancement (optional)

### 3. Reply Command

Reply to an existing thread:

```bash
./comments reply document.md --thread c123 --author "alice" --text "I agree"

# Use @filename for long replies
./comments reply document.md --thread c456 --author "bob" --text @reply.txt
```

### 4. Suggest Command

Create a multi-line edit suggestion:

```bash
# Inline text
./comments suggest document.md --start-line 15 --end-line 17 \
  --author "claude" --text "Improve clarity" \
  --original "old text" --proposed "new text"

# Use @filename for long text blocks
./comments suggest document.md --start-line 20 --end-line 25 \
  --author "claude" --text "Refactor section" \
  --original @original.txt --proposed @proposed.txt
```

**Flags:**
- `--start-line <N>` - Start line (required)
- `--end-line <N>` - End line (required)
- `--author <name>` - Author name (required)
- `--text <text|@file>` - Description of change (required)
- `--original <text|@file>` - Original text being replaced (required)
- `--proposed <text|@file>` - Proposed replacement text (required)

### 5. Accept/Reject Suggestions

Review and accept or reject suggestions:

```bash
# Preview changes
./comments accept document.md --suggestion s123 --preview

# Accept and apply
./comments accept document.md --suggestion s123

# Reject
./comments reject document.md --suggestion s456
```

### 6. List Command

List all comments with optional filters:

```bash
# List all unresolved comments
./comments list document.md

# Filter by author
./comments list document.md --author alice

# Filter by section (includes nested sections)
./comments list document.md --section "Implementation"

# Filter by type
./comments list document.md --type Q

# Search text
./comments list document.md --search "TODO"

# Line range
./comments list document.md --lines 10-30

# Show resolved comments
./comments list document.md --resolved true

# JSON output (includes full metadata)
./comments list document.md --format json

# Combine filters
./comments list document.md --section "Intro" --author alice --type Q
```

**Output Format:**
- Table format (default): Shows root comments only with summary
- JSON format: Full metadata including all replies

### 7. Batch Operations

Efficient bulk operations for LLM agents using JSON input:

#### Batch Add

```bash
# Create JSON file
cat > comments.json << 'EOF'
[
  {
    "line": 10,
    "author": "claude",
    "text": "Consider edge cases",
    "type": "Q"
  },
  {
    "section": "Implementation > Architecture",
    "author": "claude",
    "text": "Add diagram",
    "type": "S"
  }
]
EOF

./comments batch-add document.md --json comments.json

# Or use stdin for single-command workflow
echo '[{"line":10,"author":"claude","text":"Good point","type":"Q"}]' | \
  ./comments batch-add document.md --json -
```

**JSON Fields:**
- `line` OR `section` (mutually exclusive, required)
- `author` (required)
- `text` (required)
- `type` (optional: Q, S, B, T, E)

#### Batch Reply

```bash
# Create JSON file
cat > replies.json << 'EOF'
[
  {
    "thread": "c123",
    "author": "claude",
    "text": "Good point about scalability"
  },
  {
    "thread": "c456",
    "author": "claude",
    "text": "I agree with this approach"
  }
]
EOF

./comments batch-reply document.md --json replies.json

# Or use stdin
echo '[{"thread":"c123","author":"claude","text":"LGTM"}]' | \
  ./comments batch-reply document.md --json -
```

**JSON Fields:**
- `thread` (thread ID, required)
- `author` (required)
- `text` (required)

## Storage Format (v2.0)

Comments are stored in JSON sidecar files (`.md.comments.json`) alongside your markdown documents.

### Example Sidecar File

```json
{
  "version": "2.0",
  "documentHash": "sha256_hash_of_markdown",
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
      "OriginalText": "old text",
      "ProposedText": "new improved text",
      "Accepted": null,
      "Replies": []
    }
  ]
}
```

### Threading Model (v2.0)

Comments use a **nested structure** with `Replies` arrays:
- Root comments have `Replies` containing direct child comments
- Each reply can have its own `Replies` array for nested conversations
- No separate `ThreadID`/`ParentID` fields needed (simplified from v1.x)

## Workflow Examples

### Example 1: Adding Comments During Review

1. Start the TUI: `./comments view document.md`
2. Press `c` to enter line selection mode
3. Navigate with `j/k` to line you want to comment on
4. Press `c` to open comment modal
5. Type your comment
6. Press `Ctrl+S` to save
7. Comment appears in the right panel
8. File is automatically saved

### Example 2: Threading a Discussion

1. In browse mode, navigate to a comment with `j/k`
2. Press `Enter` to expand the thread
3. Press `r` to reply
4. Type your response
5. Press `Ctrl+S` to save
6. Reply appears in thread view
7. Press `Esc` to return to browse mode

### Example 3: Resolving Comments

1. Expand a comment thread with `Enter`
2. Press `x` to resolve
3. Confirmation dialog appears
4. Press `y` or `Enter` to confirm
5. Thread is marked as resolved
6. Returns to browse mode
7. Toggle resolved comments with `R`

### Example 4: LLM-Assisted Writing

```bash
# Get suggestions for a specific section
./comments ask draft.md \\
  --prompt "Suggest improvements for clarity" \\
  --start 10 --end 20 \\
  --line 10

# Ask for expansion ideas
./comments ask draft.md \\
  --prompt "What topics should I cover next?" \\
  --line 25

# Open in TUI to review LLM comments
./comments view draft.md
```

## UI Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ 📄 document.md - Mode: Browse                                   │
├──────────────────────────────────┬──────────────────────────────┤
│ Document Content (60%)           │ Comments Panel (40%)         │
│                                  │                              │
│  1    # Title                    │ Comments (3 unresolved)      │
│  2                               │                              │
│  3    Introduction text...       │ Line 5 • @user               │
│  4                               │ 2025-01-15 10:30             │
│  5 💬1 ## Section 1              │ Should we add...             │
│  6    Content here...            │ └─ 2 replies                 │
│  7                               │                              │
│  8                               │ Line 12 • @claude            │
│  9 💬2 ## Section 2              │ 2025-01-15 11:00             │
│ 10    More content...            │ Consider using...            │
│                                  │ └─ 0 replies                 │
│                                  │                              │
└──────────────────────────────────┴──────────────────────────────┘
│ j/k: navigate • c: comment • Enter: expand • R: toggle resolved │
└─────────────────────────────────────────────────────────────────┘
```

## Environment Variables

- `USER` - Used as default author name for comments in TUI mode

## Tips

1. **Keyboard-First**: All operations can be done without a mouse
2. **Quick Navigation**: Use `j/k` for vi-like navigation
3. **Section Paths**: Use full hierarchical paths like "Introduction > Overview > Background"
4. **@filename Syntax**: Great for long comments or when text is already in a file
5. **Batch Operations**: More efficient than multiple individual commands for bulk operations
6. **Resolved Toggle**: Press `R` in browse mode to see all comments
7. **Preview Suggestions**: Always use `--preview` before accepting to see what will change
8. **File Auto-Save**: Changes are saved immediately when you create/reply/resolve

## Troubleshooting

**Q: Comments not showing up?**
A: Press `R` to toggle resolved comments visibility

**Q: Can't add comments?**
A: Make sure you're in line selection mode (press `c` from browse mode)

**Q: LLM not working?**
A: Check that `ANTHROPIC_API_KEY` is set correctly

**Q: File not saving?**
A: Check file permissions and that the file path is correct

## Technical Details

- **Storage Format**: JSON sidecar files (`.md.comments.json`)
- **Threading Model**: Nested structure with `Replies` arrays (v2.0)
- **Position Tracking**: Line-based (simplified in v2.0)
- **Suggestions**: Multi-line only (v2.0)
- **Staleness Detection**: SHA-256 hash validation prevents data corruption
- **File Format**: UTF-8 markdown (clean) + JSON metadata (sidecar)

## Document Staleness Detection

When you load a document, the system:
1. Computes SHA-256 hash of markdown content
2. Compares with hash stored in sidecar file
3. If hashes don't match, the sidecar is considered stale
4. Stale sidecars are archived to `.backup.TIMESTAMP` files
5. You can choose to start fresh or restore from backup

This prevents data corruption when markdown content changes outside the tool.
