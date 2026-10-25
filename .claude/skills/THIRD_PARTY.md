# Third-party skills

Files copied unchanged from [grafana/skills](https://github.com/grafana/skills) (Apache-2.0, see `GRAFANA-LICENSE`)
at commit `3b653d9b46120c05ec02b27895028e1c31308e18` on 2026-10-25, for `app-performance-testing`.

Only the files relevant to this app (a cookie-authenticated HTTP JSON API, light 1 → 5 → 10 user
checks, LogQL/PromQL against Grafana Cloud, TraceQL against local Tempo) are kept. Upstream's
browser, cloud, gRPC, WebSocket, HTML, crypto and `ext-*` k6 files and Tempo's deployment
reference were left out, so some links in `k6/SKILL.md` and `tempo/SKILL.md` point at files that
aren't here.

| Skill | Upstream path | Files kept |
|---|---|---|
| `k6` | `skills/grafana-k6/k6` | `SKILL.md`, `SETUP.md`, `docs-guidance.md`, `examples/http.js`, `examples/executors.js`, `examples/metrics.js` |
| `promql` | `skills/grafana-core/promql` | all |
| `loki` | `skills/grafana-lgtm/loki` | all |
| `tempo` | `skills/grafana-lgtm/tempo` | `SKILL.md`, `references/traceql.md` |

Refresh (from the repo root):

```bash
T=$(mktemp -d) && git clone -q --depth=1 https://github.com/grafana/skills "$T" && S="$T/skills" D=.claude/skills && \
rm -rf $D/k6 $D/promql $D/loki $D/tempo && mkdir -p $D/k6/examples $D/tempo/references && \
cp $S/grafana-k6/k6/{SKILL.md,SETUP.md,docs-guidance.md} $D/k6/ && cp $S/grafana-k6/k6/examples/{http,executors,metrics}.js $D/k6/examples/ && \
cp -r $S/grafana-core/promql $S/grafana-lgtm/loki $D/ && cp $S/grafana-lgtm/tempo/SKILL.md $D/tempo/ && \
cp $S/grafana-lgtm/tempo/references/traceql.md $D/tempo/references/ && cp "$T/LICENSE" $D/GRAFANA-LICENSE && \
git -C "$T" rev-parse HEAD && rm -rf "$T"
```

Then update the commit SHA and date above.
