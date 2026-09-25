#!/usr/bin/env bash
# Pushes ops/grafana/ (contact point, notification policy, alert rules, dashboards) to Grafana
# via its HTTP API, upserting by UID so a repeat run changes nothing. Requires yq and jq.
#
# Usage: GRAFANA_URL=... GRAFANA_SA_TOKEN=... scripts/grafana-push.sh
set -euo pipefail

: "${GRAFANA_URL:?GRAFANA_URL is required}"
: "${GRAFANA_SA_TOKEN:?GRAFANA_SA_TOKEN is required}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ALERTS_DIR="$ROOT/ops/grafana/alerts"
DASHBOARDS_DIR="$ROOT/ops/grafana/dashboards"
BODY_FILE="$(mktemp)"
trap 'rm -f "$BODY_FILE"' EXIT

# req METHOD PATH [JSON_BODY] -> prints HTTP status, leaves the response body in $BODY_FILE
req() {
	local method="$1" path="$2" data="${3:-}"
	if [ -n "$data" ]; then
		curl -sS -o "$BODY_FILE" -w '%{http_code}' -X "$method" \
			-H "Authorization: Bearer $GRAFANA_SA_TOKEN" -H "Content-Type: application/json" \
			-d "$data" "$GRAFANA_URL$path"
	else
		curl -sS -o "$BODY_FILE" -w '%{http_code}' -X "$method" \
			-H "Authorization: Bearer $GRAFANA_SA_TOKEN" "$GRAFANA_URL$path"
	fi
}

# upsert PATH UID JSON_BODY -> PUT to update; on 404, POST to create (uid is in the body)
upsert() {
	local path="$1" uid="$2" data="$3" status
	status="$(req PUT "$path/$uid" "$data")"
	if [ "$status" = "404" ]; then
		status="$(req POST "$path" "$data")"
	fi
	case "$status" in
	2??) ;;
	*)
		echo "grafana-push: $path/$uid failed ($status): $(cat "$BODY_FILE")" >&2
		exit 1
		;;
	esac
}

echo "grafana-push: discovering datasources"
status="$(req GET /api/datasources)"
if [ "$status" != "200" ]; then
	echo "grafana-push: listing datasources failed ($status): $(cat "$BODY_FILE")" >&2
	exit 1
fi
PROM_UID="$(jq -r '[.[] | select(.type=="prometheus")][0].uid // empty' "$BODY_FILE")"
LOKI_UID="$(jq -r '[.[] | select(.type=="loki")][0].uid // empty' "$BODY_FILE")"
[ -n "$PROM_UID" ] || {
	echo "grafana-push: no prometheus datasource connected" >&2
	exit 1
}
[ -n "$LOKI_UID" ] || {
	echo "grafana-push: no loki datasource connected" >&2
	exit 1
}

echo "grafana-push: ensuring folder"
FOLDER_UID="$(yq -r '.folder.uid' "$ALERTS_DIR/rules.yaml")"
FOLDER_TITLE="$(yq -r '.folder.title' "$ALERTS_DIR/rules.yaml")"
status="$(req GET "/api/folders/$FOLDER_UID")"
if [ "$status" = "404" ]; then
	folder_body="$(jq -n --arg uid "$FOLDER_UID" --arg title "$FOLDER_TITLE" '{uid: $uid, title: $title}')"
	status="$(req POST /api/folders "$folder_body")"
	[ "$status" = "200" ] || {
		echo "grafana-push: creating folder failed ($status): $(cat "$BODY_FILE")" >&2
		exit 1
	}
fi

echo "grafana-push: pushing contact points"
while IFS= read -r cp; do
	uid="$(echo "$cp" | jq -r '.uid')"
	upsert /api/v1/provisioning/contact-points "$uid" "$cp"
done < <(yq -o=json '.contactPoints[]' -I=0 "$ALERTS_DIR/contact-points.yaml")

echo "grafana-push: pushing notification policy"
policy="$(yq -o=json -I=0 "$ALERTS_DIR/notification-policy.yaml")"
status="$(req PUT /api/v1/provisioning/policies "$policy")"
case "$status" in
2??) ;;
*)
	echo "grafana-push: notification policy failed ($status): $(cat "$BODY_FILE")" >&2
	exit 1
	;;
esac

echo "grafana-push: pushing alert rules"
while IFS= read -r rule; do
	uid="$(echo "$rule" | jq -r '.uid')"
	body="$(echo "$rule" | jq --arg folder "$FOLDER_UID" --arg group "$(yq -r '.ruleGroup' "$ALERTS_DIR/rules.yaml")" \
		'. + {folderUID: $folder, ruleGroup: $group}')"
	body="${body//__PROM_DATASOURCE_UID__/$PROM_UID}"
	body="${body//__LOKI_DATASOURCE_UID__/$LOKI_UID}"
	upsert /api/v1/provisioning/alert-rules "$uid" "$body"
done < <(yq -o=json '.rules[]' -I=0 "$ALERTS_DIR/rules.yaml")

echo "grafana-push: pushing dashboards"
for dashboard_file in "$DASHBOARDS_DIR"/*.json; do
	dashboard="$(cat "$dashboard_file")"
	dashboard="${dashboard//__PROM_DATASOURCE_UID__/$PROM_UID}"
	dashboard="${dashboard//__LOKI_DATASOURCE_UID__/$LOKI_UID}"
	body="$(jq -n --argjson dashboard "$dashboard" --arg folder "$FOLDER_UID" \
		'{dashboard: $dashboard, folderUid: $folder, overwrite: true}')"
	status="$(req POST /api/dashboards/db "$body")"
	case "$status" in
	2??) ;;
	*)
		echo "grafana-push: dashboard $(basename "$dashboard_file") failed ($status): $(cat "$BODY_FILE")" >&2
		exit 1
		;;
	esac
done

echo "grafana-push: done"
