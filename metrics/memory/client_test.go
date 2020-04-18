package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kinecosystem/agora-common/metrics"
)

func TestCount(t *testing.T) {
	config := &metrics.ClientConfig{
		Namespace:  "test",
		GlobalTags: []string{"testtag"},
	}

	client, err := newClient(config)
	require.NoError(t, err)

	records := []CountRecord{
		{
			Name:  "metric1",
			Value: 2,
			Tags:  []string{"tag1"},
		},
		{
			Name:  "metric2",
			Value: 1,
			Tags:  []string{"tag2"},
		},
		{
			Name:  "metric3",
			Value: -1,
			Tags:  []string{"tag3"},
		},
	}

	for _, record := range records {
		require.NoError(t, client.Count(record.Name, record.Value, record.Tags))
	}

	actualRecords := client.(*Client).getCountRecords()
	assert.Equal(t, 3, len(actualRecords))

	for idx, actual := range actualRecords {
		assert.Equal(t, fmt.Sprintf(metricFormat, config.Namespace, records[idx].Name), actual.Name)
		assert.Equal(t, records[idx].Value, actual.Value)
		assert.Equal(t, append(records[idx].Tags, config.GlobalTags...), actual.Tags)
	}
}

func TestGauge(t *testing.T) {
	config := &metrics.ClientConfig{
		Namespace:  "test",
		GlobalTags: []string{"testtag"},
	}

	client, err := newClient(config)
	require.NoError(t, err)

	records := []GaugeRecord{
		{
			Name:  "metric1",
			Value: 2.0,
			Tags:  []string{"tag1"},
		},
		{
			Name:  "metric2",
			Value: 1.1,
			Tags:  []string{"tag2"},
		},
		{
			Name:  "metric3",
			Value: -1,
			Tags:  []string{"tag3"},
		},
	}

	for _, record := range records {
		require.NoError(t, client.Gauge(record.Name, record.Value, record.Tags))
	}

	actualRecords := client.(*Client).getGaugeRecords()
	assert.Equal(t, 3, len(actualRecords))

	for idx, actual := range actualRecords {
		assert.Equal(t, fmt.Sprintf(metricFormat, config.Namespace, records[idx].Name), actual.Name)
		assert.Equal(t, records[idx].Value, actual.Value)
		assert.Equal(t, append(records[idx].Tags, config.GlobalTags...), actual.Tags)
	}
}

func TestTiming(t *testing.T) {
	config := &metrics.ClientConfig{
		Namespace:  "test",
		GlobalTags: []string{"testtag"},
	}

	client, err := newClient(config)
	require.NoError(t, err)

	records := []TimingRecord{
		{
			Name:  "metric1",
			Value: time.Second,
			Tags:  []string{"tag1"},
		},
		{
			Name:  "metric2",
			Value: time.Minute,
			Tags:  []string{"tag2"},
		},
		{
			Name:  "metric3",
			Value: time.Hour,
			Tags:  []string{"tag3"},
		},
	}

	for _, record := range records {
		require.NoError(t, client.Timing(record.Name, record.Value, record.Tags))
	}

	actualRecords := client.(*Client).getTimingRecords()
	assert.Equal(t, 3, len(actualRecords))

	for idx, actual := range actualRecords {
		assert.Equal(t, fmt.Sprintf(metricFormat, config.Namespace, records[idx].Name), actual.Name)
		assert.Equal(t, records[idx].Value, actual.Value)
		assert.Equal(t, append(records[idx].Tags, config.GlobalTags...), actual.Tags)
	}
}
