# Comments

Google-Docs-style review for markdown, locally. Inline comment threads and edit suggestions live in sidecar JSON files next to your docs — a TUI or browser workspace for the human, a CLI + MCP server for agents, and a machine-readable review gate between them.

[![asciicast](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf.svg)](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf)

## Overview

`comments` is built for human↔agent doc collaboration (spec-driven development). Instead of having an LLM rewrite entire documents, the agent creates a typed knowledge artifact, drafts under a template that keeps it short and reviewable, and annotates uncertain reasoning inline. The human walks those threads in the TUI or browser and signs off; the agent listens for that verdict and addresses feedback until the gate opens.

## Features

- **Inline comments & threads**: anchored to lines or markdown sections, with nested replies and content-based re-anchoring when the doc changes
- **Edit suggestions**: multi-line proposals with preview and accept/reject; queued decisions apply atomically at review verdict
- **Review gate**: `comments gate` exits 0 (approved) or 10 (changes requested); `signoff` records the human pass agents block on
- **Doc templates as guardrails**: required sections, word caps, forced alternatives, human-owned zones, `[NEEDS CLARIFICATION:]` marker caps — built-ins: `design-doc`, `mini`, `research`, `plan`, `adr`, `rfc`, `as-built`
- **OKF document bundles by default**: the first `comments new` initializes a standard `docs/artifacts` bundle, then creates frontmatter-rich concepts in template-guided folders; `comments context` exposes explicit relations, backlinks, sources, and review state without a whole-tree search
- **RPI flow**: research docs with file:line evidence → plans citing the research → reviewed in the TUI where `f` peeks any citation and Enter opens `$EDITOR` there
- **Autonomous research convergence**: draft-blind coverage scout + evidence verifier add missing `Qn` questions until clean; `comments analyze plan.md --against research.md` proves the handoff before review
- **Watch**: `comments watch --until signoff` streams NDJSON review events so agents can wait on humans
- **Browser review**: `comments serve` opens a rendered document and line-accurate source view beside live threads, suggestions, and verdict controls
- **MCP server**: 23 tools over stdio for agent integration; batch operations; `@filename` text input
- **Surface parity**: every MCP tool has a CLI equivalent backed by the same code — see `docs/ARCHITECTURE.md` decision 8

## Why OKF and Comments fit together

[Open Knowledge Format (OKF) v0.2](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md) makes a knowledge base portable: Markdown concepts carry YAML frontmatter, folders and `index.md` files provide navigation, and optional trust fields describe provenance and lifecycle. Comments adds the collaborative layer that the format deliberately does not prescribe.

| Layer | Owns | Benefit |
|---|---|---|
| OKF-compatible frontmatter and folders | type, title, status, provenance, relations, placement | agents can discover and traverse artifacts without guessing filenames or searching the whole repository |
| Markdown | research, design, plan, decision, or as-built content | the durable artifact remains readable in any Markdown tool |
| `.comments.json` sidecar | anchored threads, suggestions, verdicts, review history | agents and humans can debate and approve the artifact without polluting its content or metadata |

The first `comments new` initializes a default bundle at `docs/artifacts`; no setup command is required. Comments-specific producer configuration lives in `.comments/bundle.yaml`, while `comments.template` and `related` extend otherwise portable OKF frontmatter. Existing Markdown remains supported and is never migrated automatically. See [the OKF bundle guide](docs/OKF.md) for the exact boundary and format.

## Install

```bash
# the binary (required) — prebuilt, no Go toolchain needed:
# grab the archive for your platform from the latest release
#   https://github.com/rcliao/comments/releases/latest
curl -sL https://github.com/rcliao/comments/releases/latest/download/comments_darwin_arm64.tar.gz | tar xz comments && mv comments ~/.local/bin/

# or, with Go installed:
go install github.com/rcliao/comments/cmd/comments@latest

# the Claude Code plugin: review-comments skill + MCP server, one install
/plugin marketplace add rcliao/comments
/plugin install comments@comments
```

