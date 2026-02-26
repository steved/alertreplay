package relativetime

import (
	"reflect"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseWithMapper(t *testing.T, input string) (time.Time, error) {
	t.Helper()

	var cli struct {
		At time.Time `arg:""`
	}

	parser, err := kong.New(&cli, kong.TypeMapper(reflect.TypeFor[time.Time](), Mapper))
	require.NoError(t, err, "kong.New returned unexpected error")

	_, err = parser.Parse([]string{input})
	return cli.At, err
}

func TestMapper(t *testing.T) {
	fixed := time.Date(2026, time.February, 25, 12, 30, 45, 0, time.UTC)
	withFixedNow(t, fixed)

	tests := []struct {
		name        string
		input       string
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{
			name:  "decodes relative time",
			input: "2 hours ago",
			want:  fixed.Add(-2 * time.Hour),
		},
		{
			name:        "returns parse error",
			input:       "not-a-time",
			wantErr:     true,
			errContains: "not-a-time",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseWithMapper(t, tc.input)
			if tc.wantErr {
				require.Error(t, err, "parser.Parse(%q) should return an error", tc.input)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains, "parse error should include expected content")
				}
				return
			}

			require.NoError(t, err, "parser.Parse(%q) returned unexpected error", tc.input)

			assert.Equal(t, tc.want, got, "decoded value should match expected")
		})
	}
}
