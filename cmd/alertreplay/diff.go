package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/steved/alertreplay/internal/alert"
	"github.com/steved/alertreplay/internal/output"
	"github.com/steved/alertreplay/internal/prometheus"
	"github.com/steved/alertreplay/internal/vmrule"
)

type DiffCmd struct {
	File1        string   `arg:"" name:"file1" help:"First alert rules file (VMRule format)." required:""`
	File2        string   `arg:"" name:"file2" help:"Second alert rules file (VMRule format)." required:""`
	AlertName    string   `arg:"" name:"alert-name" help:"Name of the alert to compare." required:""`
	IgnoreLabels []string `help:"Labels to ignore when comparing alerts." name:"ignore-labels"`
}

func (cmd *DiffCmd) Run(g *Global) error {
	ctx := context.Background()

	rule1, err := vmrule.ParseAlertRule(cmd.File1, cmd.AlertName)
	if err != nil {
		return fmt.Errorf("file1 (%s): parsing alert rule: %w", cmd.File1, err)
	}

	rule2, err := vmrule.ParseAlertRule(cmd.File2, cmd.AlertName)
	if err != nil {
		return fmt.Errorf("file2 (%s): parsing alert rule: %w", cmd.File2, err)
	}

	targets, err := g.Targets(ctx)
	if err != nil {
		return err
	}

	var (
		mu      sync.Mutex
		alerts1 []alert.Alert
		alerts2 []alert.Alert
	)

	client, err := prometheus.NewAPIClient(g.PrometheusURL, g.Parallelism)
	if err != nil {
		return fmt.Errorf("creating prometheus API client: %w", err)
	}

	var eg errgroup.Group
	for _, target := range targets {
		eg.Go(func() error {
			r := *rule1

			if target.Label != "" {
				expr, err := prometheus.RewriteExpr(r.Expr, target)
				if err != nil {
					return fmt.Errorf("creating new expr for target %s: %w", target.AppendString(nil), err)
				}

				r.Expr = expr
			}

			alerts, err := alert.Evaluate(ctx, client, r, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("executing alert expr for file1 (%s): %w", cmd.File1, err)
			}

			for _, label := range cmd.IgnoreLabels {
				for i := range alerts {
					delete(alerts[i].Labels, label)
				}
			}

			mu.Lock()
			defer mu.Unlock()
			alerts1 = append(alerts1, alerts...)

			return nil
		})

		eg.Go(func() error {
			r := *rule2

			if target.Label != "" {
				expr, err := prometheus.RewriteExpr(r.Expr, target)
				if err != nil {
					return fmt.Errorf("creating new expr for target %s: %w", target.AppendString(nil), err)
				}

				r.Expr = expr
			}

			alerts, err := alert.Evaluate(ctx, client, r, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("executing alert expr for file2 (%s): %w", cmd.File2, err)
			}

			for _, label := range cmd.IgnoreLabels {
				for i := range alerts {
					delete(alerts[i].Labels, label)
				}
			}

			mu.Lock()
			defer mu.Unlock()
			alerts2 = append(alerts2, alerts...)

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	alert.Sort(alerts1)
	alert.Sort(alerts2)

	return printDiffResults(cmd.File1, cmd.File2, alerts1, alerts2)
}

func findMatchingAlert(ar alert.Alert, alerts []alert.Alert, used map[int]bool) int {
	for i, candidate := range alerts {
		if used[i] {
			continue
		}
		if ar.Match(candidate) {
			return i
		}
	}
	return -1
}

func printDiffResults(file1, file2 string, alerts1, alerts2 []alert.Alert) error {
	var (
		baseFile1 = filepath.Base(file1)
		baseFile2 = filepath.Base(file2)

		alerts []alert.Alert
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

	alert.Sort(alerts)

	return output.PrintEvents(alerts)
}
