package memory

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/kinecosystem/agora-common/metrics"
)

const ClientType = "memory"

const (
	metricFormat = "%s_%s"
)

func init() {
	metrics.RegisterClientCtor(ClientType, newClient)
}

// CountRecord is a record of a call to Count
type CountRecord struct {
	Name  string
	Value int64
	Tags  []string
}

type Client struct {
	sync.Mutex
	countRecords []CountRecord
	config       *metrics.ClientConfig
}

// newClient returns an in-memory metrics client
func newClient(config *metrics.ClientConfig) (metrics.Client, error) {
	return &Client{
		countRecords: make([]CountRecord, 0),
		config:       config,
	}, nil
}

// Count implements metrics.Client.Count
func (c *Client) Count(name string, value int64, tags []string) error {
	if rand.Float64() <= c.config.SampleRate {
		c.Lock()
		defer c.Unlock()

		tags = append(tags, c.config.GlobalTags...)
		c.countRecords = append(c.countRecords, CountRecord{
			Name:  fmt.Sprintf(metricFormat, c.config.Namespace, name),
			Value: value,
			Tags:  tags,
		})
	}
	return nil
}

// getCountRecords returns the count records that have been tracked so far.
func (c *Client) getCountRecords() []CountRecord {
	c.Lock()
	defer c.Unlock()

	records := make([]CountRecord, len(c.countRecords))
	copy(records, c.countRecords)

	return records
}

// reset resets the client ack to its original state
func (c *Client) reset() {
	c.Lock()
	c.countRecords = make([]CountRecord, 0)
	c.Unlock()
}

// Close implements metrics.Client.Close
func (c *Client) Close() error {
	return nil
}
