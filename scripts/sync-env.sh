#!/usr/bin/env bash
# Syncs an env file's key set to .env.example: adds missing keys with the example's
# value, drops keys .env.example no longer lists. Existing values are left untouched.
#
# Usage: scripts/sync-env.sh [target]   (default target: .env)
set -euo pipefail
cd "$(dirname "$0")/.."

example=".env.example"
target="${1:-.env}"

[ -f "$example" ] || {
  echo "missing $example" >&2
  exit 1
}
[ -f "$target" ] || touch "$target"
cp "$target" "$target.bak"

example_keys=$(grep -oE '^[A-Z_][A-Z0-9_]*=' "$example" | sed 's/=$//')

removed=0
tmp=$(mktemp)
while IFS= read -r line; do
  key=$(printf '%s' "$line" | sed -n 's/^\([A-Z_][A-Z0-9_]*\)=.*/\1/p')
  if [ -n "$key" ] && ! grep -qx "$key" <<<"$example_keys"; then
    echo "- $key (no longer in $example)"
    removed=$((removed + 1))
    continue
  fi
  echo "$line" >>"$tmp"
done <"$target"
mv "$tmp" "$target"

added=0
while IFS= read -r key; do
  if ! grep -qE "^${key}=" "$target"; then
    grep -m1 -E "^${key}=" "$example" >>"$target"
    echo "+ $key"
    added=$((added + 1))
  fi
done <<<"$example_keys"

echo "done: +$added -$removed (backup: $target.bak)"
