## Local Development

Start a local stack with Prometheus, [VictoriaMetrics](https://victoriametrics.com/), and [Avalanche](https://github.com/prometheus-community/avalanche):

```bash
docker compose up -d
```

This starts:
- **Avalanche** on `:9001` — generates synthetic metrics with series churn
- **Prometheus** on `:9090` — scrapes Avalanche every 15s
- **VictoriaMetrics** on `:8428` — scrapes Avalanche every 15s

Run alertreplay against the local stack:

```bash
go run ./cmd/alertreplay \
  --prometheus-url=http://localhost:9090 \
  --from '5 minutes ago' \
  dev/alert.yml \
  AvalancheSeriesAbsent
```

Tear down:

```bash
docker compose down
```
