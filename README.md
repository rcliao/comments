# Comments

A CLI tool for collaborative document writing with inline comments and suggestions, designed for seamless LLM integration.

[![asciicast](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf.svg)](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf)

## Overview

`comments` brings Google Doc-style commenting and track-changes functionality to terminal-based markdown editing, enabling better collaboration between humans and LLM agents. Instead of having LLMs rewrite entire documents, you can add contextual comments at specific lines to guide iteration and discussion, or propose edits as suggestions with preview and accept/reject workflow.

## Features

- **Inline Comments**: Add comments to specific lines or markdown sections
- **Edit Suggestions**: Propose multi-line changes with preview and accept/reject workflow
- **Thread Support**: Reply to comments and suggestions to build conversation threads
- **Section-Based Addressing**: Reference comments by markdown heading hierarchy
- **Terminal UI**: Beautiful, keyboard-driven interface built with Bubbletea
- **LLM Integration**: Collaborate with AI agents using contextual comments and suggestions
- **JSON Sidecar Storage**: Clean separation with `.md.comments.json` files
- **Document Staleness Detection**: Automatic hash-based validation prevents data corruption
- **@filename Support**: Read comment text from external files (great for LLM agents)
- **Batch Operations**: Efficient JSON-based bulk comment and reply operations

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

# Add a comment to a section
comments add document.md --section "Introduction > Overview" --author "bob" --text "Expand this"

# Create a multi-line suggestion
comments suggest document.md --start-line 15 --end-line 17 --author "claude" \
  --text "Improve clarity" --original "old text" --proposed "new text"

# Use @filename to read text from file
comments add document.md --line 25 --author "alice" --text @comment.txt

# Accept suggestion with preview
comments accept document.md --suggestion c123 --preview
comments accept document.md --suggestion c123

# List all comments and suggestions
comments list document.md

# List with filters
comments list document.md --section "Implementation" --author "alice"

# Batch operations (great for LLM agents)
echo '[{"line":10,"author":"claude","text":"Good point","type":"Q"}]' | \
  comments batch-add document.md --json -
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

# Suggestions
comments suggest <file> [options]         # Create multi-line suggestion
comments accept <file> --suggestion <id>  # Accept and apply suggestion
comments reject <file> --suggestion <id>  # Reject suggestion

# Batch Operations
comments batch-add <file> --json <file>   # Batch add comments from JSON
comments batch-reply <file> --json <file> # Batch reply to threads from JSON
```

## Storage Format (v2.0)

Comments and suggestions are stored in JSON sidecar files (`.md.comments.json`) alongside your markdown files. This approach:
- Keeps markdown files clean and readable
- Enables structured metadata and advanced features
- Separates content from collaboration data
- Supports independent version control of comments
- Provides automatic staleness detection via document hashing

### Example Storage Format

```json
{
  "version": "2.0",
  "documentHash": "sha256_hash",
  "threads": [
    {
      "ID": "c123",
      "Author": "alice",
      "Text": "[Q] What about edge cases?",
      "Line": 10,
      "SectionPath": "Introduction > Overview",
      "Replies": [
        {
          "ID": "c124",
          "Author": "bob",
          "Text": "Good point, let me add tests",
          "Replies": []
        }
      ]
    }
  ]
}
```

## Documentation

- **Architecture**: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - System design and data model
- **Development Guide**: [CLAUDE.md](CLAUDE.md) - Implementation details for contributors
- **Usage Guide**: [USAGE.md](USAGE.md) - Complete command reference

## License

MIT