## The core loop

```bash
comments new cache-policy --template design-doc
comments context docs/artifacts/designs/cache-policy.md --for drafting --include-threads
comments add docs/artifacts/designs/cache-policy.md --section "Proposed Design" \
  --author agent --text "[Q] The repository does not yet establish the proposed TTL." --blocking
comments validate docs/artifacts/designs/cache-policy.md
comments watch docs/artifacts/designs/cache-policy.md --until signoff
```

The human reviews the same artifact while the agent listens:

```bash
comments view docs/artifacts/designs/cache-policy.md   # q -> a/c/r records a verdict
comments serve docs/artifacts/designs/cache-policy.md  # browser alternative
```

After the signoff event, the agent reads `comments inbox docs/artifacts/designs/cache-policy.md` first, fixes or replies to each thread, and checks `comments gate` (exit 0 = approved, 10 = changes requested). `comments signoff` is the non-interactive verdict writer for CI or scripts; a TUI/browser verdict already records the signoff.

For Research → Plan, use the same slug and preserve lineage:

```bash
comments new cache-policy --template research-deep
comments new cache-policy --template plan --from docs/artifacts/research/cache-policy.md
comments analyze docs/artifacts/plans/cache-policy.md \
  --against docs/artifacts/research/cache-policy.md --json
```

## TUI keys

`j/k` move · `r` dive into thread at cursor · `Tab` cycle stacked threads · `n/N` next/prev NEW since your last signoff · `f` peek citation (Enter → `$EDITOR` at file:line) · `t` table of contents · `a`/`x` queue accept/reject on suggestions · `q` verdict (approve / request changes, `n` for a review note) · `?` full keybinding help

## What the templates produce

Every template ships with a self-describing, OKF-compatible worked example under [`docs/examples/`](docs/examples/) — real subjects from this repo, written to every constraint and validating clean. These are static examples; `comments new` places live artifacts in the configured bundle.

| Template | Example | Shows off |
|---|---|---|
| `design-doc` | [design-doc.md](docs/examples/design-doc.md) | one-pager: data flow story, full DBML model, contract interfaces |
| `as-built` | [as-built.md](docs/examples/as-built.md) | the gate/signoff loop as it runs today, with peekable evidence |
| `research` | [research.md](docs/examples/research.md) | documentarian findings with file:line per claim |
| `plan` | [plan.md](docs/examples/plan.md) | phases with automated/manual success criteria |
| `adr` | [adr.md](docs/examples/adr.md) | one decision, honest consequences |
| `rfc` | [rfc.md](docs/examples/rfc.md) | thread citations, guide + reference level |
| `mini` | [mini.md](docs/examples/mini.md) | a whole change in 400 words |

Review any of them in the tool itself: `comments view docs/examples/design-doc.md` — peek the citations with `f`.

## Storage

The knowledge bundle and the review record are intentionally separate:

- `.comments/bundle.yaml` maps templates to typed folders and generates navigational indexes;
- `docs/artifacts/**/*.md` contains portable OKF-compatible concepts;
- `doc.md.comments.json` contains collaboration state beside each reviewed concept.

Sidecars keep Markdown clean, version collaboration independently, and use a SHA-256 document hash to drive the re-anchoring cascade (exact → text → fuzzy → section → orphan).

## Documentation

- [docs/README.md](docs/README.md) — documentation status, active proposals, and retained design records
- [CLAUDE.md](CLAUDE.md) — command reference, architecture, agent workflow
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — system design and data model
- [USAGE.md](USAGE.md) — current CLI and TUI workflow guide
- [docs/OKF.md](docs/OKF.md) — OKF v0.2 boundary, default folder map, frontmatter, context modes, and RPI example
- [skills/review-comments/SKILL.md](skills/review-comments/SKILL.md) — the agent skill (bundled by the plugin)

## License

MIT
