package veneur

import (
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/datadog"
)

const (
	TagVeneurGlobalOnly = "veneurglobalonly"
	TagVeneurLocalOnly  = "veneurlocalonly"
	TagVeneurSinkOnly   = "veneursinkonly"
)

// The ClientConfig type is used to configure datadog clients.
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

// Client represents an datadog client that implements the stats.Handler
// interface.
type Client struct {
	*datadog.Client
	tags []stats.Tag
}

// NewClient creates and returns a new datadog client publishing metrics to the
// server running at addr.
func NewClient(addr string) *Client {
	return NewClientWith(ClientConfig{ClientConfig: datadog.ClientConfig{Address: addr}})
}

// NewClientWith creates and returns a new datadog client configured with the
// given config.
func NewClientWith(config ClientConfig) *Client {

	// Construct Veneur-specific Tags we will append to measures
	tags := []stats.Tag{}
	if config.GlobalOnly {
		tags = append(tags, stats.Tag{Name: TagVeneurGlobalOnly})
	} else if config.LocalOnly {
		tags = append(tags, stats.Tag{Name: TagVeneurLocalOnly})
	}
	for _, t := range config.SinksOnly {
		tags = append(tags, stats.Tag{Name: TagVeneurSinkOnly, Value: t})
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

// HandleMetric satisfies the stats.Handler interface.
func (c *Client) HandleMeasures(time time.Time, measures ...stats.Measure) {
	for _, m := range measures {
		m.Tags = append(m.Tags, c.tags...)
	}
	c.Client.HandleMeasures(time, measures...)
}
