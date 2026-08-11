# Plan: Typora-grade in-place rendering for the view command

## Overview

Make `comments view` read like a minimal markdown viewer — Typora as the north star (thread:research-diagram-render.md#cmhir) — without breaking the line-mapping rail.
Three moves: fences stop being mis-styled prose and gain per-line syntax highlight; inline markup gets real styling with markers dimmed; structural glyphs get typographic polish.
One new dependency (chroma); two in-repo lexers; everything line-preserving.

## Current State

- Fence interiors get prose styling — no fence awareness in `styleMarkdownLine` (research-diagram-render.md:32-33).
- chroma is verified line-preserving with go/yaml/json/sql/bash lexers; mermaid and dbml are missing, custom lexers register in-code (research-diagram-render.md:54-58).
- Glamour stays out: whole-document only, reflow breaks anchors (research-diagram-render.md:52-54); standing decision plan-tui-in-context.md:19.
- Bold/italic/code spans with dimmed markers, bullet and quote-bar coloring ALREADY ship (pkg/tui/rendering.go:85-131) — the inline delta is smaller than it looks.
- Decisions (rcliao, chat, pending thread ratification): ASCII-render veto (thread:research-diagram-render.md#cma7o); Typora north star (thread:research-diagram-render.md#cmhir).

## Desired End State

A design doc with DBML, mermaid and Go fences reads as a document: highlighted code, dimmed markers, clean bullets.
Every anchor, citation peek and the gutter still address exact source lines.
Verify: side-by-side before/after on a real doc at review; suite proves line counts unchanged for every rendered form.

## What We're NOT Doing

- No marker CONCEALMENT in v1 — markers dim, never disappear: `suggest --original` must match raw source, and terminal copy of concealed text would silently break that match. Concealment can come later behind a source-mode toggle.
- No reflow of any kind: no table column alignment, no width-aware typography (standing rail).
- No ASCII diagram rendering (vetoed, thread:research-diagram-render.md#cma7o).
- No theme system changes — new styles join the existing styleSet/theme registry.

## Implementation Phases

### Phase 1: fence tracking, suppression, and highlight

Track fence state through the document render (markdown/refs.go already models the fence logic).
Inside a fence: suppress prose styling; highlight the WHOLE block with chroma, split back to lines — multi-line constructs color correctly, count-preservation verified.

Citation composition, decided here: split each fence line at its comment marker.
Code goes to chroma; the trail keeps the existing citation styling — no ANSI/offset collision, and DBML trails are comments anyway.
Register in-repo lexers for `dbml` and minimal `mermaid`; fence delimiters render dimmed, language label visible.

**Success Criteria**
- automated: fence interiors never carry heading/bullet/quote styling; chroma output line-count equals source for every fixture; dbml + mermaid lexers tokenize sample blocks; suite green under -race
- manual: the probe-eval example's DBML block reads highlighted in view

### Phase 2: the inline DELTA — strikethrough and links

Bold/italic/code with dimmed markers already ship; this phase adds only what is missing: `~~strike~~` as ANSI strikethrough, and `[text](target)` with text link-colored and the `(target)` dimmed.
Peekable citations keep their existing styling; line width never changes.

**Success Criteria**
- automated: strike and link fixtures assert styled text + dimmed markers, byte-identical ignoring ANSI; existing span tests untouched; cursor/focus lines still raw
- manual: a sembr paragraph with mixed spans reads clean at review

### Phase 3: structural typography

Bullets render as `•`; `---` rules draw a dim line across the pane; heading `#` runs dim, titles keep their color; blockquote bars continuous.
The `#` line-number toggle and all gutter behavior untouched.

**Success Criteria**
- automated: each glyph fixture line-count-stable; hidden-line-numbers mode unaffected
- manual: your read on a real doc — closer to Typora, or name the gap

## Risks

- **chroma's ANSI interacts with lipgloss backgrounds** on cursor/range lines.
  Mitigated: cursor/focus/range lines already render raw text (existing rule); highlight applies only off-cursor.
- **Custom lexer quality** — a bad DBML lexer mis-colors the section reviewed hardest.
  Mitigated: lexer fixtures from the real probe-eval example; plain fallback on tokenize error.
