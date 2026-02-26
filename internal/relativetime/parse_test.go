package relativetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withFixedNow(t *testing.T, ts time.Time) {
	t.Helper()

	originalNow := now
	now = func() time.Time { return ts }
	t.Cleanup(func() {
		now = originalNow
	})
}

func TestParse(t *testing.T) {
	fixed := time.Date(2026, time.February, 25, 12, 30, 45, 0, time.UTC)
	withFixedNow(t, fixed)

	tests := []struct {
		name     string
		input    string
		want     time.Time
		wantErr  bool
		checkUTC bool
	}{
		{
			name:  "now trimmed and mixed case",
			input: "  NoW ",
			want:  fixed,
		},
		{
			name:  "seconds singular",
			input: "1 second ago",
			want:  fixed.Add(-1 * time.Second),
		},
		{
			name:  "minutes plural",
			input: "2 minutes ago",
			want:  fixed.Add(-2 * time.Minute),
		},
		{
			name:  "hours mixed case",
			input: "3 HOURS ago",
			want:  fixed.Add(-3 * time.Hour),
		},
		{
			name:  "days",
			input: "4 days ago",
			want:  fixed.Add(-4 * 24 * time.Hour),
		},
		{
			name:  "weeks",
			input: "5 weeks ago",
			want:  fixed.Add(-5 * 7 * 24 * time.Hour),
		},
		{
			name:  "months",
			input: "2 months ago",
			want:  fixed.AddDate(0, -2, 0),
		},
		{
			name:  "years singular",
			input: "1 year ago",
			want:  fixed.AddDate(-1, 0, 0),
		},
		{
			name:     "absolute timestamp",
			input:    "2026-02-24 11:22:33",
			want:     time.Date(2026, time.February, 24, 11, 22, 33, 0, time.UTC),
			checkUTC: true,
		},
		{
			name:    "invalid input",
			input:   "yesterday",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			if tc.wantErr {
				require.Error(t, err, "Parse(%q) should return an error", tc.input)
				return
			}

			require.NoError(t, err, "Parse(%q) returned unexpected error", tc.input)

			assert.Equal(t, tc.want, got, "Parse(%q) returned unexpected value", tc.input)

			if tc.checkUTC {
				assert.Equal(t, time.UTC, got.Location(), "Parse(%q) should return UTC time", tc.input)
			}
		})
	}
}
