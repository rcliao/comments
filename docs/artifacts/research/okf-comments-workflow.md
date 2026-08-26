---
comments:
    template: research-deep
description: Dogfood record for bundle-aware agent and human document review.
generated:
    by: agent:codex
    at: 2026-08-26T12:20:00-07:00
sources:
    - id: okf-spec
      resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
      title: Open Knowledge Format v0.2 specification
    - id: okf-v02
      resource: https://cloud.google.com/blog/products/data-analytics/okf-v0-2-adds-trust-signals/
      title: OKF v0.2 adds trust signals
status: draft
tags: [agents, okf, review]
title: OKF Comments Workflow
type: Research
---

# OKF Comments Workflow

## Research Question

Q1. Does an OKF-style bundle reduce the agent's document discovery work?

Q2. Can template guidance and inline discussion remain one workflow without conflating metadata and comments?

Q3. Does role-scoped context preserve the RPI review boundaries?

## Summary

The bundle is useful when it stays a thin discovery layer over the existing review loop.
Frontmatter makes type, template, and relationships available before an agent reads the prose.
The folder map makes new artifact placement deterministic.
Comments remain the discussion channel, and `watch` remains the signoff listener.

The strongest agent benefit is not the generated index itself.
It is one explainable context response that replaces ad hoc folder scans while preserving role boundaries.

## Findings

### F1 — Template-to-folder mapping removes placement guesses [Q1]

The committed bundle config maps every template to one typed collection.
Creation rejects templates assigned to zero or multiple collections, then writes the concept to the selected folder.
This gives humans a predictable review queue and gives agents the same path convention without prompt-specific instructions. pkg/comment/bundle.go:123

Template identity resolves from an explicit argument, frontmatter, a legacy sidecar, or an unambiguous collection.
That ordering lets new artifacts be self-describing while old documents continue to work. pkg/comment/bundle.go:171

### F2 — Metadata and discussion now have separate jobs [Q2]

OKF-compatible frontmatter carries durable document facts: type, status, template, relations, tags, and sources.
Sidecars still carry thread history, suggestions, verdicts, and anchors.
Validation combines the template rules with an OKF metadata floor only for concepts inside a configured collection. pkg/comment/bundle.go:207

The skill now tells agents to add specific comments about weak reasoning or decisions.
It explicitly avoids generic criterion threads and keeps template identity in frontmatter. skills/review-comments/SKILL.md:137

### F3 — Context modes are enforceable working-set boundaries [Q1] [Q3]

`comments context` returns relationships with reasons instead of an opaque relevance score.
Explicit frontmatter edges, Markdown links, backlinks, sources, and limited tag suggestions remain distinguishable in the response. pkg/comment/context.go:100

The coverage-scout mode forcibly strips bodies and threads, and it excludes draft-derived links and backlinks.
Tag suggestions appear only during drafting, so strict reviewer roles do not silently widen their evidence set. pkg/comment/context.go:75

### F4 — Dogfood exposed a useful limit [Q3]

Context is most useful after at least two concepts are related.
For the first concept, it mainly confirms metadata, template, and review state.
That makes `context` a valuable chain-navigation command, but not a replacement for repository research or cited evidence.

## Code References

- `pkg/comment/bundle.go:14` — bundle schema, discovery, creation, template resolution, and validation.
- `pkg/comment/context.go:59` — deterministic role-scoped neighborhood construction.
- `pkg/comment/metadata.go:27` — OKF-compatible metadata parsing.
- `skills/review-comments/SKILL.md:101` — agent creation, context, annotation, and listening workflow.

## Open Questions

- Adopt this bundle as the default for new RPI artifacts, or keep it opt-in until several projects confirm the folder map?
- Do these findings justify a follow-up plan for richer context ranking, or is deterministic navigation sufficient for now?
