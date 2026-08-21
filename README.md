# Comments

Google-Docs-style review for markdown, in the terminal. Inline comment threads and edit suggestions live in sidecar JSON files next to your docs — a TUI for the human, a CLI + MCP server for agents, and a machine-readable review gate between them.

[![asciicast](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf.svg)](https://asciinema.org/a/z6fSaof32MYS36NOtZ5Oj84Lf)

## Overview

`comments` is built for human↔agent doc collaboration (spec-driven development). Instead of having an LLM rewrite entire documents, the agent drafts under a template that keeps docs short and reviewable, the human walks the comment threads in the TUI and signs off, and the agent addresses feedback one comment at a time until the gate opens.

## Features

- **Inline comments & threads**: anchored to lines or markdown sections, with nested replies and content-based re-anchoring when the doc changes
- **Edit suggestions**: multi-line proposals with preview and accept/reject; queued decisions apply atomically at review verdict
- **Review gate**: `comments gate` exits 0 (approved) or 10 (changes requested); `signoff` records the human pass agents block on
- **Doc templates as guardrails**: required sections, word caps, forced alternatives, human-owned zones, `[NEEDS CLARIFICATION:]` marker caps — built-ins: `design-doc`, `mini`, `research`, `plan`, `adr`, `rfc`, `as-built`
- **RPI flow**: research docs with file:line evidence → plans citing the research → reviewed in the TUI where `f` peeks any citation and Enter opens `$EDITOR` there
- **Autonomous research convergence**: draft-blind coverage scout + evidence verifier add missing `Qn` questions until clean; `comments analyze plan.md --against research.md` proves the handoff before review
- **Watch**: `comments watch --until signoff` streams NDJSON review events so agents can wait on humans
- **MCP server**: 21 tools over stdio for agent integration; batch operations; `@filename` text input
- **Surface parity**: every MCP tool has a CLI equivalent backed by the same code — see `docs/ARCHITECTURE.md` decision 8

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
comments view doc.md            # TUI review; q -> a/c signs off (n adds a note)
comments add doc.md --line 10 --author eric --text "Fix this first" --blocking
comments gate doc.md            # exit 0 = approved, 10 = changes requested
comments signoff doc.md         # same review record, non-interactively (CI, --note)
comments watch specs/ --until signoff     # block until signed off, either way
```

Agent-side drafting under a template:

```bash
comments template list                        # when to use which, per description
comments template show design-doc             # the writing brief
comments validate draft.md --template design-doc
comments seed draft.md --template design-doc  # criteria + markers become review threads
comments analyze plan.md --against research.md --json  # advisory coverage manifest
```

## TUI keys

`j/k` move · `r` dive into thread at cursor · `Tab` cycle stacked threads · `n/N` next/prev NEW since your last signoff · `f` peek citation (Enter → `$EDITOR` at file:line) · `t` table of contents · `a`/`x` queue accept/reject on suggestions · `q` verdict (approve / request changes, `n` for a review note) · `?` full keybinding help

## What the templates produce

Every template ships with a worked example under [`docs/examples/`](docs/examples/) — real subjects from this repo, written to every constraint, validating clean:

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

Comments live in `doc.md.comments.json` sidecars: markdown stays clean, collaboration data versions independently, and a SHA-256 document hash drives staleness detection and the re-anchoring cascade (exact → text → fuzzy → section → orphan).

## Documentation

- [docs/README.md](docs/README.md) — documentation status, active proposals, and retained design records
- [CLAUDE.md](CLAUDE.md) — command reference, architecture, agent workflow
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — system design and data model
- [USAGE.md](USAGE.md) — current CLI and TUI workflow guide
- [skills/review-comments/SKILL.md](skills/review-comments/SKILL.md) — the agent skill (bundled by the plugin)

## License

MIT
