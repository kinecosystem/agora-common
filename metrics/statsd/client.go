package statsd

import (
	"os"
	"strconv"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/kinecosystem/agora-common/metrics"
)

const ClientType = "statsd"

const (
	connAddrEnvVar = "METRICS_CONN_ADDR"
	bufferEnvVar   = "METRICS_BUFFER"
	sampleRateEnvVar = "METRICS_SAMPLE_RATE"

	defaultConnStr = "localhost:8125"
	defaultBuffer  = 128
	defaultSampleRate = 1.0
)

func init() {
	metrics.RegisterClientCtor(ClientType, newClient)
}

type Client struct {
	client *statsd.Client
	config *metrics.ClientConfig
	sampleRate float64
}

// newClient returns a metrics.Client backed by a StatsD-based Datadog client
func newClient(config *metrics.ClientConfig) (metrics.Client, error) {
	log := logrus.StandardLogger().WithField("type", "metrics/statsd")

	var connAddr string
	var buffer int
	var sampleRate float64

	connAddr = os.Getenv(connAddrEnvVar)
	if len(connAddr) == 0 {
		log.Infof("connection address not configured, using default (%s)", defaultConnStr)
		connAddr = defaultConnStr
	}

	bufferStr := os.Getenv(bufferEnvVar)
	if len(bufferStr) == 0 {
		log.Infof("buffer not configured, using default (%d)", defaultBuffer)
		buffer = defaultBuffer
	} else {
		parsed, err := strconv.ParseInt(bufferStr, 10, 64)
		if err != nil {
			return nil, errors.Errorf("configured buffer invalid (%s)", bufferStr)
		}
		buffer = int(parsed)
	}

	sampleRateStr := os.Getenv(sampleRateEnvVar)
	if len(sampleRateStr) == 0 {
		log.Infof("sample rate not configured, using default (%.2f)", defaultSampleRate)
		sampleRate = defaultSampleRate
	} else {
		parsed, err := strconv.ParseFloat(sampleRateStr, 64)
		if err != nil {
			return nil, errors.Errorf("configured sample rate invalid (%s)", sampleRateStr)
		}
		sampleRate = parsed
	}

	client, err := statsd.NewBuffered(connAddr, buffer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create statsd client")
	}

	client.Namespace = config.Namespace
	client.Tags = config.GlobalTags

	return &Client{
		client: client,
		config: config,
		sampleRate: sampleRate,
	}, nil
}

// Count implements metrics.Client.Count
func (c *Client) Count(name string, value int64, tags []string) error {
	return c.client.Count(name, value, tags, c.sampleRate)
}

func (c *Client) Close() error {
	return c.client.Close()
}
