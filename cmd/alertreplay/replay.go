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
	"github.com/steved/alertreplay/internal/rule"
)

type ReplayCmd struct {
	AlertFile string `arg:"" name:"alert-file" help:"Alert rules file (Prometheus or VMRule format)." required:""`
	AlertName string `arg:"" name:"alert-name" help:"Name of the alert to replay." required:""`
}

func (cmd *ReplayCmd) Run(g *Global) error {
	ctx := context.Background()

	r, err := rule.ParseAlertRule(cmd.AlertFile, cmd.AlertName)
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
		allAlerts []alert.Row
	)

	var eg errgroup.Group
	for _, target := range targets {
		eg.Go(func() error {
			exec, err := prometheus.NewExecutor(g.PrometheusURL, g.Parallelism)
			if err != nil {
				return fmt.Errorf("target %s: creating executor: %w", target, err)
			}

			targetRule := *r
			if target.Label != "" {
				expr, err := prometheus.RewriteExpr(r.Expr, target)
				if err != nil {
					return fmt.Errorf("target %s: creating new expr: %w", target, err)
				}

				targetRule.Expr = expr
			}

			alerts, err := alert.Run(ctx, exec, &targetRule, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("target %s: %w", target, err)
			}

			alerts = alert.AddLabel(alerts, target)

			mu.Lock()
			allAlerts = append(allAlerts, alerts...)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	alert.SortRows(allAlerts)

	return output.PrintEvents(allAlerts)
}
