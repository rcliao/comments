# Comments CLI - Usage Guide

A terminal-based collaborative document commenting tool with LLM integration.

## Features

### ✅ Core Functionality
- **Interactive TUI** - Browse, create, and manage comments in a split-pane interface
- **Threading** - Reply to comments to form conversation threads
- **Resolution** - Mark comment threads as resolved
- **LLM Integration** - Use Claude to generate comments and suggestions
- **File Management** - Built-in file picker for markdown documents

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

### 2. List Command

List all comments in a file:

```bash
./comments list document.md
```

Output shows:
- Line number
- Author
- Timestamp
- Comment text

### 3. Ask Command (LLM Integration)

Get AI-generated comments using Claude:

```bash
# Ask a question (response shown but not saved)
./comments ask document.md --prompt "Explain this section"

# Ask a question and save as comment at specific line
./comments ask document.md --prompt "Suggest improvements" --line 10

# Provide context for the LLM
./comments ask document.md --prompt "Review this" --start 5 --end 15 --line 5
```

**Flags:**
- `--prompt` (required) - Question or request for Claude
- `--line` (optional) - Line number to add comment at
- `--start` (optional) - Context start line
- `--end` (optional) - Context end line

**Requirements:**
Set `ANTHROPIC_API_KEY` environment variable:

```bash
export ANTHROPIC_API_KEY=your_api_key_here
```

## Comment Format

Comments are stored inline using an extended CriticMarkup format:

```markdown
{>>[@author:id:threadid:line:timestamp] comment text <<}
```

### Threading

- **Root Comment**: ThreadID equals Comment ID, no ParentID
- **Reply**: ThreadID references root comment, ParentID references parent comment

Example with thread:

```markdown
# Document Title

Some content here.

{>>[@user:c1:c1:5:2025-01-15T10:30:00Z] Original comment <<}

{>>[@claude:c2:c1:5:2025-01-15T11:00:00Z] Reply to original <<}

More content.
```

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

- `USER` - Used as default author name for comments
- `ANTHROPIC_API_KEY` - Required for `ask` command

## Tips

1. **Keyboard-First**: All operations can be done without a mouse
2. **Quick Navigation**: Use `j/k` for vi-like navigation
3. **Context Matters**: When using LLM, provide `--start` and `--end` for better results
4. **Resolved Toggle**: Press `R` in browse mode to see all comments
5. **File Auto-Save**: Changes are saved immediately when you create/reply/resolve

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

- **Comment Format**: Extended CriticMarkup with threading metadata
- **Backward Compatible**: Supports both old (4-field) and new (5-field) formats
- **File Format**: UTF-8 markdown with inline comment annotations
- **Threading Model**: Parent-child relationships with ThreadID linking

## Future Enhancements

Potential features for future versions:
- Edit existing comments
- Delete comments
- Comment search and filtering
- Export to HTML with comments
- Multiple LLM providers (OpenAI, local models)
- Comment mentions (@username)
- Comment tags (#topic)
- Diff view showing changes
