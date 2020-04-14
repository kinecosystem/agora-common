package memory

import (
	"fmt"
	"testing"

	"github.com/kinecosystem/agora-common/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCount(t *testing.T) {
	config := &metrics.ClientConfig{
		Namespace:  "test",
		SampleRate: 1.0,
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
