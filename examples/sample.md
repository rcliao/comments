# Product Design Document

## Overview

This document outlines the architecture for our new commenting system. {>>[@user:c1:c1:5:2025-01-15T10:30:00Z] Should we also consider scalability for 10k+ concurrent users? The current architecture might need load balancing. <<}

## Core Features

1. Inline comments {>>[@claude:c2:c2:9:2025-01-15T11:00:00Z] This is a great feature set. I'd also recommend adding: 5. Comment search/filtering, 6. Export to HTML with comments <<}
2. Thread support
3. LLM integration

## Technical Stack {>>[@rcliao:c1760728303217563000:c1760728303217563000:13:2025-10-17T12:11:43-07:00] Test comment for line 13

This comment is multiple lines <<} {>>[@rcliao:c1760728337419422000:c1760728303217563000:0:2025-10-17T12:12:17-07:00] A test reply to the thread of test comment

Multiple lines still <<}

- Go + Bubbletea
- CriticMarkup protocol {>>[@user:c3:c3:20:2025-01-15T12:00:00Z] Can we document the exact syntax we're using? <<} {>>[@tester:c1760729560218111000:c1760729560218111000:20:2025-10-17T12:32:40-07:00] Testing the new add command <<} {>>[@reviewer:c1760729568782877000:c1760729560218111000:20:2025-10-17T12:32:48-07:00] This is a test reply <<}
- Terminal rendering

## Implementation Plan

Phase 1: Core parser
Phase 2: TUI development
Phase 3: LLM integration {>>[@claude:c4:c4:27:2025-01-15T13:00:00Z] For Phase 3, I recommend starting with Claude API first, then adding OpenAI support. We should also consider local models via Ollama. <<}
