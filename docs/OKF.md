# OKF bundles in Comments

Comments uses [Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md) as the default shape for newly created artifacts. The result is a portable Markdown knowledge bundle with a review and approval layer that remains useful to both agents and humans.

## What comes from OKF

At its minimum, an OKF bundle is a directory of Markdown concepts with YAML frontmatter. A concept requires a non-empty `type`. A reserved `index.md` can provide navigation, and the root index can declare `okf_version`. Version 0.2 also defines optional provenance and lifecycle fields such as `sources`, `generated`, `verified`, `stale_after`, `status`, and `superseded_by`.

Comments preserves that open shape. The Markdown does not require Comments to remain readable, searchable, or reusable by another OKF consumer.

## What Comments adds

Comments is an opinionated OKF producer and a review-aware consumer. Its additions are deliberately separable from the portable concept:

| Surface | Purpose | Part of OKF v0.2? |
|---|---|---|
| YAML fields such as `type`, `title`, `status`, `sources` | concept identity, lifecycle, and provenance | Yes |
| `.comments/bundle.yaml` | map templates to project folders and generate indexes | No; Comments producer configuration |
| `comments.template` frontmatter | select the structural writing and validation contract | No; namespaced Comments extension |
| `related` frontmatter | record typed, deterministic document edges | No; Comments producer extension |
| `doc.md.comments.json` | anchored threads, suggestions, verdicts, and review history | No; Comments collaboration layer |
| `comments context` | assemble an explainable, role-scoped working set | No; Comments agent interface over the bundle |

This separation is the central benefit. Frontmatter answers “what is this and where did it come from?”, Markdown answers “what does it say?”, and the sidecar answers “what are we still discussing, and did a human approve it?”

## Default folder map

The first `comments new` in a repository creates `.comments/bundle.yaml` and the standard bundle automatically:

```text
.comments/
  bundle.yaml
docs/artifacts/
  index.md
  research/
  plans/
  designs/
  decisions/
  as-built/
  briefs/
```

The generated configuration maps built-in templates to one collection:

| Collection | OKF type | Templates |
|---|---|---|
| `research/` | `Research` | `research`, `research-deep` |
| `plans/` | `Plan` | `plan` |
| `designs/` | `Design` | `design-doc`, `rfc` |
| `decisions/` | `Decision` | `adr` |
| `as-built/` | `AsBuilt` | `as-built` |
| `briefs/` | `Brief` | `mini` |

Commit `.comments/bundle.yaml` when the taxonomy is shared project policy. Teams can change the root, collection paths, types, and template assignments. A template must map to exactly one collection so agents never guess where a new artifact belongs.

`index.md` files are generated navigation. Run `comments bundle index` after manual metadata changes; `comments new` refreshes them automatically.

## Concept format

`comments new cache-policy --template plan --from docs/artifacts/research/cache-policy.md` produces a concept shaped like this:

```markdown
---
comments:
  template: plan
description: Implementation and verification strategy for cache invalidation.
related:
  - path: ../research/cache-policy.md
    relation: informed_by
status: draft
title: Cache Policy
type: Plan
---

# Cache Policy

## Overview
```

Only `type` is required by OKF itself. Comments also validates that a concept inside a configured collection has frontmatter, uses an allowed status (`draft`, `stable`, or `deprecated`), and matches the collection type. The template adds its own section and writing constraints.

Frontmatter is excluded from template word counts, citation checks, heading parsing, and rendered document prose. Its source lines remain in place, so inline comment and citation line numbers still match the Markdown file.

The optional `sources` field is especially useful for research:

```yaml
sources:
  - id: okf-spec
    resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
    title: Open Knowledge Format v0.2 specification
```

Standard OKF trust fields can coexist with Comments metadata. Comments does not currently turn a review signoff into an OKF `verified` attestation automatically; the review record remains in the sidecar.

## Agent context

`comments context` packages a bounded neighborhood as JSON. Every included edge carries a reason, so an agent can distinguish an explicit relation from a Markdown link, backlink, source, or tag suggestion.

```bash
comments context docs/artifacts/plans/cache-policy.md \
  --for drafting --include-threads --json
```

Choose the smallest role that matches the task:

| Mode | Intended use | Boundary |
|---|---|---|
| `drafting` | write or revise the current concept | may add up to five tag-related suggestions |
| `review` | inspect the artifact and its declared neighborhood | no tag-based expansion |
| `human-review` | prepare a human-facing review view | no tag-based expansion |
| `coverage-scout` | find missing research questions | exposes only the Research Question as `focus`; forcibly excludes bodies, threads, and draft-derived links or backlinks |
| `evidence-verifier` | check findings against sources | no tag-based expansion; bodies are opt-in |
| `implementation` | resume or monitor an approved plan | plan-only; returns approval freshness, ordered phases, latest status, success criteria, warnings, and phase-scoped attention |

Bodies and review threads are excluded unless explicitly requested. `--include-body` can produce a large response, so agents should start with metadata and relations, then load only what they need.

Context complements citations; it does not replace evidence. A related document explains why another artifact may matter. A source or `file:line` citation explains why a claim should be trusted.

## Research → Plan → Implement

Use one slug across phases and let `--from` record lineage:

```bash
comments new cache-policy --template research-deep
comments context docs/artifacts/research/cache-policy.md --for drafting

comments new cache-policy --template plan \
  --from docs/artifacts/research/cache-policy.md
comments context docs/artifacts/plans/cache-policy.md --for drafting --include-threads
comments analyze docs/artifacts/plans/cache-policy.md \
  --against docs/artifacts/research/cache-policy.md --json
comments context docs/artifacts/plans/cache-policy.md --for implementation
```

For work spanning days, a phase may carry an H4 `Status` list. Each dated top-level entry uses a `pending`, `active`, `blocked`, or `done` state and nests `Summary`, `Evidence`, and `Next` fields; the latest entry wins. Status history is capped separately at 20 entries and 60 words per entry, and does not consume the plan's normal document or phase word budget. New plan signoffs hash both the full document and stable intent, so status-only edits preserve approval while scope or success-criteria edits make it stale.

The agent validates the artifact and posts specific anchored doubts with `comments add`, `comments batch-add`, or `comments suggest`. There is no seeding step and no separate annotation command.

For handoff, the human opens `comments view <doc>` or `comments serve <doc>`. The agent can remain active with:

```bash
comments watch docs/artifacts/plans/cache-policy.md --until signoff
```

The TUI or browser verdict writes the signoff. When the event arrives, the agent reads `comments inbox` before acting because thread replies carry the human’s detailed feedback. `comments gate` remains a mechanical check; a clean gate without a signoff is not proof of human review.

## Existing repositories

OKF is the creation default, not a migration requirement.

- Existing Markdown and sidecars keep working in place.
- The first `comments new` adds a bundle; it does not move or rewrite old files.
- Template resolution remains compatible with explicit flags and legacy sidecar metadata.
- Teams can link new bundle concepts to older Markdown with normal Markdown citations, while `related` paths are intended for concepts inside the knowledge bundle.

The maintained worked examples in [`docs/examples/`](examples/) include OKF-compatible frontmatter. This repository’s live dogfood bundle is [`docs/artifacts/`](artifacts/).

## References

- [Open Knowledge Format v0.2 specification](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
- [Google Cloud: OKF v0.2 adds trust signals](https://cloud.google.com/blog/products/data-analytics/okf-v0-2-adds-trust-signals/)
- [Comments usage guide](../USAGE.md)
- [Comments architecture](ARCHITECTURE.md)
