package testutil

import (
	"errors"
	"time"
)

// WaitFor waits for a condition to be met before the specified timeout
func WaitFor(timeout, interval time.Duration, condition func() bool) error {
	if timeout < interval {
		return errors.New("timeout must be greater than interval")
	}
	start := time.Now()
	for {
		select {
		case <-time.After(interval):
			if time.Since(start) >= timeout {
				return errors.New("condition was not met in time")
			}
			if condition() {
				return nil
			}
		}
	}
}
