package datadog

import (
	"log"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

const (
	// MaxBufferSize is a hard-limit on the max size of the datagram buffer.
	MaxBufferSize = 65507

	// DefaultAddress is the default address to which clients connection to.
	DefaultAddress = "localhost:8125"

	// DefaultBufferSize is the default size of the client buffer.
	DefaultBufferSize = 8192

	// DefaultFlushInterval is the default interval at which clients flush
	// metrics from their stats engine.
	DefaultFlushInterval = 1 * time.Second
)

// The ClientConfig type is used to configure datadog clients.
type ClientConfig struct {
	// Address of the dogstatsd agent to send metrics to.
	Address string

	// BufferSize is the size of the output buffer used by the client.
	BufferSize int
}

// Client represents a datadog client that pulls metrics from a stats engine and
// forward them to a dogstatsd agent.
type Client struct {
	conn *Conn
	once sync.Once
}

// NewClient creates and returns a new datadog client publishing metrics to the
// dogstatsd server listening for UDP datagram at addr.
func NewClient(addr string) *Client {
	return NewClientWith(ClientConfig{
		Address: addr,
	})
}

// NewClientWith creates and returns a new datadog client configured with config.
func NewClientWith(config ClientConfig) *Client {
	conn, err := DialConfig(ConnConfig{
		Address:    config.Address,
		BufferSize: config.BufferSize,
	})

	if err != nil {
		log.Printf("stats/datadog: opening a connection to %s failed: %s", config.Address, err)
	}

	return &Client{
		conn: conn,
	}
}

// Close satisfies the io.Closer interface.
func (c *Client) Close() (err error) {
	c.once.Do(func() {
		if c.conn != nil {
			err = c.conn.Close()
		}
	})
	return
}

// Flsuh satisfies the stats.Flusher interface.
func (c *Client) Flush() {
	if c.conn != nil {
		if err := c.conn.Flush(); err != nil {
			log.Printf("stats/datadog: sending metrics to %s failed: %s", c.conn.RemoteAddr(), err)
		}
	}
}

// HandleMetric satisfies the stats.Handler interface.
func (c *Client) HandleMetric(m *stats.Metric) {
	if c.conn != nil {
		buf := bufferPool.Get().(*buffer)
		buf.b = appendMetric(buf.b[:0], Metric{
			Type:      metricType(m),
			Namespace: m.Namespace,
			Name:      m.Name,
			Value:     m.Value,
			Tags:      m.Tags,
		})
		if _, err := c.conn.Write(buf.b); err != nil {
			log.Printf("stats/datadog: sending metric %s to %s failed: %s", m.Name, c.conn.RemoteAddr(), err)
		}
		bufferPool.Put(buf)
	}
}
