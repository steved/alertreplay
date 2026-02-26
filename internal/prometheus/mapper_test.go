package prometheus

import (
	"reflect"
	"testing"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseWithMapper(t *testing.T, input string) ([]metricsql.LabelFilter, error) {
	t.Helper()

	var cli struct {
		Filters []metricsql.LabelFilter
	}

	parser, err := kong.New(&cli, kong.TypeMapper(reflect.TypeFor[metricsql.LabelFilter](), LabelFilterMapper))
	require.NoError(t, err, "kong.New returned unexpected error")

	_, err = parser.Parse([]string{"--filters", input})
	return cli.Filters, err
}

func TestLabelFilterMapper(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected []metricsql.LabelFilter
		wantErr  bool
	}{
		{
			name:     "single filter",
			input:    "cluster=region1",
			expected: []metricsql.LabelFilter{{Label: "cluster", Value: "region1"}},
		},
		{
			name:     "multiple filters",
			input:    "cluster=region1,cluster=region2",
			expected: []metricsql.LabelFilter{{Label: "cluster", Value: "region1"}, {Label: "cluster", Value: "region2"}},
		},
		{
			name:    "invalid",
			input:   "cluster",
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := parseWithMapper(t, tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, filters)
			}
		})
	}

}
