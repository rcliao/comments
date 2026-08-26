---
comments:
    template: mini
description: Implementation brief for bundle creation, context loading, and unified review annotations.
related:
    - path: ../research/okf-comments-workflow.md
      relation: informed_by
status: draft
title: Ship OKF-Aware Comments Workflow
type: Brief
---

# Ship OKF-Aware Comments Workflow

## Problem

Agents could follow a document template, but they still guessed where artifacts belonged and searched folders for related work.
Template identity lived in review sidecars because the removed `seed` command doubled as configuration.
That mixed durable document metadata with discussion threads.

## Change

Add a committed `.comments/bundle.yaml` that maps templates to typed folders beneath one OKF knowledge root.
`comments new` creates the Markdown skeleton, OKF-compatible frontmatter, review sidecar, and generated indexes.

Add `comments context` with drafting, review, coverage-scout, evidence-verifier, and human-review modes.
It returns explicit relations, Markdown links, backlinks, sources, review state, and bounded tag suggestions with inclusion reasons.
Coverage-scout mode cannot return document bodies, threads, or draft-derived edges.

Resolve templates from an explicit argument, `comments.template` frontmatter, a legacy sidecar, or a single-template collection.
Keep `add`, `batch-add`, and `suggest` as the only annotation surfaces.
Remove `seed`; agents now post specific inline doubts and use `watch --until signoff` to listen for the human verdict.

Expose equivalent `comments_new`, `comments_context`, and `comments_bundle_index` MCP tools.
Update the review skill so RPI creates related research and plan concepts and loads role-scoped context before drafting.

## Definition of Done

- Automated: `go test ./...` and `./scripts/ci.sh` pass.
- Automated: bundle, metadata, frontmatter, context isolation, CLI, and MCP round trips have tests.
- Manual: the generated research and this related brief validate without an explicit template flag.
- Manual: `context` shows the research edge from this brief and the backlink from research.
- Manual: an agent-authored inline comment remains visible for human review in the TUI or browser.
