# Comments

A CLI tool for collaborative document writing with inline comments and suggestions, designed for seamless LLM integration.

[![asciicast](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf.svg)](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf)

## Overview

`comments` brings Google Doc-style commenting and track-changes functionality to terminal-based markdown editing, enabling better collaboration between humans and LLM agents. Instead of having LLMs rewrite entire documents, you can add contextual comments at specific lines to guide iteration and discussion, or propose edits as suggestions with preview and accept/reject workflow.

## Features

- **Inline Comments**: Add comments to specific lines in markdown files
- **Edit Suggestions**: Propose changes with 4 granularity levels (line, character-range, multi-line, diff-hunk)
- **Preview & Review**: Accept or reject suggestions with visual preview
- **Thread Support**: Reply to comments and suggestions to build conversation threads
- **Terminal UI**: Beautiful, keyboard-driven interface built with Bubbletea
- **LLM Integration**: Collaborate with AI agents using contextual comments and suggestions
- **JSON Storage**: Clean separation with sidecar `.md.comments.json` files
- **Conflict Detection**: Automatic detection and resolution of overlapping suggestions

## Installation

```bash
go install github.com/rcliao/comments/cmd/comments@latest
```

## Quick Start

```bash
# Open interactive viewer
comments view document.md

# Add a comment from command line
comments add document.md --line 10 --author "alice" --text "What about edge cases?" --type Q

# Create a suggestion
comments suggest document.md --line 15 --author "bob" --text "Improve wording" \
  --type line --original "old text" --proposed "new text"

# Accept suggestion with preview
comments accept document.md --suggestion c123 --preview
comments accept document.md --suggestion c123

# Batch accept suggestions
comments batch-accept document.md --author "bob"

# List all comments and suggestions
comments list document.md

# Ask LLM for help
comments ask document.md --context "lines 5-15" --prompt "Expand this section"
```

## Usage

### Interactive Mode (TUI)

**Navigation:**
- `↑/↓` or `j/k`: Navigate comments
- `Enter`: View thread details
- `c`: Add new comment
- `r`: Reply to thread
- `R`: Toggle resolved comments
- `q`: Quit

**Suggestion Review:**
- `a`: Accept suggestion (shows preview)
- `x`: Reject suggestion
- `y/Enter`: Confirm acceptance
- `n/Esc`: Cancel

### Command Line

```bash
# Comments
comments view <file>                      # Open interactive TUI
comments add <file> [options]             # Add a comment
comments list <file> [options]            # List all comments
comments reply <file> [options]           # Reply to thread
comments resolve <file> --thread <id>     # Mark thread as resolved

# Suggestions (NEW)
comments suggest <file> [options]         # Create edit suggestion
comments accept <file> --suggestion <id>  # Accept and apply suggestion
comments reject <file> --suggestion <id>  # Reject suggestion
comments batch-accept <file> [options]    # Batch accept suggestions

# Batch Operations
comments batch-add <file> --json <file>   # Batch add comments/suggestions
comments batch-reply <file> --json <file> # Batch reply to threads

# LLM Integration
comments ask <file> [options]             # Ask LLM for help
```

## Storage Format

Comments and suggestions are stored in JSON sidecar files (`.md.comments.json`) alongside your markdown files. This approach:
- Keeps markdown files clean and readable
- Enables structured metadata and advanced features
- Separates content from collaboration data
- Supports version control of comments independently

### Suggestion Types

1. **Line**: Replace entire line(s)
   ```bash
   --type line --start-line 10 --end-line 10 --original "old" --proposed "new"
   ```

2. **Character Range**: Precise byte-offset replacements
   ```bash
   --type char-range --offset 150 --length 10 --original "old" --proposed "new"
   ```

3. **Multi-Line**: Replace blocks of lines
   ```bash
   --type multi-line --start-line 10 --end-line 15 --original "..." --proposed "..."
   ```

4. **Diff Hunk**: Unified diff format
   ```bash
   --type diff-hunk --proposed "@@ -10,3 +10,3 @@\n-old\n+new"
   ```

## Project Status

Currently in active development. See [docs/](docs/) for architecture and design documentation.

## License

MIT
