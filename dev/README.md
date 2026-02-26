## Local Development

Start a local stack with Prometheus, [VictoriaMetrics](https://victoriametrics.com/), [Grafana](https://grafana.com/), and [Avalanche](https://github.com/prometheus-community/avalanche):

```bash
docker compose up -d
```

This starts:
- **Avalanche** on `:9001` — generates synthetic metrics with series churn
- **Prometheus** on `:9090` — scrapes Avalanche every 15s
- **VictoriaMetrics** on `:8428` — scrapes Avalanche every 15s
- **Grafana** on `:3000` — pre-configured with Prometheus datasource

Wait ~2 minutes for metrics to populate, then run alertreplay with any UI type:

### Prometheus UI

```bash
go run ./cmd/alertreplay \
  --prometheus-url=http://localhost:9090 \
  --ui-type prometheus \
  --ui-url http://localhost:9090/graph \
  --from '5 minutes ago' \
  dev/alert.yml \
  AvalancheSeriesAbsent
```

### VMUI (VictoriaMetrics)

```bash
go run ./cmd/alertreplay \
  --prometheus-url=http://localhost:8428 \
  --ui-type vmui \
  --ui-url http://localhost:8428/vmui \
  --from '5 minutes ago' \
  dev/alert.yml \
  AvalancheSeriesAbsent
```

### Grafana Explore

```bash
go run ./cmd/alertreplay \
  --prometheus-url=http://localhost:9090 \
  --ui-type grafana \
  --ui-url 'http://localhost:3000/explore?orgId=1&ds=prometheus' \
  --from '5 minutes ago' \
  dev/alert.yml \
  AvalancheSeriesAbsent
```

Tear down:

```bash
docker compose down
```
