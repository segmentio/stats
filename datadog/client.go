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
	DefaultBufferSize = 1024

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

	// Datadog has complained numerous times that the request paths
	// generate too many custom metrics on their side, setting this
	// flag to true strips out the http_req_path tag found in metrics.
	//
	// Programs that know they won't produce high-cardinality tags for
	// http_req_path can enable the tag.
	//
	// The default value is false.
	EnableHttpPathTag bool
}

// Client represents a datadog client that pulls metrics from a stats engine and
// forward them to a dogstatsd agent.
type Client struct {
	conn *Conn
	once sync.Once

	enableHttpPathTag bool
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
	} else {
		log.Printf("stats/datadog: connection opened to %s with a buffer size of %d B", config.Address, cap(conn.b))
	}

	return &Client{
		conn:              conn,
		enableHttpPathTag: config.EnableHttpPathTag,
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
		if !c.enableHttpPathTag {
			stripHttpPathTag(m)
		}
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

func stripHttpPathTag(m *stats.Metric) {
	for i, tag := range m.Tags {
		// fast path: no writes in the common case where the tag doesn't exist.
		if tag.Name == "http_req_path" {
			n := i
			for _, tag := range m.Tags[i:] {
				if tag.Name != "http_req_path" {
					m.Tags[n] = tag
					n++
				}
			}
			m.Tags = m.Tags[:n]
			break
		}
	}
}
