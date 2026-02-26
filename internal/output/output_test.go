package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcTableHeight(t *testing.T) {
	frameSize := baseStyle.GetVerticalFrameSize()

	for _, tt := range []struct {
		name       string
		termHeight int
		want       int
	}{
		{
			name:       "reasonable terminal height",
			termHeight: 40,
			want:       40 - 1 - frameSize,
		},
		{
			name:       "small terminal returns minimum",
			termHeight: 1,
			want:       5,
		},
		{
			name:       "zero terminal height returns minimum",
			termHeight: 0,
			want:       5,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, calcTableHeight(tt.termHeight))
		})
	}
}

func TestBuildColumns(t *testing.T) {
	for _, tt := range []struct {
		name         string
		termWidth    int
		hasSourceCol bool
		wantCols     int
		wantFirst    string
	}{
		{
			name:         "without source column",
			termWidth:    140,
			hasSourceCol: false,
			wantCols:     4,
			wantFirst:    "Opened",
		},
		{
			name:         "with source column",
			termWidth:    140,
			hasSourceCol: true,
			wantCols:     5,
			wantFirst:    "Source",
		},
		{
			name:         "narrow terminal still has minimum labels width",
			termWidth:    50,
			hasSourceCol: false,
			wantCols:     4,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cols := buildColumns(tt.termWidth, tt.hasSourceCol)
			assert.Len(t, cols, tt.wantCols)
			if tt.wantFirst != "" {
				assert.Equal(t, tt.wantFirst, cols[0].Title)
			}

			if tt.hasSourceCol {
				assert.Equal(t, colWidthSource, cols[0].Width)
				assert.Equal(t, colWidthOpened, cols[1].Width)
				assert.Equal(t, colWidthResolved, cols[2].Width)
				assert.Equal(t, colWidthDuration, cols[3].Width)
				assert.GreaterOrEqual(t, cols[4].Width, 20)
			} else {
				assert.Equal(t, colWidthOpened, cols[0].Width)
				assert.Equal(t, colWidthResolved, cols[1].Width)
				assert.Equal(t, colWidthDuration, cols[2].Width)
				assert.GreaterOrEqual(t, cols[3].Width, 20)
			}
		})
	}
}
