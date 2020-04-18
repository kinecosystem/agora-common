package metrics

import (
	"sync"
	"time"
)

// Timer tracks the time a metric
type Timer struct {
	client Client
	name   string
	tags   []string
}

// TimerContext is the context for an in-flight datapoint
type TimerContext struct {
	stateMu        sync.Mutex
	timer          *Timer
	start          time.Time
	stopped        bool
	additionalTags []TagOption
}

// NewTimer returns a new Timer
func NewTimer(client Client, name string, tagOptions ...TagOption) (*Timer, error) {
	if err := validateMetricName(name); err != nil {
		return nil, err
	}

	return &Timer{
		client: client,
		name:   name,
		tags:   GetTags(tagOptions...),
	}, nil
}

// Time begins tracking a new datapoint. Use TimerContext.Stop on the
// returned context to record the time passed since calling Time
func (t *Timer) Time(tags ...TagOption) *TimerContext {
	return &TimerContext{
		start:          time.Now(),
		timer:          t,
		additionalTags: tags,
	}
}

// AddTiming emits a timing value that has already been observed
func (t *Timer) AddTiming(value time.Duration, tags ...TagOption) {
	t.submitTiming(value, tags)
}

func (t *Timer) submitTiming(value time.Duration, additionalTags []TagOption) {
	tags := append(t.tags, GetTags(additionalTags...)...)
	_ = t.client.Timing(t.name, value, tags)
}

// Stop records the time since the context's creation if it hasn't already
// been stopped
func (tc *TimerContext) Stop() {
	tc.stateMu.Lock()
	if !tc.stopped {
		tc.timer.submitTiming(time.Since(tc.start), tc.additionalTags)
		tc.stopped = true
	}
	tc.stateMu.Unlock()
}
