#!/usr/bin/env bash
set -euo pipefail

dry_run=false
[ "${1:-}" = "--dry-run" ] && dry_run=true

run() {
	echo "+ $*"
	$dry_run || "$@"
}

worktrees=$(git worktree list --porcelain | sed -n 's/^worktree //p')
main=$(head -n 1 <<<"$worktrees")
live_projects=$(xargs -n 1 basename <<<"$worktrees" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_\n-')
volume_suffix='_(db_data|rabbitmq_data|alloy_data|valkey_data)$'

finished() {
	local project=$1
	case "$project" in
	agent-*) return 0 ;;
	fleet-*)
		! grep -qE "issue-${project#fleet-}([^0-9]|$)" <<<"$worktrees"
		;;
	job-scraper-*) ! grep -qxF "$project" <<<"$live_projects" ;;
	*) return 1 ;;
	esac
}

volumes=$(docker volume ls -q)
stacks=$(docker compose ls -a -q)
projects=$(
	{
		grep -E "$volume_suffix" <<<"$volumes" | sed -E "s/$volume_suffix//"
		echo "$stacks"
	} | grep -E '^(agent-|fleet-[0-9]+$|job-scraper-)' | sort -u
) || true

for project in $projects; do
	finished "$project" || continue
	run docker compose -f "$main/docker-compose.yml" -p "$project" down -v --remove-orphans
	for volume in $(docker volume ls -q | grep -E "^${project}${volume_suffix}"); do
		run docker volume rm "$volume"
	done
done

for status in exited created dead; do
	for id in $(docker ps -aq --filter label=org.testcontainers=true --filter "status=$status"); do
		run docker rm -f "$id"
	done
done

if $dry_run; then
	echo "+ docker volume prune -f  # $(docker volume ls -q --filter dangling=true | grep -cE '^[0-9a-f]{64}$') anonymous volumes"
else
	docker volume prune -f
fi
