# comments doctor — install preflight

## Problem

The plugin ships a skill and an MCP server; both are dead unless the `comments`
binary is on PATH, and nothing says so until a call fails with `command not
found`. Verifying this install took four separate manual checks — binary on
PATH, plugin enabled, MCP handshake, sidecars loadable. An agent that hits a
missing binary mid-loop reports a broken tool rather than a missing install
step, and burns a turn doing it.

## Change

Add `comments doctor`: one command, one pass/warn/fail line per check. Default
target is the current directory; `doctor [path]` overrides.

- **binary** — resolved path and version. Needs a version to report: add
  `-X main.version` to `.goreleaser.yaml`, then reconcile the three strings
  that disagree today (plugin manifest 2.1.0, `pkg/mcp/server.go:13` 1.0.0,
  binary none).
- **mcp** — probe `comments serve-mcp` with a stateless `server/discover`
  (SEP-2575), falling back to `initialize` for older servers; report the tool
  count. Catches a stale binary serving an old tool set.
- **plugin** — read `~/.claude/plugins/installed_plugins.json` for install
  state. Warn-only: it is another tool's undocumented file, so an unexpected
  shape reports "unknown" rather than failing.
- **sidecars** — load every `*.comments.json` under the target; report the
  total and how many are stale (document-hash mismatch).

Exit 0 when every check passes, 1 on any failure; warnings alone stay 0 so CI
can gate on it. `--json` for agents.

Where: `doctorCommand` in `cmd/comments/main.go` following the `gateCommand`
pattern; check logic in `pkg/comment/doctor.go` so MCP can expose it later
without duplicating it.

Rejected: folding these checks into `gate`. Gate answers "is this document
approved"; doctor answers "is this install sound". Separate concerns.

## Definition of Done

- [automated] `go test ./pkg/comment -run TestDoctor` — healthy install,
  missing binary, stale sidecar.
- [automated] `comments doctor --json` emits one object per check with `name`,
  `status`, `detail`; exit 0 on a clean machine.
- [manual] With no `comments` on PATH, the output names the install command
  instead of reporting a generic failure.
