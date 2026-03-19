package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFile(t *testing.T) {
	for _, tt := range []struct {
		name       string
		filePath   string
		wantGroups int
		wantRule   string
		wantErr    string
	}{
		{
			name:       "parses VMRule groups",
			filePath:   "testdata/vmrule-valid.yml",
			wantGroups: 2,
			wantRule:   "HighLatency",
		},
		{
			name:       "parses Prometheus rule groups",
			filePath:   "testdata/prometheus-valid.yml",
			wantGroups: 2,
			wantRule:   "HighLatency",
		},
		{
			name:     "invalid YAML",
			filePath: "testdata/invalid.yml",
			wantErr:  "unmarshal",
		},
		{
			name:     "file not found",
			filePath: "/nonexistent/path/rules.yaml",
			wantErr:  "reading file",
		},
		{
			name:     "invalid VMRule duration",
			filePath: "testdata/vmrule-invalid-duration.yml",
			wantErr:  "duration",
		},
		{
			name:     "invalid Prometheus duration",
			filePath: "testdata/prometheus-invalid-duration.yml",
			wantErr:  "duration",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			groups, err := parseFile(tt.filePath)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, groups.Groups, tt.wantGroups)
			require.NotEmpty(t, groups.Groups[0].Rules)
			assert.Equal(t, tt.wantRule, groups.Groups[0].Rules[0].Alert)
		})
	}
}

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
			name:      "finds VMRule alert in first group",
			filePath:  "testdata/vmrule-valid.yml",
			alertName: "HighLatency",
			wantAlert: "HighLatency",
			wantExpr:  "histogram_quantile(0.99, rate(http_duration_seconds_bucket[5m])) > 1",
		},
		{
			name:      "finds VMRule alert in second group",
			filePath:  "testdata/vmrule-valid.yml",
			alertName: "DiskFull",
			wantAlert: "DiskFull",
			wantExpr:  "node_filesystem_avail_bytes / node_filesystem_size_bytes < 0.1",
		},
		{
			name:      "finds Prometheus alert without for duration",
			filePath:  "testdata/prometheus-valid.yml",
			alertName: "LowAvailability",
			wantAlert: "LowAvailability",
			wantExpr:  "up == 0",
		},
		{
			name:      "alert not found",
			filePath:  "testdata/prometheus-valid.yml",
			alertName: "NonExistent",
			wantErr:   `alert "NonExistent" not found`,
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

func TestParseAlertRuleForDuration(t *testing.T) {
	rule, err := ParseAlertRule("testdata/vmrule-valid.yml", "HighLatency")
	require.NoError(t, err)
	assert.NotZero(t, rule.For, "expected non-zero For duration")

	rule, err = ParseAlertRule("testdata/prometheus-valid.yml", "LowAvailability")
	require.NoError(t, err)
	assert.Zero(t, rule.For, "expected zero For duration when not specified")
}

func TestParseAlertRuleLabelsAndAnnotations(t *testing.T) {
	rule, err := ParseAlertRule("testdata/prometheus-valid.yml", "HighLatency")
	require.NoError(t, err)
	assert.Equal(t, "critical", rule.Labels["severity"])
	assert.Equal(t, "High latency detected", rule.Annotations["summary"])
}
