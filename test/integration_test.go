//go:build integration

package test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gotest.tools/v3/golden"

	"github.com/steved/alertreplay/internal/alert"
	"github.com/steved/alertreplay/internal/dashboard"
	"github.com/steved/alertreplay/internal/output"
	promclient "github.com/steved/alertreplay/internal/prometheus"
	"github.com/steved/alertreplay/internal/vmrule"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	const (
		prometheusScrapeInterval = 5 * time.Second
		evaluationInterval       = 15 * time.Second
		downScrapesRequired      = 6
		recoveryScrapesRequired  = 3
		pollTimeout              = 2 * time.Minute
	)

	reg, promEndpoint := setupPrometheus(t)

	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_service_up",
		Help: "Test service health.",
	}, []string{"service", "instance"})
	reg.MustRegister(gauge)
	gauge.WithLabelValues("abc", "host1").Set(1)

	apiClient, err := api.NewClient(api.Config{Address: promEndpoint})
	require.NoError(t, err)
	promAPI := v1.NewAPI(apiClient)

	pollCtx, pollCancel := context.WithTimeout(t.Context(), pollTimeout)
	defer pollCancel()
	waitForData(t, pollCtx, promAPI, `test_service_up{service="abc"}`)

	rule, err := vmrule.ParseAlertRule("testdata/test_alert.yml", "TestServiceDown")
	require.NoError(t, err)

	client, err := promclient.NewAPIClient(promEndpoint, 10)
	require.NoError(t, err)

	urlBuilder, err := dashboard.New(dashboard.VMUI, "https://vmui/")
	require.NoError(t, err)

	evalFrom := time.Now().UTC().Add(-15 * time.Second)

	gauge.WithLabelValues("abc", "host1").Set(0)
	waitForScrapedSamples(
		t,
		pollCtx,
		promAPI,
		`test_service_up{service="abc"}`,
		0,
		downScrapesRequired,
		prometheusScrapeInterval,
	)

	gauge.WithLabelValues("abc", "host1").Set(1)
	waitForScrapedSamples(
		t,
		pollCtx,
		promAPI,
		`test_service_up{service="abc"}`,
		1,
		recoveryScrapesRequired,
		prometheusScrapeInterval,
	)

	evalTo := waitForResolvedAlert(
		t,
		pollCtx,
		client,
		*rule,
		evalFrom,
		evaluationInterval,
		urlBuilder,
	)

	alerts, err := alert.Evaluate(t.Context(), client, *rule, evalFrom, evalTo, evaluationInterval, urlBuilder)
	require.NoError(t, err)

	freezeAlerts(alerts, rule.Expr, urlBuilder)

	var buf bytes.Buffer
	err = output.RenderMarkdown(&buf, alerts)
	require.NoError(t, err)

	golden.Assert(t, buf.String(), "output.md")
}

func setupPrometheus(t *testing.T) (*prometheus.Registry, string) {
	t.Helper()

	reg := prometheus.NewRegistry()
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	srv := &http.Server{Handler: mux}

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	t.Logf("metrics server listening on port %d", port)

	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	promConfig := fmt.Sprintf(`global:
  scrape_interval: 5s
  evaluation_interval: 5s
scrape_configs:
  - job_name: test
    static_configs:
      - targets:
          - host.testcontainers.internal:%d
`, port)

	promContainer, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:           "prom/prometheus:v3",
			ExposedPorts:    []string{"9090/tcp"},
			HostAccessPorts: []int{port},
			Files: []testcontainers.ContainerFile{
				{
					Reader:            bytes.NewReader([]byte(promConfig)),
					ContainerFilePath: "/etc/prometheus/prometheus.yml",
					FileMode:          0o644,
				},
			},
			WaitingFor: wait.ForHTTP("/-/ready").WithPort("9090/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = promContainer.Terminate(context.Background()) })

	promEndpoint, err := promContainer.Endpoint(t.Context(), "http")
	require.NoError(t, err)
	t.Logf("prometheus endpoint: %s", promEndpoint)

	return reg, promEndpoint
}

var (
	frozenOpened   = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	frozenResolved = time.Date(2000, 1, 1, 0, 0, 30, 0, time.UTC)
)

func freezeAlerts(alerts []alert.Alert, expr string, urlBuilder dashboard.URLBuilder) {
	for i := range alerts {
		alert := &alerts[i]
		delete(alert.Labels, "instance")

		alert.OpenedAt = frozenOpened
		if alert.ResolvedAt != nil {
			alert.ResolvedAt = new(frozenResolved)
		}

		if urlBuilder != nil {
			alert.URL = urlBuilder.BuildURL(expr, alert.OpenedAt, alert.ResolvedAt)
		}
	}
}

func waitForData(t *testing.T, ctx context.Context, api v1.API, query string) {
	t.Helper()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for data in Prometheus")
		case <-ticker.C:
			result, _, err := api.Query(ctx, query, time.Now())
			if err != nil {
				t.Logf("poll query error (retrying): %v", err)
				continue
			}
			if vec, ok := result.(model.Vector); ok && vec.Len() > 0 {
				t.Logf("data available: %s", vec.String())
				return
			}
		}
	}
}

func waitForScrapedSamples(
	t *testing.T,
	ctx context.Context,
	api v1.API,
	query string,
	expected float64,
	required int,
	step time.Duration,
) {
	t.Helper()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf(
				"timed out waiting for query %q to observe %.0f in at least %d samples",
				query,
				expected,
				required,
			)
		case <-ticker.C:
			end := time.Now().UTC()
			start := end.Add(-time.Duration(required) * step)

			result, _, err := api.QueryRange(ctx, query, v1.Range{
				Start: start,
				End:   end,
				Step:  step,
			})
			if err != nil {
				t.Logf("poll query error (retrying): %v", err)
				continue
			}

			matrix, ok := result.(model.Matrix)
			if !ok || len(matrix) == 0 {
				t.Logf("query result unavailable yet: %q returned %T", query, result)
				continue
			}

			maxMatches := 0
			for _, stream := range matrix {
				matches := 0
				for _, sample := range stream.Values {
					if float64(sample.Value) == expected {
						matches++
					}
				}
				if matches > maxMatches {
					maxMatches = matches
				}
			}

			if maxMatches >= required {
				t.Logf(
					"query threshold reached: %q observed %.0f in %d samples",
					query,
					expected,
					maxMatches,
				)
				return
			}
		}
	}
}

func waitForResolvedAlert(
	t *testing.T,
	ctx context.Context,
	client promclient.Client,
	rule rulefmt.Rule,
	from time.Time,
	interval time.Duration,
	urlBuilder dashboard.URLBuilder,
) time.Time {
	t.Helper()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for a resolved alert in evaluation")
		case <-ticker.C:
			to := time.Now().UTC()
			alerts, err := alert.Evaluate(ctx, client, rule, from, to, interval, urlBuilder)
			if err != nil {
				t.Logf("evaluation poll error (retrying): %v", err)
				continue
			}

			for _, alert := range alerts {
				if alert.ResolvedAt != nil {
					t.Logf("resolved alert observed in evaluation at %s", to.Format(time.RFC3339))
					return to
				}
			}
		}
	}
}
