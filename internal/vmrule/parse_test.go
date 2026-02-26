package vmrule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAlertRule(t *testing.T) {
	for _, tt := range []struct {
		name      string
		filePath  string
		alertName string
		wantAlert string
		wantExpr  string
		wantErr   string
	}{
		{
			name:      "finds alert in first group",
			filePath:  "testdata/vmrule-valid.yml",
			alertName: "HighLatency",
			wantAlert: "HighLatency",
			wantExpr:  "histogram_quantile(0.99, rate(http_duration_seconds_bucket[5m])) > 1",
		},
		{
			name:      "finds alert in second group",
			filePath:  "testdata/vmrule-valid.yml",
			alertName: "DiskFull",
			wantAlert: "DiskFull",
			wantExpr:  "node_filesystem_avail_bytes / node_filesystem_size_bytes < 0.1",
		},
		{
			name:      "alert without for duration",
			filePath:  "testdata/vmrule-valid.yml",
			alertName: "LowAvailability",
			wantAlert: "LowAvailability",
			wantExpr:  "up == 0",
		},
		{
			name:      "alert not found",
			filePath:  "testdata/vmrule-valid.yml",
			alertName: "NonExistent",
			wantErr:   `alert "NonExistent" not found`,
		},
		{
			name:      "invalid YAML",
			filePath:  "testdata/vmrule-invalid.yml",
			alertName: "Test",
			wantErr:   "unmarshaling YAML",
		},
		{
			name:      "not found",
			filePath:  "/nonexistent/path/rules.yaml",
			alertName: "Test",
			wantErr:   "reading file",
		},

		{
			name:      "invalid for duration",
			filePath:  "testdata/vmrule-invalid-duration.yml",
			alertName: "BadDuration",
			wantErr:   "parsing duration",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := ParseAlertRule(tt.filePath, tt.alertName)

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

func TestParseAlertRule_forDuration(t *testing.T) {
	rule, err := ParseAlertRule("testdata/vmrule-valid.yml", "HighLatency")
	require.NoError(t, err)
	assert.NotZero(t, rule.For, "expected non-zero For duration")

	rule, err = ParseAlertRule("testdata/vmrule-valid.yml", "LowAvailability")
	require.NoError(t, err)
	assert.Zero(t, rule.For, "expected zero For duration when not specified")
}

func TestParseAlertRule_labelsAndAnnotations(t *testing.T) {
	rule, err := ParseAlertRule("testdata/vmrule-valid.yml", "HighLatency")
	require.NoError(t, err)
	assert.Equal(t, "critical", rule.Labels["severity"])
	assert.Equal(t, "High latency detected", rule.Annotations["summary"])
}
