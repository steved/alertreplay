package main

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/steved/alertreplay/internal/alert"
	"github.com/steved/alertreplay/internal/output"
	"github.com/steved/alertreplay/internal/prometheus"
	"github.com/steved/alertreplay/internal/rule"
)

type DiffCmd struct {
	File1        string   `arg:"" name:"file1" help:"First alert rules file (Prometheus or VMRule format)." required:""`
	File2        string   `arg:"" name:"file2" help:"Second alert rules file (Prometheus or VMRule format)." required:""`
	AlertName    string   `arg:"" name:"alert-name" help:"Name of the alert to compare." required:""`
	IgnoreLabels []string `help:"Labels to ignore when comparing alerts." name:"ignore-labels"`
}

func (cmd *DiffCmd) Run(g *Global) error {
	ctx := context.Background()

	rule1, err := rule.ParseAlertRule(cmd.File1, cmd.AlertName)
	if err != nil {
		return fmt.Errorf("file1 (%s): parsing alert rule: %w", cmd.File1, err)
	}

	rule2, err := rule.ParseAlertRule(cmd.File2, cmd.AlertName)
	if err != nil {
		return fmt.Errorf("file2 (%s): parsing alert rule: %w", cmd.File2, err)
	}

	targets, err := g.Targets(ctx)
	if err != nil {
		return err
	}

	var (
		mu      sync.Mutex
		alerts1 []alert.Row
		alerts2 []alert.Row
	)

	var eg errgroup.Group
	for _, target := range targets {
		target := target
		eg.Go(func() error {
			exec, err := prometheus.NewExecutor(g.PrometheusURL, g.Parallelism)
			if err != nil {
				return fmt.Errorf("target %s: creating executor: %w", target, err)
			}

			targetRule1 := *rule1
			if target.Label != "" {
				expr, err := prometheus.RewriteExpr(rule1.Expr, target)
				if err != nil {
					return fmt.Errorf("target %s: creating new expr: %w", target, err)
				}

				targetRule1.Expr = expr
			}

			targetRule2 := *rule2
			if target.Label != "" {
				expr, err := prometheus.RewriteExpr(rule2.Expr, target)
				if err != nil {
					return fmt.Errorf("target %s: creating new expr: %w", target, err)
				}

				targetRule2.Expr = expr
			}

			targetAlerts1, err := alert.Run(ctx, exec, &targetRule1, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("target %s file1 (%s): %w", target, cmd.File1, err)
			}

			targetAlerts2, err := alert.Run(ctx, exec, &targetRule2, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("target %s file2 (%s): %w", target, cmd.File2, err)
			}

			// TODO...
			targetAlerts1 = alert.AddLabel(targetAlerts1, target)
			targetAlerts2 = alert.AddLabel(targetAlerts2, target)

			mu.Lock()
			alerts1 = append(alerts1, targetAlerts1...)
			alerts2 = append(alerts2, targetAlerts2...)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	alert.SortRows(alerts1)
	alert.SortRows(alerts2)

	for _, label := range cmd.IgnoreLabels {
		for i := range alerts1 {
			delete(alerts1[i].Labels, label)
		}

		for i := range alerts2 {
			delete(alerts2[i].Labels, label)
		}
	}

	return printDiffResults(cmd.File1, cmd.File2, alerts1, alerts2)
}

const alertMatchThreshold = 2 * time.Minute

func alertsMatch(a, b alert.Row) bool {
	if !reflect.DeepEqual(a.Labels, b.Labels) {
		return false
	}
	diff := a.OpenedAt.Sub(b.OpenedAt)
	if diff < 0 {
		diff = -diff
	}
	return diff <= alertMatchThreshold
}

func findMatchingAlert(ar alert.Row, alerts []alert.Row, used map[int]bool) int {
	for i, candidate := range alerts {
		if used[i] {
			continue
		}
		if alertsMatch(ar, candidate) {
			return i
		}
	}
	return -1
}

func printDiffResults(file1, file2 string, alerts1, alerts2 []alert.Row) error {
	var (
		baseFile1 = filepath.Base(file1)
		baseFile2 = filepath.Base(file2)

		alerts []alert.Row
		used2  = make(map[int]bool)
	)

	for _, ar := range alerts1 {
		idx := findMatchingAlert(ar, alerts2, used2)

		if idx == -1 {
			ar.Source = baseFile1
		} else {
			used2[idx] = true
		}

		alerts = append(alerts, ar)
	}

	for i, ar := range alerts2 {
		if !used2[i] {
			ar.Source = baseFile2
			alerts = append(alerts, ar)
		}
	}

	slices.SortFunc(alerts, func(a, b alert.Row) int {
		return a.OpenedAt.Compare(b.OpenedAt)
	})

	return output.PrintEvents(alerts)
}
