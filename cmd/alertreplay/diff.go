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

	targets, err := g.clusterTargets(ctx)
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
			exec, err := prometheus.NewExecutor(target.promURL, g.Parallelism)
			if err != nil {
				return fmt.Errorf("cluster %s: creating executor: %w", target.nameForLog(), err)
			}

			clusterAlerts1, err := alert.Run(ctx, exec, rule1, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("cluster %s file1 (%s): %w", target.nameForLog(), cmd.File1, err)
			}

			clusterAlerts2, err := alert.Run(ctx, exec, rule2, g.From, g.To, g.Interval)
			if err != nil {
				return fmt.Errorf("cluster %s file2 (%s): %w", target.nameForLog(), cmd.File2, err)
			}

			clusterAlerts1 = alert.AddClusterLabel(clusterAlerts1, target.name)
			clusterAlerts2 = alert.AddClusterLabel(clusterAlerts2, target.name)

			mu.Lock()
			alerts1 = append(alerts1, clusterAlerts1...)
			alerts2 = append(alerts2, clusterAlerts2...)
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
