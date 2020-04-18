package memory

import (
	"fmt"
	"sync"
	"time"

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

// GaugeRecord is a record of a call to Gauge
type GaugeRecord struct {
	Name  string
	Value float64
	Tags  []string
}

// TimingRecord is a record of a call to Timing
type TimingRecord struct {
	Name  string
	Value time.Duration
	Tags  []string
}

type Client struct {
	sync.Mutex
	countRecords  []CountRecord
	gaugeRecords  []GaugeRecord
	timingRecords []TimingRecord
	config        *metrics.ClientConfig
}

// newClient returns an in-memory metrics client
func newClient(config *metrics.ClientConfig) (metrics.Client, error) {
	return &Client{
		countRecords:  make([]CountRecord, 0),
		gaugeRecords:  make([]GaugeRecord, 0),
		timingRecords: make([]TimingRecord, 0),
		config:        config,
	}, nil
}

// Count implements metrics.Client.Count
func (c *Client) Count(name string, value int64, tags []string) error {
	c.Lock()
	defer c.Unlock()

	tags = append(tags, c.config.GlobalTags...)
	c.countRecords = append(c.countRecords, CountRecord{
		Name:  fmt.Sprintf(metricFormat, c.config.Namespace, name),
		Value: value,
		Tags:  tags,
	})
	return nil
}

// Gauge implements metrics.Client.Gauge
func (c *Client) Gauge(name string, value float64, tags []string) error {
	c.Lock()
	defer c.Unlock()

	tags = append(tags, c.config.GlobalTags...)
	c.gaugeRecords = append(c.gaugeRecords, GaugeRecord{
		Name:  fmt.Sprintf(metricFormat, c.config.Namespace, name),
		Value: value,
		Tags:  tags,
	})
	return nil
}

// Timing implements metrics.Client.Timing
func (c *Client) Timing(name string, value time.Duration, tags []string) error {
	c.Lock()
	defer c.Unlock()

	tags = append(tags, c.config.GlobalTags...)
	c.timingRecords = append(c.timingRecords, TimingRecord{
		Name:  fmt.Sprintf(metricFormat, c.config.Namespace, name),
		Value: value,
		Tags:  tags,
	})
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

// getGaugeRecords returns the count records that have been tracked so far.
func (c *Client) getGaugeRecords() []GaugeRecord {
	c.Lock()
	defer c.Unlock()

	records := make([]GaugeRecord, len(c.gaugeRecords))
	copy(records, c.gaugeRecords)

	return records
}

// getTimingRecords returns the count records that have been tracked so far.
func (c *Client) getTimingRecords() []TimingRecord {
	c.Lock()
	defer c.Unlock()

	records := make([]TimingRecord, len(c.timingRecords))
	copy(records, c.timingRecords)

	return records
}

// Close implements metrics.Client.Close
func (c *Client) Close() error {
	return nil
}
