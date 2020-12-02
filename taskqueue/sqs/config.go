package sqs

import "time"

type config struct {
	// TaskConcurrency configure the number of concurrent task workers
	// in the processor.
	TaskConcurrency int

	// PollingInterval is the 'long poll' interval against the SQS queue.
	//
	// If messages are continually available, this parameter has no effect.
	PollingInterval time.Duration

	// VisibilityTimeout configures the SQS visibility timeout.
	//
	// See: https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html
	VisibilityTimeout time.Duration

	// VisibilityExtensionEnabled configures whether or not the queue should
	// refresh the VisibilityTimeout of a message being processed.
	//
	// This is useful for tasks that take a significant amount of time.
	// Tasks that utilize this feature should continually check the provided
	// context to see if the task has been cancelled or not.
	VisibilityExtensionEnabled bool

	// MaxVisibilityExtensions is the maximum amount of of extensions that can
	// be made for a single task before becoming visibile on the queue again.
	MaxVisibilityExtensions int

	// PausedStart indicates that the processor's initial state should be paused.
	// In this state, the processor won't process tasks until Start() is called.
	PausedStart bool
}

// Option configures a Processor.
type Option func(c *config)

// WithTaskConcurrency configures the task concurrency.
func WithTaskConcurrency(concurrency int) Option {
	return func(c *config) {
		c.TaskConcurrency = concurrency
	}
}

// WithPollingInterval configures the polling interval.
func WithPollingInterval(interval time.Duration) Option {
	return func(c *config) {
		c.PollingInterval = interval
	}
}

// WithVisibilityTimeout configures the visibility timeout.
func WithVisibilityTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.VisibilityTimeout = timeout
	}
}

// WithVisibilityExtensionEnabled configures whether or not visibility extensions are enabled.
func WithVisibilityExtensionEnabled(enabled bool) Option {
	return func(c *config) {
		c.VisibilityExtensionEnabled = enabled
	}
}

// WithMaxVisibilityExtensions configures the maximum number of visibility extensions per message.
func WithMaxVisibilityExtensions(max int) Option {
	return func(c *config) {
		c.MaxVisibilityExtensions = max
	}
}

// WithPausedStart configures the processor to be initialized in a paused state.
func WithPausedStart() Option {
	return func(c *config) {
		c.PausedStart = true
	}
}

var defaultConfig = config{
	TaskConcurrency:            4,
	PollingInterval:            10 * time.Second,
	VisibilityTimeout:          30 * time.Second,
	VisibilityExtensionEnabled: false,
	MaxVisibilityExtensions:    10,
}
