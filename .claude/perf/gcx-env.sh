#!/usr/bin/env bash
set -euo pipefail
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
set -a
. ./.env
set +a
export GRAFANA_SERVER="$GRAFANA_URL" GRAFANA_TOKEN="$GRAFANA_SA_TOKEN" GCX_TELEMETRY=disabled
exec gcx "$@"
