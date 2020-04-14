package metrics

// Client is used for exporting metrics
type Client interface {
	// Count measures the count of a metric
	Count(name string, value int64, tags []string) error

	// Close closes the client and any underlying resources
	Close() error
}
