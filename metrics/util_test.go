package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidMetricName(t *testing.T) {
	require.NoError(t, isValidMetricName("name"))
	require.NoError(t, isValidMetricName("metric_name"))

	require.Error(t, isValidMetricName(""))
	require.Error(t, isValidMetricName("_metric"))
}
