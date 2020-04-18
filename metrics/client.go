package metrics

import "time"

// Client is used for exporting metrics
type Client interface {
	// Count measures the count of a metric
	Count(name string, value int64, tags []string) error

	// Gauge measures a metric at a point in time
	Gauge(name string, value float64, tags []string) error

	// Timing measures the time of a metric.
	Timing(name string, value time.Duration, tags []string) error

	// Close closes the client and any underlying resources
	Close() error
}
