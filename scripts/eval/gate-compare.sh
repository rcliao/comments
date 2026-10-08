#!/usr/bin/env bash
# Gate every doc under a folder with two comments binaries and report any doc
# whose decision differs. Used to show a change leaves existing docs gating as
# before: gate-compare.sh <old-binary> <new-binary> [dir]
set -euo pipefail
old=$1 new=$2 dir=${3:-docs/artifacts}
total=0 differ=0 fresh=0
while IFS= read -r doc; do
  total=$((total + 1))
  # gate exits 10 on changes_requested; the JSON is the answer, not the code
  a=$({ "$old" gate "$doc" --json 2>/dev/null || true; } | jq -r '.decision // "error"' 2>/dev/null || echo error)
  b=$({ "$new" gate "$doc" --json 2>/dev/null || true; } | jq -r '.decision // "error"' 2>/dev/null || echo error)
  # a doc the old binary cannot gate at all (a template it never had) is new,
  # not changed
  if [ "$a" = "error" ] || [ -z "$a" ]; then
    fresh=$((fresh + 1))
    printf 'NEW %s: %s\n' "$doc" "$b"
    continue
  fi
  if [ "$a" != "$b" ]; then
    differ=$((differ + 1))
    printf 'DIFFERS %s: %s -> %s\n' "$doc" "$a" "$b"
  fi
done < <(find "$dir" -name '*.md' ! -name 'index.md' ! -path '*/.comments/*' | sort)
printf '%d docs gated, %d differ, %d new\n' "$total" "$differ" "$fresh"
[ "$differ" -eq 0 ]
