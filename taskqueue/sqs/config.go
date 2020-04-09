package sqs

import "time"

type config struct {
	TaskConcurrency            int
	PollingInterval            time.Duration
	VisibilityTimeout          time.Duration
	VisibilityExtensionEnabled bool
	MaxVisibilityExtensions    int
}

type Option func(c *config)

func WithTaskConcurrency(concurrency int) Option {
	return func(c *config) {
		c.TaskConcurrency = concurrency
	}
}
func WithPollingInterval(interval time.Duration) Option {
	return func(c *config) {
		c.PollingInterval = interval
	}
}

func WithVisibilityTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.VisibilityTimeout = timeout
	}
}

func WithVsibilityExtensionEnabled(enabled bool) Option {
	return func(c *config) {
		c.VisibilityExtensionEnabled = enabled
	}
}
func WithMaxVisibilityExtensions(max int) Option {
	return func(c *config) {
		c.MaxVisibilityExtensions = max
	}
}

var defaultConfig = config{
	TaskConcurrency:            4,
	PollingInterval:            10 * time.Second,
	VisibilityTimeout:          30 * time.Second,
	VisibilityExtensionEnabled: false,
	MaxVisibilityExtensions:    10,
}
