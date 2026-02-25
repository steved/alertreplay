package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/steved/alertreplay/internal/prometheus"
)

const timeFormat = "2006-01-02 15:04:05"

type commonFlags struct {
	PrometheusURL string        `help:"Prometheus API URL." default:"http://vmselect-multi-cluster:8481/select/13/prometheus"`
	From          string        `help:"Start time: 'YYYY-MM-DD HH:MM:SS' or relative like '30 days ago'." required:"" placeholder:"time"`
	To            string        `help:"End time: 'YYYY-MM-DD HH:MM:SS' or relative like 'now'." default:"now" placeholder:"time"`
	Interval      time.Duration `help:"Query interval." default:"30s"`
	Parallelism   int           `help:"Number of parallel queries." default:"10"`
	Cluster       string        `help:"Append cluster filter via VictoriaMetrics extra_filters[]."`
	ByCluster     bool          `help:"Discover clusters via Prometheus and run the alert per cluster (mutually exclusive with --cluster)."`
	Verbose       bool          `help:"Enable debug logging." short:"v"`
}

func (c *commonFlags) configureLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if c.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func (c *commonFlags) timeRange(now time.Time) (time.Time, time.Time, error) {
	if c.Cluster != "" && c.ByCluster {
		return time.Time{}, time.Time{}, fmt.Errorf("--cluster and --by-cluster are mutually exclusive")
	}

	fromTime, err := parseTime(c.From, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --from time: %w", err)
	}

	toTime, err := parseTime(c.To, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --to time: %w", err)
	}

	if !fromTime.Before(toTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("--from must be before --to")
	}

	if c.Parallelism < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("--parallelism must be at least 1")
	}

	return fromTime, toTime, nil
}

type clusterTarget struct {
	name    string
	promURL string
}

func (ct clusterTarget) nameForLog() string {
	if ct.name != "" {
		return ct.name
	}
	return "all-clusters"
}

func (c *commonFlags) clusterTargets(ctx context.Context, toTime time.Time) ([]clusterTarget, error) {
	if c.ByCluster {
		exec, err := prometheus.NewExecutor(c.PrometheusURL, c.Parallelism)
		if err != nil {
			return nil, fmt.Errorf("creating Prometheus client: %w", err)
		}

		clusters, err := exec.ListClusters(ctx, toTime)
		if err != nil {
			return nil, fmt.Errorf("discovering clusters: %w", err)
		}

		zlog.Debug().Strs("clusters", clusters).Msg("running alert per cluster")

		targets := make([]clusterTarget, 0, len(clusters))
		for _, cluster := range clusters {
			promURL, err := appendClusterFilter(c.PrometheusURL, cluster)
			if err != nil {
				return nil, fmt.Errorf("building cluster URL for %q: %w", cluster, err)
			}

			targets = append(targets, clusterTarget{
				name:    cluster,
				promURL: promURL,
			})
		}

		return targets, nil
	}

	if c.Cluster != "" {
		promURL, err := appendClusterFilter(c.PrometheusURL, c.Cluster)
		if err != nil {
			return nil, fmt.Errorf("adding cluster filter: %w", err)
		}

		return []clusterTarget{{
			name:    c.Cluster,
			promURL: promURL,
		}}, nil
	}

	return []clusterTarget{{
		promURL: c.PrometheusURL,
	}}, nil
}

type cli struct {
	Backtest backtestCmd `cmd:"" help:"Backtest an alert rule against historical data." default:"withargs"`
	Diff     diffCmd     `cmd:"" help:"Compare an alert rule between two files."`
}

func main() {
	root := cli{}
	ctx := kong.Parse(&root,
		kong.Name("alert-backtester"),
		kong.Description("Backtest alert rules or compare rule changes against historical data."),
		kong.UsageOnError(),
	)

	ctx.FatalIfErrorf(ctx.Run(&root))
}

func appendClusterFilter(prometheusURL string, cluster string) (string, error) {
	u, err := url.Parse(prometheusURL)
	if err != nil {
		return "", fmt.Errorf("parsing Prometheus URL: %w", err)
	}

	q := u.Query()
	q.Add("extra_filters[]", fmt.Sprintf(`{cluster="%s"}`, cluster))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

var relativeTimeRE = regexp.MustCompile(`(?i)^(\d+)\s*(second|minute|hour|day|week|month|year)s?\s+ago$`)

func parseTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)

	if strings.EqualFold(s, "now") {
		return now, nil
	}

	if matches := relativeTimeRE.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := strings.ToLower(matches[2])

		var d time.Duration
		switch unit {
		case "second":
			d = time.Duration(n) * time.Second
		case "minute":
			d = time.Duration(n) * time.Minute
		case "hour":
			d = time.Duration(n) * time.Hour
		case "day":
			d = time.Duration(n) * 24 * time.Hour
		case "week":
			d = time.Duration(n) * 7 * 24 * time.Hour
		case "month":
			return now.AddDate(0, -n, 0), nil
		case "year":
			return now.AddDate(-n, 0, 0), nil
		}
		return now.Add(-d), nil
	}

	return time.ParseInLocation(timeFormat, s, time.UTC)
}
