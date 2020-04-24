package metrics

import (
	"time"

	"github.com/goburrow/cache"
)

type statsCounter struct {
	c    Client
	tags []string
}

// NewMangoStatsCounter provides an adapter for github.com/goburrow/cache.StatsCounter
// over this packages Client.
func NewMangoStatsCounter(c Client, tags ...string) cache.StatsCounter {
	return &statsCounter{
		c:    c,
		tags: tags,
	}
}

// RecordHits records cache hits.
func (s *statsCounter) RecordHits(count uint64) {
	_ = s.c.Count("cache_hits", int64(count), s.tags)
}

// RecordMisses records cache misses.
func (s *statsCounter) RecordMisses(count uint64) {
	_ = s.c.Count("cache_misses", int64(count), s.tags)
}

// RecordLoadSuccess records successful load of a new entry.
func (s *statsCounter) RecordLoadSuccess(loadTime time.Duration) {
	_ = s.c.Timing("cache_loads", loadTime, s.tags)
}

// RecordLoadError records failed load of a new entry.
func (s *statsCounter) RecordLoadError(loadTime time.Duration) {
	_ = s.c.Timing("cache_load_errors", loadTime, s.tags)
}

// RecordEviction records eviction of an entry from the cache.
func (s *statsCounter) RecordEviction() {
	_ = s.c.Count("cache_evictions", 1, s.tags)
}

// Snapshot writes snapshot of this counter values to the given Stats pointer.
func (s *statsCounter) Snapshot(_ *cache.Stats) {
}
