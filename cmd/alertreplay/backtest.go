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

type backtestCmd struct {
	commonFlags

	AlertFile string `arg:"" name:"alert-file" help:"Alert rules file (Prometheus or VMRule format)." required:""`
	AlertName string `arg:"" name:"alert-name" help:"Name of the alert to replay." required:""`
}

func (cmd *backtestCmd) Run() error {
	cmd.configureLogger()

	ctx := context.Background()

	now := time.Now()
	fromTime, toTime, err := cmd.timeRange(now)
	if err != nil {
		return err
	}

	r, err := rule.ParseAlertRule(cmd.AlertFile, cmd.AlertName)
	if err != nil {
		return fmt.Errorf("parsing alert rule: %w", err)
	}
	zlog.Debug().
		Str("alert", r.Alert).
		Str("expr", r.Expr).
		Dur("for", time.Duration(r.For)).
		Msg("parsed alert rule")

	targets, err := cmd.clusterTargets(ctx, toTime)
	if err != nil {
		return err
	}

	var (
		mu        sync.Mutex
		allAlerts []alert.Row
	)

	var eg errgroup.Group
	for _, target := range targets {
		target := target
		eg.Go(func() error {
			exec, err := prometheus.NewExecutor(target.promURL, cmd.Parallelism)
			if err != nil {
				return fmt.Errorf("cluster %s: creating executor: %w", target.nameForLog(), err)
			}

			alerts, err := alert.Run(ctx, exec, r, fromTime, toTime, cmd.Interval)
			if err != nil {
				return fmt.Errorf("cluster %s: %w", target.nameForLog(), err)
			}

			alerts = alert.AddClusterLabel(alerts, target.name)

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
