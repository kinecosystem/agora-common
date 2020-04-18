package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidMetricName(t *testing.T) {
	require.NoError(t, validateMetricName("name"))
	require.NoError(t, validateMetricName("metric_name"))

	require.Error(t, validateMetricName(""))
	require.Error(t, validateMetricName("_metric"))
}
