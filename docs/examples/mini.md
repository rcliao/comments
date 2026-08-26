---
comments:
  template: mini
description: A small rendering change expressed as a bounded, reviewable brief.
status: stable
title: "Mini: bullet glyph substitution in the document pane"
type: Brief
---

# Mini: bullet glyph substitution in the document pane

## Problem

Markdown bullets render as their source `-`, which reads as punctuation, not structure.
Typora renders `•`; our Phase 3 left substitution out because it is the one place a CHARACTER would change, and terminal copy of a bullet line would yield `•` where the source has `-`.

## Change

Render list bullets as `•` in the document pane only, keeping the source untouched.
The glyph swap happens inside `styleLinePrefix` (pkg/tui/rendering.go), which already isolates the bullet character; width is identical, so wrap and anchors hold.
Cursor, focus and range lines keep raw text (existing rule), so line-select always shows the true source.
A note lands in pkg/tui/CLAUDE.md: this is the single deliberate character substitution in the renderer, and copy-fidelity of bullet lines is knowingly traded.

## Definition of Done

- automated: bullet fixture renders `•`, ANSI-stripped width unchanged; cursor line still shows `-`.
- manual: one real doc read side-by-side; revert if the copy trade-off ever bites.
