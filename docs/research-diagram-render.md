# Research: rendering diagram blocks in the review TUI

## Research Question

Q1. What renders diagram syntax (mermaid and kin) to ASCII from Go, and how mature is it?
Q2. How does the view command treat fenced blocks today, and what breaks if rendered output is injected below a block?
Q3. Which diagram sources in our docs would benefit — and does DBML fit?

## Summary

A viable renderer exists: pgavlin/mermaid-ascii, an importable MIT Go package covering 22 mermaid types including flowchart, sequence and erDiagram. [Q1]

The TUI styles headings only; fences are plain text, and nothing today inserts rows below source lines. [Q2]
Injection is precedented — wrapped lines already break one-row-per-line — but anchors and the gutter must number only source lines.

The templates already push authors toward diagrams. [Q3]
DBML could reach the same renderer via erDiagram translation, making the data model visual in-terminal.

## Findings

### F1 — An importable renderer exists; quality verified, unevenly [Q1]

pgavlin/mermaid-ascii (MIT) exposes `pkg/render`: `Render(input, config)` — verified empirically, not just from its docs.
Flowcharts and sequence diagrams render cleanly: boxed nodes, arrows, lifelines.

erDiagram is immature: one attribute per entity survives, relations come out as a text line, not connected boxes.
It forks AlexanderGrooff/mermaid-ascii (1.5k stars, CLI-only); the library fork has no tagged releases — a pseudo-versioned dependency.
It is also the only candidate surveyed; none excluded by evidence.

### F2 — Fences are plain text today; no virtual-line machinery exists [Q2]

`styleMarkdownLine` (pkg/tui/rendering.go:137) styles headings, bullets, blockquote bars and inline spans with NO fence awareness — fence interiors get prose styling today, so rendering must add fence tracking AND styling suppression.
Both document renders share one wrap width so scroll math holds across modes (pkg/tui/CLAUDE.md notes this invariant).
End-of-line thread summaries (`L`) are the only virtual text, appended within a line — nothing inserts rows.

### F3 — Injected rows must not renumber the source [Q2]

Comments anchor to source line numbers; the `#` gutter numbers source lines.
Wrapped lines already produce multiple screen rows per source line, so the machinery tolerates row expansion.
A rendered block below a fence is the same shape of expansion, numbered as belonging to no line.
A diagram wider than the doc pane cannot soft-wrap without mangling; clipping or a full-width toggle is required.

### F4 — Authors are already steered toward diagrams; DBML is the prize [Q3]

The design-doc template asks "would an ASCII diagram say it better than prose?" (pkg/comment/templates/design-doc.yaml:35), and Data Flow's criteria treat diagrams as a second telling.
Data models are written as DBML by review preference (decided in PR #12's review).

Mermaid's erDiagram is structurally close to DBML.
But the verified output (F1) drops attributes and draws relations as text — the DBML prize needs upstream er work or a purpose-built table renderer, not just translation.

### F5 — Glamour is still whole-document; chroma is the extractable piece [Q1] [Q2]

Glamour v2 renders whole documents only — no per-block API, no source-line mapping — so the standing decline (plan-tui-in-context.md:19) remains correct, verified against its current README.
Its ingredients separate: chroma, the highlighter glamour uses, imports alone and is verified line-preserving (3 lines in, 3 out).
Per-line fence highlighting therefore fits the line-mapping rail.

chroma ships 150+ lexers (go, yaml, json, sql, bash verified present); mermaid and dbml are MISSING.
Custom lexers register in-code, so a small in-repo DBML lexer (Table/Ref/pk/types/`//` trails) and a minimal mermaid one add zero dependencies beyond chroma itself.
Glamour's style-sheet gallery remains useful as theme reference only.

## Code References

- pkg/tui/rendering.go:137 — styleMarkdownLine (headings only)
- pkg/tui/CLAUDE.md — shared wrap-width invariant the injection must respect
- pkg/comment/templates/design-doc.yaml:31 — the diagram nudge
- github.com/pgavlin/mermaid-ascii — pkg/render API (MIT, untagged)

## Open Questions

- ~~ASCII rendering, dependency, toggles~~ — vetoed in review (thread:cma7o): mermaid-ER reads worse than DBML and the renderer is an untrusted fork. Redirect: better in-place markdown styling instead.
- The redirect's rail is a standing decision: no rendered-markdown mode, line mapping sacred (plan-tui-in-context.md:19). Headroom within it: fence-aware suppression (F2), per-line fence highlighting (F5), span polish.
- ~~Own research or plan from F2?~~ — answered by F5: glamour stays out, chroma comes in, custom lexers in-repo. Enough to plan from.
