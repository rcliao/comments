#!/usr/bin/env bash
# Reports whether this repo's git hooks (.githooks) are wired up in the repo
# the caller is standing in. Exit 0 = wired, 1 = not wired (with a note saying
# how to fix it). ci.sh runs it after the gates; the smoke test runs it against
# scratch repos.
#
# It asks git where hooks actually run from instead of comparing the config
# string: an absolute core.hooksPath is just as wired as ".githooks", and a
# string compare once reported a clone whose pre-push ran on every push as
# "not wired up".
set -euo pipefail

not_wired() {
  printf '\nNote: git hooks are not wired up in this clone (%s). Enable them with:\n' "$1"
  printf '  git config core.hooksPath .githooks\n'
  exit 1
}

# Unset or empty means git falls back to .git/hooks, which holds whatever
# happens to be there (husky, stale copies), not this repo's gates.
configured=$(git config --get core.hooksPath || true)
[ -n "$configured" ] || not_wired "core.hooksPath is unset"

# Relative to the current directory, which is where we test the hooks from.
hooks_dir=$(git rev-parse --git-path hooks)
for hook in pre-commit pre-push; do
  [ -x "$hooks_dir/$hook" ] || not_wired "$hooks_dir/$hook is missing or not executable"
done
