package datadog

import (
	"io"
	"log"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/segmentio/stats"
)

const (
	// DefaultAddress is the default address to which the datadog client tries
	// to connect to.
	DefaultAddress = "localhost:8125"

	// DefaultBufferSize is the default size for batches of metrics sent to
	// datadog.
	DefaultBufferSize = 1024

	// MaxBufferSize is a hard-limit on the max size of the datagram buffer.
	MaxBufferSize = 65507
)

// The ClientConfig type is used to configure datadog clients.
type ClientConfig struct {
	// Address of the datadog database to send metrics to.
	Address string

	// Maximum size of batch of events sent to datadog.
	BufferSize int
}

// Client represents an datadog client that implements the stats.Handler
// interface.
type Client struct {
	serializer
	err    error
	buffer stats.Buffer
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
	if len(config.Address) == 0 {
		config.Address = DefaultAddress
	}

	if config.BufferSize == 0 {
		config.BufferSize = DefaultBufferSize
	}

	c := &Client{
		serializer: serializer{},
	}

	conn, bufferSize, err := dial(config.Address, config.BufferSize)
	if err != nil {
		log.Printf("stats/datadog: %s", err)
	}

	c.conn, c.err = conn, err
	c.buffer.BufferSize = bufferSize
	c.buffer.Serializer = &c.serializer
	return c
}

// HandleMetric satisfies the stats.Handler interface.
func (c *Client) HandleMeasures(time time.Time, measures ...stats.Measure) {
	c.buffer.HandleMeasures(time, measures...)
}

// Flush satisfies the stats.Flusher interface.
func (c *Client) Flush() {
	c.buffer.Flush()
}

// Close flushes and closes the client, satisfies the io.Closer interface.
func (c *Client) Close() error {
	c.Flush()
	c.close()
	return c.err
}

type serializer struct {
	conn net.Conn
}

func (*serializer) AppendMeasures(b []byte, _ time.Time, measures ...stats.Measure) []byte {
	for _, m := range measures {
		b = AppendMeasure(b, m)
	}
	return b
}

func (s *serializer) Write(b []byte) (int, error) {
	if s.conn != nil {
		return s.conn.Write(b)
	}
	return 0, io.ErrClosedPipe
}

func (s *serializer) close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

func dial(address string, sizehint int) (conn net.Conn, bufsize int, err error) {
	var f *os.File

	if conn, err = net.Dial("udp", address); err != nil {
		return
	}

	if f, err = conn.(*net.UDPConn).File(); err != nil {
		conn.Close()
		return
	}
	defer f.Close()
	fd := int(f.Fd())

	// The kernel refuses to send UDP datagrams that are larger than the size of
	// the size of the socket send buffer. To maximize the number of metrics
	// sent in one batch we attempt to attempt to adjust the kernel buffer size
	// to accept larger datagrams, or fallback to the default socket buffer size
	// if it failed.
	if bufsize, err = syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF); err != nil {
		conn.Close()
		return
	}

	// The kernel applies a 2x factor on the socket buffer size, only half of it
	// is available to write datagrams from user-space, the other half is used
	// by the kernel directly.
	bufsize /= 2

	for sizehint > bufsize && sizehint > 0 {
		if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, sizehint); err == nil {
			bufsize = sizehint
			break
		}
		sizehint /= 2
	}

	// Even tho the buffer agrees to support a bigger size it shouldn't be
	// possible to send datagrams larger than 65 KB on an IPv4 socket, so let's
	// enforce the max size.
	if bufsize > MaxBufferSize {
		bufsize = MaxBufferSize
	}

	// Use the size hint as an upper bound, event if the socket buffer is
	// larger, this gives control in situations where the receive buffer size
	// on the other side is known but cannot be controlled so the client does
	// not produce datagrams that are too large for the receiver.
	//
	// Related issue: https://github.com/DataDog/dd-agent/issues/2638
	if bufsize > sizehint {
		bufsize = sizehint
	}

	// Creating the file put the socket in blocking mode, reverting.
	syscall.SetNonblock(fd, true)
	return
}
