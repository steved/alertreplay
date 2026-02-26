package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/steved/alertreplay/internal/relativetime"
)

type Global struct {
	PrometheusURL string            `help:"Prometheus API URL."`
	From          time.Time         `help:"Start time: 'YYYY-MM-DD HH:MM:SS' or relative like '30 days ago'." required:"" placeholder:"time" type:"relativetime"`
	To            time.Time         `help:"End time: 'YYYY-MM-DD HH:MM:SS' or relative like 'now'." default:"now" placeholder:"time" type:"relativetime"`
	Interval      time.Duration     `help:"Query interval." default:"30s"`
	Parallelism   int               `help:"Number of parallel queries." default:"10"`
	Filters       map[string]string `help:"Append filters via VictoriaMetrics extra_filters[]."`
	By            string            `help:"Discover filter values via Prometheus and run the alert once per value."`
	Verbose       VerboseFlag       `help:"Enable debug logging." short:"v"`
}

type VerboseFlag bool

func (v VerboseFlag) BeforeApply() error {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	return nil
}

func (g *Global) Validate() error {
	if g.By != "" && g.Filters[g.By] != "" {
		return fmt.Errorf("--by and --filters for the same key are mutually exclusive")
	}

	if g.Parallelism < 1 {
		return fmt.Errorf("--parallelism must be at least 1")
	}

	if !g.From.Before(g.To) {
		return fmt.Errorf("--from must be before --to")
	}

	return nil
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

func (g *Global) clusterTargets(ctx context.Context) ([]clusterTarget, error) {
	return []clusterTarget{{promURL: g.PrometheusURL}}, nil
}

// 	if c.ByCluster {
// 		exec, err := prometheus.NewExecutor(c.PrometheusURL, c.Parallelism)
// 		if err != nil {
// 			return nil, fmt.Errorf("creating Prometheus client: %w", err)
// 		}
//
// 		clusters, err := exec.ListClusters(ctx, toTime)
// 		if err != nil {
// 			return nil, fmt.Errorf("discovering clusters: %w", err)
// 		}
//
// 		zlog.Debug().Strs("clusters", clusters).Msg("running alert per cluster")
//
// 		targets := make([]clusterTarget, 0, len(clusters))
// 		for _, cluster := range clusters {
// 			promURL, err := appendClusterFilter(c.PrometheusURL, cluster)
// 			if err != nil {
// 				return nil, fmt.Errorf("building cluster URL for %q: %w", cluster, err)
// 			}
//
// 			targets = append(targets, clusterTarget{
// 				name:    cluster,
// 				promURL: promURL,
// 			})
// 		}
//
// 		return targets, nil
// 	}
//
// 	if c.Cluster != "" {
// 		promURL, err := appendClusterFilter(c.PrometheusURL, c.Cluster)
// 		if err != nil {
// 			return nil, fmt.Errorf("adding cluster filter: %w", err)
// 		}
//
// 		return []clusterTarget{{
// 			name:    c.Cluster,
// 			promURL: promURL,
// 		}}, nil
// 	}
//
// 	return []clusterTarget{{
// 		promURL: c.PrometheusURL,
// 	}}, nil
// }

type CLI struct {
	Global

	Replay ReplayCmd `cmd:"" help:"Replay an alert rule against historical data." default:"withargs"`
	Diff   DiffCmd   `cmd:"" help:"Compare an alert rule between two files."`
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	root := CLI{}
	ctx := kong.Parse(&root,
		kong.Name("alertreplay"),
		kong.Description("Replay or compare alert rules against historical data."),
		kong.UsageOnError(),
		kong.TypeMapper(reflect.TypeFor[time.Time](), relativetime.Mapper),
	)

	ctx.FatalIfErrorf(ctx.Run(&root.Global))
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
