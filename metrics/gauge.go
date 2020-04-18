package metrics

import (
	"sync"
	"time"
)

const DefaultGaugePollingInterval = 1 * time.Second

var (
	gaugeMu       sync.Mutex
	gaugeRegistry = make([]*Gauge, 0)
)

func init() {
	go pollAllGauges()
}

// GaugeFunc is used in a polling loop to get the metric's value. Processing here is
// expected to be fairly light weight since we use a single polling thread to process
// all gauges.
type GaugeFunc func() float64

// Gauge measures a metric's value at a point in time
type Gauge struct {
	client Client
	f      GaugeFunc
	name   string
	tags   []string
}

// NewGauge returns a new Gauge that polls the value returned by f
func NewGauge(client Client, name string, f GaugeFunc, tagOptions ...TagOption) (*Gauge, error) {
	if err := validateMetricName(name); err != nil {
		return nil, err
	}

	g := &Gauge{
		client: client,
		f:      f,
		name:   name,
		tags:   GetTags(tagOptions...),
	}

	// Lazy load the poller
	gaugeMu.Lock()
	gaugeRegistry = append(gaugeRegistry, g)
	gaugeMu.Unlock()
	return g, nil
}

func pollAllGauges() {
	for {
		time.Sleep(DefaultGaugePollingInterval)

		gaugeMu.Lock()
		for _, gauge := range gaugeRegistry {
			_ = gauge.client.Gauge(gauge.name, gauge.f(), gauge.tags)
		}
		gaugeMu.Unlock()
	}
}

// Stop removes the gauge from being polled
func (g *Gauge) Stop() {
	gaugeMu.Lock()
	for i, gauge := range gaugeRegistry {
		if g == gauge {
			gaugeRegistry = append(gaugeRegistry[:i], gaugeRegistry[i+1:]...)
			break
		}
	}
	gaugeMu.Unlock()
}
