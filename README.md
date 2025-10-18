# Comments

A CLI tool for collaborative document writing with inline comments, designed for seamless LLM integration.

[![asciicast](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf.svg)](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf)

## Overview

`comments` brings Google Doc-style commenting to terminal-based markdown editing, enabling better collaboration between humans and LLM agents. Instead of having LLMs rewrite entire documents, you can add contextual comments at specific lines to guide iteration and discussion.

## Features

- **Inline Comments**: Add comments to specific lines in markdown files
- **Thread Support**: Reply to comments and build conversation threads
- **Terminal UI**: Beautiful, keyboard-driven interface built with Bubbletea
- **LLM Integration**: Collaborate with AI agents using contextual comments
- **CriticMarkup Protocol**: Uses extended CriticMarkup syntax for compatibility

## Installation

```bash
go install github.com/rcliao/comments/cmd/comments@latest
```

## Quick Start

```bash
# Open interactive viewer
comments view document.md

# Add a comment from command line
comments add document.md --line 10 --text "What about edge cases?"

# List all comments
comments list document.md

# Ask LLM for help
comments ask document.md --context "lines 5-15" --prompt "Expand this section"
```

## Usage

### Interactive Mode

Press `c` to add a comment, `Enter` to view a comment thread, `r` to reply, and `q` to quit.

### Command Line

```bash
comments view <file>              # Open interactive TUI
comments add <file> [options]     # Add a comment
comments list <file>              # List all comments
comments resolve <comment-id>     # Mark comment as resolved
comments ask <file> [options]     # Ask LLM for help
```

## Comment Format

Comments use extended CriticMarkup syntax:

```markdown
{>>[@user:comment-id:line:timestamp] comment text <<}
```

This format is:
- Human-readable in plain text
- Compatible with standard markdown processors
- Preserves metadata for threading and tracking

## Project Status

Currently in active development. See [docs/](docs/) for architecture and design documentation.

## License

MIT
