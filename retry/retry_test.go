package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kinecosystem/agora-common/retry/backoff"
)

func TestRealSleeper(t *testing.T) {
	sleeperImpl = &realSleeper{}

	start := time.Now()
	Retry(func() error { return errors.New("err") },
		Limit(2),
		Backoff(backoff.Constant(500*time.Millisecond), 500*time.Millisecond))

	assert.True(t, 500*time.Millisecond <= time.Since(start))
	assert.True(t, 1*time.Second > time.Since(start))
}

func TestRetrier(t *testing.T) {
	retriableErr := errors.New("retriable")
	r := NewRetrier(Limit(5), RetriableErrors(retriableErr))

	// Happy path always goes through
	attempts, err := r.Retry(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, uint(1), attempts)

	// Test ordering does not matter, by triggering 1 filter, then the other.
	attempts, err = r.Retry(func() error { return errors.New("unknown") })
	assert.Error(t, err)
	assert.Equal(t, uint(1), attempts)

	attempts, err = r.Retry(func() error { return retriableErr })
	assert.EqualError(t, retriableErr, err.Error())
	assert.Equal(t, uint(5), attempts)
}
