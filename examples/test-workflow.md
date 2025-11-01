# Collaborative Writing Test Document

This document demonstrates the complete workflow for comments and suggestions.

## Introduction

The comments system allows teams to collaborate on markdown documents with inline feedback. Think of it as Google Docs commenting for terminal-based editing.

## Core Features

### Comments
Comments let you ask questions, provide feedback, or mark TODOs without modifying the original text. They're perfect for:
- Ask questions and request clarification
- Provide feedback and suggestions
- Flag bugs and issues
- Build threaded discussions

### Suggestions

Suggestions go beyond comments by proposing actual edits with preview and accept/reject workflow. This is like track changes in Microsoft Word.

## How It Works

The system uses sidecar JSON files to store all comments and suggestions separately from your markdown. This keeps your content clean while enabling powerful collaboration features.

### Storage Model
- Markdown files remain pristine
- Comments stored in `.md.comments.json`
- Version control both separately
- No markup pollution

## Example Use Cases

### Documentation Review
Technical writers can receive feedback from engineers (e.g., "add error code reference here") without cluttering the document with inline markup.

### Academic Writing
Advisors can suggest changes to student papers, and students can accept or reject each suggestion individually.

### Blog Post Editing
Editors can propose rewording while the author maintains final control over which changes to accept.

## Advanced Features

### Batch Operations
For LLM agents and automation, batch operations allow multiple comments or suggestions in one command.

### Conflict Detection
When multiple suggestions overlap, the system automatically detects conflicts and helps resolve them.

### Position Tracking
As the document changes, comment positions are automatically recalculated to stay accurate.

## Conclusion

This system bridges the gap between terminal-based editing and collaborative document workflows.
