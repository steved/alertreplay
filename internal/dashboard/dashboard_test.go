package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	testFrom = time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testTo   = time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
)

func TestNew(t *testing.T) {
	_, err := New(VMUI, "http://localhost:8428")
	require.NoError(t, err)

	_, err = New(Prometheus, "http://localhost:9090")
	require.NoError(t, err)

	_, err = New(Grafana, "http://localhost:3000/explore")
	require.NoError(t, err)

	_, err = New("unknown", "http://localhost")
	require.Error(t, err)
}
