package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	zlog "github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/steved/alertreplay/internal/alert"
	"github.com/steved/alertreplay/internal/output"
	"github.com/steved/alertreplay/internal/prometheus"
	"github.com/steved/alertreplay/internal/vmrule"
)

type ReplayCmd struct {
	AlertFile string `arg:"" name:"alert-file" help:"Alert rules file (VMRule format)." required:""`
	AlertName string `arg:"" name:"alert-name" help:"Name of the alert to replay." required:""`
}

func (cmd *ReplayCmd) Run(g *Global) error {
	ctx := context.Background()

	r, err := vmrule.ParseAlertRule(cmd.AlertFile, cmd.AlertName)
	if err != nil {
		return fmt.Errorf("parsing alert rule: %w", err)
	}

	zlog.Debug().
		Str("alert", r.Alert).
		Str("expr", r.Expr).
		Dur("for", time.Duration(r.For)).
		Msg("parsed alert rule")

	targets, err := g.Targets(ctx)
	if err != nil {
		return err
	}

	var (
		mu        sync.Mutex
		allAlerts []alert.Alert
	)

	client, err := prometheus.NewAPIClient(g.PrometheusURL, g.Parallelism)
	if err != nil {
		return fmt.Errorf("creating prometheus API client: %w", err)
	}

	var eg errgroup.Group
	for _, target := range targets {
		eg.Go(func() error {
			targetRule := *r
			if target.Label != "" {
				expr, err := prometheus.RewriteExpr(r.Expr, target)
				if err != nil {
					return fmt.Errorf("creating new expr for target %s: %w", target.AppendString(nil), err)
				}

				targetRule.Expr = expr
			}

			alerts, err := alert.Evaluate(ctx, client, targetRule, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("executing alert expr: %w", err)
			}

			mu.Lock()
			allAlerts = append(allAlerts, alerts...)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	alert.Sort(allAlerts)

	return output.PrintEvents(allAlerts)
}
