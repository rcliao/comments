#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
skill="$repo_root/skills/review-comments/SKILL.md"

required=(
  "Coverage scout — source-derived, draft-blind"
  "NEVER the draft"
  "Evidence verifier — draft-derived"
  "next \`Qn\`"
  "negative-coverage memory"
  "scout pass adds no accepted question"
  "no blocking verifier thread"
  "resolved round-summary thread"
  "stop after 3 semantic passes"
  "pause-on-shape"
  "comments analyze <plan.md> --against <research.md> --json"
  "only the human plan signoff authorizes implementation"
)

for phrase in "${required[@]}"; do
  if ! grep -Fq "$phrase" "$skill"; then
    printf 'autonomous research contract missing: %s\n' "$phrase" >&2
    exit 1
  fi
done

echo "autonomous research contract present"
