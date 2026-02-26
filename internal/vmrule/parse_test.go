package vmrule

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validVMRule = `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMRule
metadata:
  name: test-rules
spec:
  groups:
    - name: test-group
      rules:
        - alert: HighLatency
          expr: histogram_quantile(0.99, rate(http_duration_seconds_bucket[5m])) > 1
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "High latency detected"
        - alert: LowAvailability
          expr: up == 0
          labels:
            severity: warning
    - name: second-group
      rules:
        - alert: DiskFull
          expr: node_filesystem_avail_bytes / node_filesystem_size_bytes < 0.1
          for: 10m
`

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestParseAlertRule(t *testing.T) {
	for _, tt := range []struct {
		name      string
		content   string
		alertName string
		wantAlert string
		wantExpr  string
		wantErr   string
	}{
		{
			name:      "finds alert in first group",
			content:   validVMRule,
			alertName: "HighLatency",
			wantAlert: "HighLatency",
			wantExpr:  "histogram_quantile(0.99, rate(http_duration_seconds_bucket[5m])) > 1",
		},
		{
			name:      "finds alert in second group",
			content:   validVMRule,
			alertName: "DiskFull",
			wantAlert: "DiskFull",
			wantExpr:  "node_filesystem_avail_bytes / node_filesystem_size_bytes < 0.1",
		},
		{
			name:      "alert without for duration",
			content:   validVMRule,
			alertName: "LowAvailability",
			wantAlert: "LowAvailability",
			wantExpr:  "up == 0",
		},
		{
			name:      "alert not found",
			content:   validVMRule,
			alertName: "NonExistent",
			wantErr:   `alert "NonExistent" not found`,
		},
		{
			name:      "invalid YAML",
			content:   "not: valid: yaml: [",
			alertName: "Test",
			wantErr:   "unmarshaling YAML",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTestFile(t, tt.content)
			rule, err := ParseAlertRule(path, tt.alertName)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAlert, rule.Alert)
			assert.Equal(t, tt.wantExpr, rule.Expr)
		})
	}
}

func TestParseAlertRule_fileNotFound(t *testing.T) {
	_, err := ParseAlertRule("/nonexistent/path/rules.yaml", "Test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading file")
}

func TestParseAlertRule_forDuration(t *testing.T) {
	path := writeTestFile(t, validVMRule)

	rule, err := ParseAlertRule(path, "HighLatency")
	require.NoError(t, err)
	assert.NotZero(t, rule.For, "expected non-zero For duration")

	rule, err = ParseAlertRule(path, "LowAvailability")
	require.NoError(t, err)
	assert.Zero(t, rule.For, "expected zero For duration when not specified")
}

func TestParseAlertRule_labelsAndAnnotations(t *testing.T) {
	path := writeTestFile(t, validVMRule)

	rule, err := ParseAlertRule(path, "HighLatency")
	require.NoError(t, err)
	assert.Equal(t, "critical", rule.Labels["severity"])
	assert.Equal(t, "High latency detected", rule.Annotations["summary"])
}

func TestParseAlertRule_invalidForDuration(t *testing.T) {
	content := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMRule
spec:
  groups:
    - name: test
      rules:
        - alert: BadDuration
          expr: up == 0
          for: notaduration
`
	path := writeTestFile(t, content)
	_, err := ParseAlertRule(path, "BadDuration")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing duration")
}
