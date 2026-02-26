# alertreplay

A CLI that replays alerting rules over historical Prometheus/VictoriaMetrics data.

When stdout is a TTY, results are displayed in an interactive table that lets you press Enter to open the dashboard URL for any alert row. When piped or redirected, output is rendered as markdown.

## Installation

```bash
go install github.com/steved/alertreplay/cmd/alertreplay@latest
```

## Usage

### Replay

Replay an alert rule against historical data:

```bash
alertreplay \
  --prometheus-url http://localhost:9090 \
  --from '2025-12-01 00:00:00' \
  --to '2025-12-15 00:00:00' \
  /path/to/alerts.yaml \
  MyAlertName
```

Discover values for a label and replay each in parallel:

```bash
alertreplay \
  --prometheus-url http://localhost:9090 \
  --from '30 days ago' \
  --by cluster \
  /path/to/alerts.yaml \
  MyAlertName
```

With a dashboard URL so each alert row links to Prometheus:

```bash
alertreplay \
  --prometheus-url http://localhost:9090 \
  --from '7 days ago' \
  --ui-type prometheus \
  --ui-url http://localhost:9090/graph \
  /path/to/alerts.yaml \
  MyAlertName
```

### Diff

Compare the same alert across two rule files:

```bash
alertreplay diff \
  --prometheus-url http://localhost:9090 \
  --from '2025-12-01 00:00:00' \
  --ignore-labels instance \
  /path/to/alerts_old.yaml \
  /path/to/alerts_new.yaml \
  MyAlertName
```

### Global flags

| Flag | Description | Default |
|---|---|---|
| `--prometheus-url` | Prometheus or VMSelect API URL. | |
| `--from` | Start time: `YYYY-MM-DD HH:MM:SS` (UTC) or relative like `30 days ago`. | *required* |
| `--to` | End time: same format as `--from`. | `now` |
| `--interval` | Step size between evaluations. | `30s` |
| `--parallelism` | Number of parallel Prometheus queries. | `10` |
| `--filters` | Append label filters to alert expressions (e.g. `--filters cluster=us-east`). | |
| `--by` | Discover filter values via query and run the alert once per value. Mutually exclusive with `--filters`. | |
| `--ui-url` | Base URL for the dashboard UI (e.g. `http://localhost:9090/graph`). | |
| `--ui-type` | Dashboard UI type: `prometheus`, `vmui`, or `grafana`. | `prometheus` |
| `-v` | Enable debug logging. | |

### Dashboard UI types

The `--ui-url` and `--ui-type` flags control the clickable URL generated for each alert.

- **prometheus** -- Standard Prometheus graph UI. Pass the base graph URL (e.g. `http://localhost:9090/graph`).
- **vmui** -- VictoriaMetrics UI. Pass the vmui URL (e.g. `http://victoriametrics:8428/vmui`). Existing query params like `?tenant=0` are preserved.
- **grafana** -- Grafana Explore. Pass the full explore URL including datasource params (e.g. `http://grafana:3000/explore?orgId=1&ds=production`).

### Diff flags

| Flag | Description |
|---|---|
| `--ignore-labels` | Labels to ignore when matching alerts between files. Can be repeated. |

## Development

### Prerequisites

- Go 1.26+
- [mise](https://mise.jdx.dev/) (optional, for task runner)

### Building

```bash
go build ./cmd/alertreplay
```

### Running tests

```bash
go test ./...
```

Or with mise (integration tests require Docker):

```bash
mise run test
```

### Linting and formatting

```bash
mise run lint
mise run fmt
```

## Contributing

1. Fork the repository.
2. Create a feature branch from `main`.
3. Make your changes and add tests.
4. Ensure `go test ./...` and `mise run lint` pass.
5. Open a pull request.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
