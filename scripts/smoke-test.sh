#!/usr/bin/env bash
# Review-flow smoke test: drives the real CLI through a full review cycle.
#
# CI and developers run THIS script — the workflow does not carry its own copy.
# A duplicated smoke test would drift exactly the way this repo's hand-written
# tool banner and usage text did.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

# A stale root binary has masqueraded as missing features before; always rebuild.
go build -o comments ./cmd/comments

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
doc="$workdir/ci-doc.md"

printf '%s\n' '---' 'comments:' '  template: design-doc' '---' '' '# Doc' '' '## Pitch' '' 'Cache it: reads are slow and a cache is the smallest fix.' '' '## Problem' '' 'It is slow.' '' '## Goals / Non-Goals' '' 'Faster. Non-goal: rewrite.' '' '## Proposed Design' '' 'Cache.' '' '## Options Considered' '' '### Option 1: Cache (recommended)' '' 'Good.' '' '### Option 2: Rewrite' '' 'Big.' '' '## Risks' '' 'Staleness: accepted.' '' '## Definition of Done' '' 'Cache hit rate measured above 90 percent in the smoke benchmark.' '' '## Unresolved Questions' '' 'None.' > "$doc"

./comments validate "$doc"
./comments add "$doc" --anchor 'It is slow.' --author agent --type Q --blocking --text 'Is this the right problem framing?'
./comments add "$doc" --anchor 'Staleness: accepted.' --author agent --type Q --blocking --text 'Confirm the accepted risk.'

# The gate must fail (exit 10) while explicit blocking threads are open.
if ./comments gate "$doc"; then
  echo "FAIL: gate should have failed with blocking threads open" >&2
  exit 1
fi
echo "✓ gate fails while blocking threads are open"

# zone:human guard: with no TTY and no override the CLI is treated as an agent,
# so a thread added to a human-decision section (design-doc marks Problem as
# zone: human) must be refused. Regression cover for the /dev/null bypass.
human_id=$(./comments get "$doc" --json 2>/dev/null \
  | jq -r '[.comments[] | select(.section_path | test("Problem"))][0].id')
if [ -z "$human_id" ] || [ "$human_id" = "null" ]; then
  echo "FAIL: no human-zone thread found to test the guard" >&2
  exit 1
fi
if ./comments reply "$doc" --thread "$human_id" --author agent --text 'done' --resolve >/dev/null 2>&1; then
  echo "FAIL: zone:human guard did not refuse an agent resolve" >&2
  exit 1
fi
# The refusal is atomic: the reply that travelled with the resolve must not land.
if [ "$(./comments get "$doc" --thread "$human_id" --json 2>/dev/null | jq -r '.comment.reply_count')" != "0" ]; then
  echo "FAIL: a refused resolve still posted its reply" >&2
  exit 1
fi
echo "✓ zone:human guard refuses an agent resolve"

# The verdict and suggestion decisions are the human's. They are made in
# `comments view` / `comments serve`, so no command may exist that makes them:
# a scripted signoff once let an agent record an approval under a human's name.
for removed in signoff accept reject batch-accept; do
  if ./comments "$removed" "$doc" >/dev/null 2>&1; then
    echo "FAIL: '$removed' must not exist as a command" >&2
    exit 1
  fi
done
echo "✓ no command can record a verdict or decide a suggestion"

# CI stands in for the human reviewer here — that is what the override is for.
# Agents must not set it.
export COMMENTS_ACTOR=human
for id in $(./comments get "$doc" --json 2>/dev/null | jq -r '.comments[].id'); do
  ./comments reply "$doc" --thread "$id" --resolve >/dev/null
done
./comments gate "$doc"
echo "✓ gate passes once the human resolves the blocking threads"

# The agent's single read agrees with the gate: approved, and nothing waiting.
inbox=$(./comments inbox "$doc" --json)
if [ "$(jq -r '.decision' <<<"$inbox")" != "approved" ] || [ "$(jq -r '.count' <<<"$inbox")" != "0" ]; then
  echo "FAIL: inbox disagrees with the gate: $inbox" >&2
  exit 1
fi
echo "✓ inbox reports approved with nothing waiting"

# doctor must report a sound install; warnings are allowed, failures are not.
# Scans the repo, whose docs/ carry real sidecars.
./comments doctor . --json >/dev/null
./comments doctor .
echo "✓ doctor reports a sound install"

# Surface parity: the skill's whole review loop, driven through the CLI and
# through a live MCP server on twin documents. Fails if the two ever succeed
# differently, return different JSON, or leave different sidecars — and if an
# agent can resolve in a human zone or reach a human decision.
go build -o "$workdir/verdict" ./scripts/eval/surface-parity/verdict
python3 scripts/eval/surface-parity/drive.py ./comments "$workdir/parity" "$workdir/verdict" | tail -2
echo "✓ CLI and MCP agree across the full review loop"

# Hooks check: ci.sh once string-compared core.hooksPath to ".githooks" and
# called a clone with an absolute path "not wired up" while its pre-push ran.
# Each case runs check-hooks.sh inside a scratch repo (git -C would not do: the
# script resolves hooks relative to where it stands).
check_hooks="$repo_root/scripts/check-hooks.sh"
hooks_repo="$workdir/hooks-repo"
git init -q "$hooks_repo"
mkdir -p "$hooks_repo/.githooks"
printf '#!/bin/sh\n' > "$hooks_repo/.githooks/pre-commit"
printf '#!/bin/sh\n' > "$hooks_repo/.githooks/pre-push"
chmod +x "$hooks_repo/.githooks/pre-commit" "$hooks_repo/.githooks/pre-push"
expect_hooks() { # <want: wired|unwired> <label>
  # Ignore global/system config: a developer's global core.hooksPath would
  # otherwise break the "unset" case.
  if (cd "$hooks_repo" && GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 "$check_hooks" >/dev/null); then
    got=wired
  else
    got=unwired
  fi
  if [ "$got" != "$1" ]; then
    echo "FAIL: hooks check with $2: want $1, got $got" >&2
    exit 1
  fi
}
expect_hooks unwired "core.hooksPath unset"
git -C "$hooks_repo" config core.hooksPath .githooks
expect_hooks wired "a relative core.hooksPath"
git -C "$hooks_repo" config core.hooksPath "$hooks_repo/.githooks"
expect_hooks wired "an absolute core.hooksPath"
chmod -x "$hooks_repo/.githooks/pre-push"
expect_hooks unwired "a non-executable pre-push"
echo "✓ hooks check follows where git runs hooks from"

echo "SMOKE TEST PASSED"
