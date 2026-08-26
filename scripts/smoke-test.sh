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

printf '%s\n' '---' 'comments:' '  template: design-doc' '---' '' '# Doc' '' '## Problem' '' 'It is slow.' '' '## Goals / Non-Goals' '' 'Faster. Non-goal: rewrite.' '' '## Proposed Design' '' 'Cache.' '' '## Options Considered' '' '### Option 1: Cache (recommended)' '' 'Good.' '' '### Option 2: Rewrite' '' 'Big.' '' '## Risks' '' 'Staleness: accepted.' '' '## Definition of Done' '' 'Cache hit rate measured above 90 percent in the smoke benchmark.' '' '## Unresolved Questions' '' 'None.' > "$doc"

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
human_id=$(./comments list "$doc" --format json 2>/dev/null \
  | jq -r '[.[] | select(.section_path | test("Problem"))][0].id')
if [ -z "$human_id" ] || [ "$human_id" = "null" ]; then
  echo "FAIL: no human-zone thread found to test the guard" >&2
  exit 1
fi
if ./comments resolve "$doc" --thread "$human_id" >/dev/null 2>&1; then
  echo "FAIL: zone:human guard did not refuse an agent resolve" >&2
  exit 1
fi
echo "✓ zone:human guard refuses an agent resolve"

# CI stands in for the human reviewer here — that is what the override is for.
# Agents must not set it.
export COMMENTS_ACTOR=human
for id in $(./comments list "$doc" --format json 2>/dev/null | jq -r '.[].id'); do
  ./comments resolve "$doc" --thread "$id" >/dev/null
done
./comments signoff "$doc" --author ci
./comments gate "$doc"
echo "✓ gate passes after review"

# doctor must report a sound install; warnings are allowed, failures are not.
# Scans the repo, whose docs/ carry real sidecars.
./comments doctor . --json >/dev/null
./comments doctor .
echo "✓ doctor reports a sound install"

echo "SMOKE TEST PASSED"
