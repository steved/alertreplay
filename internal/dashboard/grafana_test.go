package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrafana(t *testing.T) {
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
			baseURL: "http://localhost:3000/explore?orgId=1&ds=abc",
			expr:    `up{job="test"} == 0`,
			from:    testFrom,
			to:      &testTo,
			want:    `http://localhost:3000/explore?ds=abc&left=%7B%22queries%22%3A%5B%7B%22refId%22%3A%22A%22%2C%22expr%22%3A%22up%7Bjob%3D%5C%22test%5C%22%7D+%3D%3D+0%22%7D%5D%2C%22range%22%3A%7B%22from%22%3A%221705312500000%22%2C%22to%22%3A%221705316700000%22%7D%7D&orgId=1`,
		},
		{
			name:    "nil to pads from",
			baseURL: "http://localhost:3000/explore",
			expr:    "up",
			from:    testFrom,
			to:      nil,
			want:    `http://localhost:3000/explore?left=%7B%22queries%22%3A%5B%7B%22refId%22%3A%22A%22%2C%22expr%22%3A%22up%22%7D%5D%2C%22range%22%3A%7B%22from%22%3A%221705312500000%22%2C%22to%22%3A%221705313100000%22%7D%7D`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := newGrafana(tt.baseURL)
			require.NoError(t, err)
			assert.Equal(t, tt.want, b.BuildURL(tt.expr, tt.from, tt.to))
		})
	}
}
