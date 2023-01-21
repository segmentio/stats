package veneur

import (
	"time"

	"github.com/vertoforce/stats"
	"github.com/vertoforce/stats/datadog"
)

// Const Sink Configuration types
const (
	GlobalOnly = "veneurglobalonly"
	LocalOnly  = "veneurlocalonly"
	SinkOnly   = "veneursinkonly"

	SignalfxSink = "signalfx"
	DatadogSink  = "datadog"
	KafkaSink    = "kafka"
)

// SinkOnly tags
var (
	TagSignalfxOnly = stats.Tag{Name: SinkOnly, Value: SignalfxSink}
	TagDatadogOnly  = stats.Tag{Name: SinkOnly, Value: DatadogSink}
	TagKafkaOnly    = stats.Tag{Name: SinkOnly, Value: KafkaSink}
)

// The ClientConfig type is used to configure veneur clients.
// It inherits the datadog config since the veneur client reuses
// the logic in the datadog client to emit metrics
type ClientConfig struct {
	datadog.ClientConfig

	// Veneur Specific Configuration
	// If set true, all metrics will be sent with veneurglobalonly tag
	GlobalOnly bool

	// If set true, all metrics will be sent with veneurlocalonly tag
	// Cannot be set in conjunction with GlobalOnly
	LocalOnly bool

	// Adds veneursinkonly:<sink> tag to all metrics. Valid sinks can be
	// found here: https://github.com/stripe/veneur#routing-metrics
	SinksOnly []string
}

// Client represents an veneur client that implements the stats.Handler
// interface.
type Client struct {
	*datadog.Client
	tags []stats.Tag
}

// NewClient creates and returns a new veneur client publishing metrics to the
// server running at addr.
func NewClient(addr string) *Client {
	return NewClientWith(ClientConfig{ClientConfig: datadog.ClientConfig{Address: addr}})
}

// NewClientGlobal creates a client that sends all metrics to the Global Veneur Aggregator
func NewClientGlobal(addr string) *Client {
	return NewClientWith(ClientConfig{ClientConfig: datadog.ClientConfig{Address: addr}, GlobalOnly: true})
}

// NewClientWith creates and returns a new veneur client configured with the
// given config.
func NewClientWith(config ClientConfig) *Client {

	// Construct Veneur-specific Tags we will append to measures
	tags := []stats.Tag{}
	if config.GlobalOnly {
		tags = append(tags, stats.Tag{Name: GlobalOnly})
	} else if config.LocalOnly {
		tags = append(tags, stats.Tag{Name: LocalOnly})
	}
	for _, t := range config.SinksOnly {
		tags = append(tags, stats.Tag{Name: SinkOnly, Value: t})
	}

	return &Client{
		Client: datadog.NewClientWith(datadog.ClientConfig{
			Address:    config.Address,
			BufferSize: config.BufferSize,
			Filters:    config.Filters,
		}),
		tags: tags,
	}
}

// HandleMeasures satisfies the stats.Handler interface.
func (c *Client) HandleMeasures(time time.Time, measures ...stats.Measure) {

	// If there are no tags to add, call HandleMeasures with measures directly
	if len(c.tags) == 0 {
		c.Client.HandleMeasures(time, measures...)
		return
	}

	finalMeasures := make([]stats.Measure, len(measures))
	for i := range measures {
		finalMeasures[i] = measures[i].Clone()
		finalMeasures[i].Tags = append(measures[i].Tags, c.tags...)
	}

	c.Client.HandleMeasures(time, finalMeasures...)
}
