# Template examples

These files are maintained examples for every built-in document template. Each one has OKF-compatible frontmatter so it shows the metadata a live `comments new` artifact carries, and each validates against `comments.template` without requiring an explicit `--template` flag.

They are intentionally static examples outside this repository’s configured `docs/artifacts` bundle. For a real artifact, use `comments new`; it selects the collection, creates the sidecar, and refreshes bundle indexes.

| Example | Type | Template | Creation command |
|---|---|---|---|
| [research.md](research.md) | `Research` | `research` | `comments new <slug> --template research` |
| [plan.md](plan.md) | `Plan` | `plan` | `comments new <slug> --template plan --from <research.md>` |
| [design-doc.md](design-doc.md) | `Design` | `design-doc` | `comments new <slug> --template design-doc` |
| [rfc.md](rfc.md) | `Design` | `rfc` | `comments new <slug> --template rfc` |
| [adr.md](adr.md) | `Decision` | `adr` | `comments new <slug> --template adr` |
| [as-built.md](as-built.md) | `AsBuilt` | `as-built` | `comments new <slug> --template as-built` |
| [mini.md](mini.md) | `Brief` | `mini` | `comments new <slug> --template mini` |

Validate all examples from the repository root:

```bash
for doc in docs/examples/{research,plan,design-doc,rfc,adr,as-built,mini}.md; do
  comments validate "$doc"
done
```

See [OKF bundles in Comments](../OKF.md) for the format boundary, default folder layout, and context modes.
