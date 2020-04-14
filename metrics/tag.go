package metrics

import "fmt"

// TagOption specifies a tag that should be added to a metric
type TagOption func() string

// WithTypeTag adds a "type" tag to a metric. This is typically used to
// differentiate metrics from different implementations of an interface.
func WithTypeTag(typeName string) TagOption {
	return func() string {
		return "type:" + typeName
	}
}

// WithServiceTag adds a "service" tag to a metric. This is typically used
// to indicate which service a metric came from.
func WithServiceTag(serviceName string) TagOption {
	return func() string {
		return "service:" + serviceName
	}
}

// WithAppTAg adds an "app" tag to a metric. This is typically used to
// indicate which app a metric pertains to.
func WithApp(appIdx int16) TagOption {
	return func() string {
		return fmt.Sprintf("app:%d", appIdx)
	}
}

// GetTags returns a slice of tags given a set of TagOptions
func GetTags(opts ...TagOption) []string {
	tags := make([]string, 0)
	for _, opt := range opts {
		tags = append(tags, opt())
	}
	return tags
}
