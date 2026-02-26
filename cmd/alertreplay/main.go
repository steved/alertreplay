package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/steved/alertreplay/internal/prometheus"
	"github.com/steved/alertreplay/internal/relativetime"
)

type Global struct {
	PrometheusURL string                  `help:"Prometheus API URL."`
	From          time.Time               `help:"Start time: 'YYYY-MM-DD HH:MM:SS' or relative like '30 days ago'." required:"" placeholder:"time"`
	To            time.Time               `help:"End time: 'YYYY-MM-DD HH:MM:SS' or relative like 'now'." default:"now" placeholder:"time"`
	Interval      time.Duration           `help:"Query interval." default:"30s"`
	Parallelism   int                     `help:"Number of parallel queries." default:"10"`
	Filters       []metricsql.LabelFilter `help:"Append filters to alert expressions."`
	By            string                  `help:"Discover filter values via Prometheus and run the alert once per value."`
	Verbose       VerboseFlag             `help:"Enable debug logging." short:"v"`
}

type VerboseFlag bool

func (v VerboseFlag) BeforeApply() error {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	return nil
}

func (g *Global) Validate() error {
	if g.By != "" && len(g.Filters) > 0 {
		return fmt.Errorf("--by and --filters are mutually exclusive")
	}

	if g.Parallelism < 1 {
		return fmt.Errorf("--parallelism must be at least 1")
	}

	if !g.From.Before(g.To) {
		return fmt.Errorf("--from must be before --to")
	}

	return nil
}

func (g *Global) Targets(ctx context.Context) ([]metricsql.LabelFilter, error) {
	if g.By != "" {
		exec, err := prometheus.NewExecutor(g.PrometheusURL, g.Parallelism)
		if err != nil {
			return nil, fmt.Errorf("creating Prometheus client: %w", err)
		}

		values, err := exec.LabelValues(ctx, g.By, g.To)
		if err != nil {
			return nil, fmt.Errorf("discovering label values: %w", err)
		}

		zlog.Debug().Strs("targets", values).Msg("running alert per label value")

		targets := make([]metricsql.LabelFilter, 0, len(values))
		for _, value := range values {
			targets = append(targets, metricsql.LabelFilter{
				Label: g.By,
				Value: value,
			})
		}

		return targets, nil
	}

	if len(g.Filters) > 0 {
		zlog.Debug().Interface("targets", g.Filters).Msg("running alert per filter")
		return g.Filters, nil
	}

	return []metricsql.LabelFilter{{}}, nil
}

type CLI struct {
	Global

	Replay ReplayCmd `cmd:"" help:"Replay an alert rule against historical data." default:"withargs"`
	Diff   DiffCmd   `cmd:"" help:"Compare an alert rule between two files."`
}

func main() {
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	root := CLI{}
	ctx := kong.Parse(&root,
		kong.Name("alertreplay"),
		kong.Description("Replay or compare alert rules against historical data."),
		kong.UsageOnError(),
		kong.TypeMapper(reflect.TypeFor[time.Time](), relativetime.Mapper),
		kong.TypeMapper(reflect.TypeFor[metricsql.LabelFilter](), prometheus.LabelFilterMapper),
	)

	ctx.FatalIfErrorf(ctx.Run(&root.Global))
}
