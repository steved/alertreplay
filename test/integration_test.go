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
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gotest.tools/v3/golden"

	"github.com/steved/alertreplay/internal/alert"
	"github.com/steved/alertreplay/internal/output"
	promclient "github.com/steved/alertreplay/internal/prometheus"
	"github.com/steved/alertreplay/internal/vmrule"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	reg := prometheus.NewRegistry()
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "test_service_up",
		Help: "Test service health.",
	}, []string{"service", "instance"})
	reg.MustRegister(gauge)
	gauge.WithLabelValues("abc", "host1").Set(1)

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

	apiClient, err := api.NewClient(api.Config{Address: promEndpoint})
	require.NoError(t, err)
	promAPI := v1.NewAPI(apiClient)

	pollCtx, pollCancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer pollCancel()
	waitForData(t, pollCtx, promAPI, `test_service_up{service="abc"}`)

	evalFrom := time.Now().UTC().Add(-15 * time.Second)

	gauge.WithLabelValues("abc", "host1").Set(0)
	t.Log("set gauge to 0, waiting 30s for scrapes...")
	time.Sleep(30 * time.Second)

	gauge.WithLabelValues("abc", "host1").Set(1)
	t.Log("set gauge to 1, waiting 15s for recovery scrapes...")
	time.Sleep(15 * time.Second)

	evalTo := time.Now().UTC()

	rule, err := vmrule.ParseAlertRule("testdata/test_alert.yml", "TestServiceDown")
	require.NoError(t, err)

	client, err := promclient.NewAPIClient(promEndpoint, 10)
	require.NoError(t, err)

	alerts, err := alert.Evaluate(t.Context(), client, *rule, evalFrom, evalTo, 15*time.Second)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = output.RenderMarkdown(&buf, alerts)
	require.NoError(t, err)

	golden.Assert(t, buf.String(), "output.md")
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
			if result.String() != "" && result.String() != "[]" {
				t.Logf("data available: %s", result.String())
				return
			}
		}
	}
}
