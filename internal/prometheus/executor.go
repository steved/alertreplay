package prometheus

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const (
	defaultQueryTimeout = 2 * time.Minute
	maxPointsPerQuery   = 10000
)

// Executor handles Prometheus query execution.
type Executor struct {
	api          v1.API
	parallelism  int
	queryTimeout time.Duration
}

// NewExecutor creates a new Prometheus query executor.
func NewExecutor(prometheusURL string, parallelism int) (*Executor, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Prometheus client: %w", err)
	}

	return &Executor{
		api:          v1.NewAPI(client),
		parallelism:  parallelism,
		queryTimeout: defaultQueryTimeout,
	}, nil
}

// LabelValues retrieves all values for a given label via Prometheus.
func (e *Executor) LabelValues(ctx context.Context, label string, ts time.Time) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, e.queryTimeout)
	defer cancel()

	query := fmt.Sprintf("clamp_max(count(up) by (%s), 1)", label)
	result, warnings, err := e.api.Query(ctx, query, ts)
	if err != nil {
		return nil, err
	}

	for _, w := range warnings {
		zlog.Warn().Str("warning", w).Msg("label values discovery warning")
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type for label values discovery: %T", result)
	}

	values := make([]string, 0, len(vector))
	for _, sample := range vector {
		value := string(sample.Metric[model.LabelName(label)])
		if value == "" {
			continue
		}
		values = append(values, value)
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("no values found with query %q", query)
	}

	slices.Sort(values)
	values = slices.Compact(values)

	return values, nil
}

// Execute runs a range query and returns vectors indexed by timestamp.
func (e *Executor) Execute(
	ctx context.Context,
	expr string,
	from time.Time,
	to time.Time,
	interval time.Duration,
) (map[int64]promql.Vector, []time.Time, error) {
	from = alignToStep(from, interval)
	to = alignToStep(to, interval)
	timestamps := generateTimestamps(from, to, interval)
	vectors := make(map[int64]promql.Vector, len(timestamps))
	var vectorsMu sync.Mutex

	expected := make(map[int64]struct{}, len(timestamps))
	for _, ts := range timestamps {
		expected[ts.UnixMilli()] = struct{}{}
	}

	windows := splitTimeRange(from, to, interval)
	zlog.Debug().Int("windows", len(windows)).Msg("split time range")

	var eg errgroup.Group
	eg.SetLimit(e.parallelism)

	for i, w := range windows {
		eg.Go(func() error {
			zlog.Debug().
				Str("query", expr).
				Int("window", i+1).
				Int("total", len(windows)).
				Time("from", w.start).
				Time("to", w.end).
				Msg("executing query")

			matrix, err := e.queryRange(ctx, expr, w.start, w.end, interval)
			if err != nil {
				return fmt.Errorf("querying window %d/%d: %w", i+1, len(windows), err)
			}

			e.processMatrix(matrix, expected, vectors, &vectorsMu)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	return vectors, timestamps, nil
}

type timeWindow struct {
	start time.Time
	end   time.Time
}

func splitTimeRange(from, to time.Time, interval time.Duration) []timeWindow {
	maxDuration := time.Duration(maxPointsPerQuery) * interval

	var windows []timeWindow
	for start := from; start.Before(to); {
		end := start.Add(maxDuration)
		if end.After(to) {
			end = to
		}
		windows = append(windows, timeWindow{start: start, end: end})
		start = end.Add(interval)
	}

	if len(windows) == 0 {
		windows = append(windows, timeWindow{start: from, end: to})
	}

	return windows
}

func (e *Executor) queryRange(
	ctx context.Context,
	expr string,
	from, to time.Time,
	interval time.Duration,
) (model.Matrix, error) {
	ctx, cancel := context.WithTimeout(ctx, e.queryTimeout)
	defer cancel()

	result, warnings, err := e.api.QueryRange(ctx, expr, v1.Range{
		Start: from,
		End:   to,
		Step:  interval,
	})
	if err != nil {
		return nil, err
	}

	for _, w := range warnings {
		zlog.Warn().Str("warning", w).Msg("query warning")
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return matrix, nil
}

func (e *Executor) processMatrix(
	matrix model.Matrix,
	expected map[int64]struct{},
	vectors map[int64]promql.Vector,
	mu *sync.Mutex,
) {
	for _, stream := range matrix {
		lb := labels.NewBuilder(labels.EmptyLabels())
		for k, v := range stream.Metric {
			lb.Set(string(k), string(v))
		}
		metricLabels := lb.Labels()

		for _, sample := range stream.Values {
			tsMs := sample.Timestamp.Time().UnixMilli()
			if _, ok := expected[tsMs]; !ok {
				continue
			}

			promSample := promql.Sample{
				T:      tsMs,
				F:      float64(sample.Value),
				Metric: metricLabels,
			}

			mu.Lock()
			vectors[tsMs] = append(vectors[tsMs], promSample)
			mu.Unlock()
		}
	}
}

func generateTimestamps(from time.Time, to time.Time, interval time.Duration) []time.Time {
	n := int((to.Sub(from) / interval) + 1)
	timestamps := make([]time.Time, 0, n)
	for t := from; !t.After(to); t = t.Add(interval) {
		timestamps = append(timestamps, t)
	}

	return timestamps
}

func alignToStep(t time.Time, step time.Duration) time.Time {
	return time.UnixMilli((t.UnixMilli() / int64(step/time.Millisecond)) * int64(step/time.Millisecond))
}
