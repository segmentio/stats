package lambda

import (
	"os"
	"time"

	"github.com/segmentio/stats"
)

const (
	DefaultBufferSize = 1024
)

// ClientConfig represents the Lambda client configuration
type ClientConfig struct {
	BufferSize int
}

// Client implements the stats.Handler interface....
type Client struct {
	serializer
	buffer stats.Buffer
}

// NewClient creates and returns a Lambda client with default buffer size.
func NewClient() *Client {
	return NewClientWith(ClientConfig{})
}

// NewClientWith creates and returns a Lambda client with the given configuration.
func NewClientWith(config ClientConfig) *Client {
	if config.BufferSize == 0 {
		config.BufferSize = DefaultBufferSize
	}

	clt := &Client{
		serializer: serializer{},
	}
	clt.buffer.BufferSize = config.BufferSize
	clt.buffer.Serializer = &clt.serializer
	return clt
}

// HandleMeasures implements the stats.Handler interface
func (c *Client) HandleMeasures(t time.Time, measures ...stats.Measure) {
	c.buffer.HandleMeasures(t, measures...)
}

type serializer struct{}

// AppendMeasures implements the stats.Serializer interface
func (c *serializer) AppendMeasures(b []byte, time time.Time, measures ...stats.Measure) []byte {
	for _, m := range measures {
		b = AppendMeasure(b, time, m)
	}
	return b
}

// Write implements the io.Writer interface
func (s *serializer) Write(b []byte) (int, error) {
	return os.Stdout.Write(b)
}
