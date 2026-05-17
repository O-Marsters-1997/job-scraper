#!/usr/bin/env bash
# Runs `just ci` before gh pr create. Blocks the PR if any step fails.
set -euo pipefail

echo "Running just ci before raising PR..."
echo ""

if ! just ci; then
  echo ""
  echo "PR blocked: just ci failed. Fix the issues above and try again." >&2
  exit 2
fi

echo ""
echo "just ci passed — proceeding with PR."
