package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGlobalValidate(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name    string
		global  Global
		wantErr string
	}{
		{
			name: "valid config",
			global: Global{
				From:        base,
				To:          base.Add(time.Hour),
				Interval:    time.Second,
				Parallelism: 10,
			},
		},
		{
			name: "from after to",
			global: Global{
				From:        base.Add(time.Hour),
				To:          base,
				Interval:    time.Second,
				Parallelism: 10,
			},
			wantErr: "--from must be before --to",
		},
		{
			name: "from equals to",
			global: Global{
				From:        base,
				To:          base,
				Interval:    time.Second,
				Parallelism: 10,
			},
			wantErr: "--from must be before --to",
		},
		{
			name: "parallelism zero",
			global: Global{
				From:        base,
				To:          base.Add(time.Hour),
				Interval:    time.Second,
				Parallelism: 0,
			},
			wantErr: "--parallelism must be at least 1",
		},
		{
			name: "parallelism negative",
			global: Global{
				From:        base,
				To:          base.Add(time.Hour),
				Interval:    time.Second,
				Parallelism: -1,
			},
			wantErr: "--parallelism must be at least 1",
		},
		{
			name: "interval zero",
			global: Global{
				From:        base,
				To:          base.Add(time.Hour),
				Interval:    0,
				Parallelism: 10,
			},
			wantErr: "--interval must be at least 1ms",
		},
		{
			name: "interval below 1ms",
			global: Global{
				From:        base,
				To:          base.Add(time.Hour),
				Interval:    500 * time.Microsecond,
				Parallelism: 10,
			},
			wantErr: "--interval must be at least 1ms",
		},
		{
			name: "interval negative",
			global: Global{
				From:        base,
				To:          base.Add(time.Hour),
				Interval:    -time.Second,
				Parallelism: 10,
			},
			wantErr: "--interval must be at least 1ms",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.global.Validate()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
