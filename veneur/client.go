package veneur

import (
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/datadog"
)

const (
	// DefaultAddress is the default address to which the veneur client tries
	// to connect to.
	DefaultAddress = "localhost:8125"

	// DefaultBufferSize is the default size for batches of metrics sent to
	// datadog.
	DefaultBufferSize = 1024

	// MaxBufferSize is a hard-limit on the max size of the datagram buffer.
	MaxBufferSize = 65507

	TagVeneurGlobalOnly = "veneurglobalonly"
	TagVeneurLocalOnly  = "veneurlocalonly"
	TagVeneurSinkOnly   = "veneursinkonly"
)

// DefaultFilter is the default tag to filter before sending to
// datadog. Using the request path as a tag can overwhelm datadog's
// servers if there are too many unique routes due to unique IDs being a
// part of the path. Only change the default filter if there is a static
// number of routes.
var (
	DefaultFilters = []string{"http_req_path"}
)

// The ClientConfig type is used to configure datadog clients.
type ClientConfig struct {
	// Address of the datadog database to send metrics to.
	Address string

	// Maximum size of batch of events sent to datadog.
	BufferSize int

	// List of tags to filter. If left nil is set to DefaultFilters.
	Filters []string

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
	ddClient *datadog.Client
	tags     []stats.Tag
}

// NewClient creates and returns a new datadog client publishing metrics to the
// server running at addr.
func NewClient(addr string) *Client {
	return NewClientWith(ClientConfig{
		Address: addr,
	})
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
		ddClient: datadog.NewClientWith(datadog.ClientConfig{
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
	c.ddClient.HandleMeasures(time, measures...)
}

// Flush satisfies the stats.Flusher interface.
func (c *Client) Flush() {
	c.ddClient.Flush()
}

// Write satisfies the io.Writer interface.
func (c *Client) Write(b []byte) (int, error) {
	return c.ddClient.Write(b)
}

// Close flushes and closes the client, satisfies the io.Closer interface.
func (c *Client) Close() error {
	return c.ddClient.Close()
}
