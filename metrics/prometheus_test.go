package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRegistration(t *testing.T) {
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "my_metric",
	})

	x := Register(c)
	assert.Equal(t, c, x)

	x = Register(c)
	assert.Equal(t, c, x)
}
