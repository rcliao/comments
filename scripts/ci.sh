#!/usr/bin/env bash
# Runs every gate CI runs, in the same order, so a green run here means a green
# run there. Use this before you commit or push — see CLAUDE.md.
#
#   ./scripts/ci.sh              full run
#   SKIP_LINT=1 ./scripts/ci.sh  skip golangci-lint (fast inner loop only —
#                                never as your final check before pushing)
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

# Pinned to the same version .github/workflows/ci.yml uses. Bump both together.
GOLANGCI_VERSION="v2.12.2"

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }

step "gofmt"
unformatted=$(gofmt -l .)
if [ -n "$unformatted" ]; then
  echo "gofmt failures:" >&2
  echo "$unformatted" >&2
  exit 1
fi
echo "clean"

step "build"
go build ./...

step "vet"
go vet ./...

step "test -race"
go test -race ./...

step "lint (golangci-lint $GOLANGCI_VERSION)"
if [ "${SKIP_LINT:-}" = "1" ]; then
  # Loud, and not a pass: lint caught two real defects that gofmt/vet/test did
  # not, so a skipped lint must never read as a green run.
  echo "SKIPPED via SKIP_LINT=1 — re-run without it before pushing" >&2
elif command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run
else
  echo "golangci-lint not on PATH; running it through the Go toolchain"
  go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$GOLANGCI_VERSION" run
fi

step "review-flow smoke test"
./scripts/smoke-test.sh

printf '\n\033[1mAll CI gates passed.\033[0m\n'

# A hook that was never wired up is indistinguishable from a hook that passed,
# so say so rather than letting the guard be silently absent.
if [ "$(git config --get core.hooksPath || true)" != ".githooks" ]; then
  printf '\nNote: git hooks are not wired up in this clone. Enable them with:\n'
  printf '  git config core.hooksPath .githooks\n'
fi
