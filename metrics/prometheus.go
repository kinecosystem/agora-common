package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// MinuteDistributionBuckets is a set of buckets that provide insight for
// both subsecond latency requests, as well as exceptionally long ones.
var MinuteDistributionBuckets = append(
	[]float64{
		0.01,
		0.025,
		0.05,
		0.1,
		0.25,
		0.5,
		1.0,
	},
	prometheus.LinearBuckets(5.0, 5.0, 12)...,
)

// Register regsiters the provided prometheus collector, or returns
// the previously registered metric if it exists.
func Register(m prometheus.Collector) prometheus.Collector {
	if err := prometheus.Register(m); err != nil {
		if e, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return e.ExistingCollector
		}

		logrus.WithError(err).Error("failed to register metric")
	}
	return m
}
