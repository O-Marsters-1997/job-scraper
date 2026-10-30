#!/bin/sh
set -eu

base="$1"
dir=scripts/migrations

latest=$(git ls-tree --name-only "$base" "$dir/" | sed -n 's|.*/\([0-9]\{14\}\)_.*\.sql$|\1|p' | sort | tail -n 1)
[ -n "$latest" ] || exit 0

status=0
for f in $(git diff --name-only --no-renames --diff-filter=A "$base" HEAD -- "$dir/*.sql"); do
	version=$(basename "$f" | cut -d_ -f1)
	if [ "$version" -le "$latest" ]; then
		echo "::error file=$f::$version is not after $latest, the newest migration on $base; goose rejects it as missing once $latest is applied. Rename it with a later timestamp (just migrate-create)."
		status=1
	fi
done
exit $status
