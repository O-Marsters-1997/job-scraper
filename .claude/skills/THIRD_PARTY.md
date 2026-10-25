# Third-party skills

Copied unchanged from [grafana/skills](https://github.com/grafana/skills) (Apache-2.0, see `GRAFANA-LICENSE`)
at commit `3b653d9b46120c05ec02b27895028e1c31308e18` on 2026-10-25. Used by `app-performance-testing`.

| Skill | Upstream path |
|---|---|
| `k6` | `skills/grafana-k6/k6` |
| `promql` | `skills/grafana-core/promql` |
| `tempo` | `skills/grafana-lgtm/tempo` |
| `loki` | `skills/grafana-lgtm/loki` |

Refresh (from the repo root):

```bash
T=$(mktemp -d) && git clone -q --depth=1 https://github.com/grafana/skills "$T" && for p in grafana-k6/k6 grafana-core/promql grafana-lgtm/tempo grafana-lgtm/loki; do rm -rf ".claude/skills/${p#*/}" && cp -r "$T/skills/$p" ".claude/skills/${p#*/}"; done && cp "$T/LICENSE" .claude/skills/GRAFANA-LICENSE && git -C "$T" rev-parse HEAD && rm -rf "$T"
```

Then update the commit SHA and date above.
