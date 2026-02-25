# alertreplay

A CLI that replays alerting rules over historical Prometheus/VictoriaMetrics data.

## Usage

```bash
alertreplayer \
  --from '2025-12-01 00:00:00' \
  --by-cluster \                # optional: discover available clusters and run each in parallel
  /path/to/alerts.yaml \
  MyAlertName
```

- `--prometheus-url`: Prometheus or VMSelect URL.
- `--from` / `--to` (required): Time range in `YYYY-MM-DD HH:MM:SS` (UTC).
- `--interval`: Step size between evaluations (default 30s).
- `--parallelism`: Prometheus query parallelism (default 10).
- `--cluster`: Optional cluster filter.
- `--by-cluster`: Replay the alert per all discoverable clusters. Mutually exclusive with `--cluster`.
- Positional args: the alert rules file and the alert name to replay.

The tool opens an interactive table showing open/resolved windows and lets you jump to Grafana Explore for the selected alert row.

### Compare two rule files

Use `diff` to replay the same alert across two files and compare:

```bash
alertreplayer diff \
  --from '2025-12-01 00:00:00' \
  /path/to/alerts_old.yaml \
  /path/to/alerts_new.yaml \
  MyAlertName
```

- `--ignore-labels` can be repeated; labels listed there are ignored when matching alerts between files.
- Non-interactive output (when stdout is not a TTY) is rendered as markdown for easy copying.
