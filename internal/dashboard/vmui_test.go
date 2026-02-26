package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVMUI(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		expr    string
		from    time.Time
		to      *time.Time
		want    string
	}{
		{
			name:    "with resolved time",
			baseURL: "http://localhost:8428/vmui",
			expr:    `up{job="test"} == 0`,
			from:    testFrom,
			to:      &testTo,
			want:    `http://localhost:8428/vmui?g0.end_input=2024-01-15T11%3A05%3A00&g0.expr=up%7Bjob%3D%22test%22%7D+%3D%3D+0&g0.range_input=1h10m`,
		},
		{
			name:    "nil to pads from",
			baseURL: "http://localhost:8428/vmui",
			expr:    "up",
			from:    testFrom,
			to:      nil,
			want:    `http://localhost:8428/vmui?g0.end_input=2024-01-15T10%3A05%3A00&g0.expr=up&g0.range_input=10m`,
		},
		{
			name:    "preserves existing query params",
			baseURL: "http://localhost:8428/vmui?tenant=0",
			expr:    "up",
			from:    testFrom,
			to:      nil,
			want:    `http://localhost:8428/vmui?g0.end_input=2024-01-15T10%3A05%3A00&g0.expr=up&g0.range_input=10m&tenant=0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := newVMUI(tt.baseURL)
			require.NoError(t, err)
			assert.Equal(t, tt.want, b.BuildURL(tt.expr, tt.from, tt.to))
		})
	}
}
