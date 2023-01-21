package datadog

import (
	"log"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/vertoforce/stats"
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

var (
	// DefaultFilters are the default tags to filter before sending to
	// datadog. Using the request path as a tag can overwhelm datadog's
	// servers if there are too many unique routes due to unique IDs being a
	// part of the path. Only change the default filters if there are a static
	// number of routes.
	DefaultFilters = []string{"http_req_path"}

	// DefaultDistributionPrefixes is the default set of name prefixes for
	// metrics to be sent as distributions instead of as histograms.
	DefaultDistributionPrefixes = []string{}
)

// The ClientConfig type is used to configure datadog clients.
type ClientConfig struct {
	// Address of the datadog database to send metrics to.
	Address string

	// Maximum size of batch of events sent to datadog.
	BufferSize int

	// List of tags to filter. If left nil is set to DefaultFilters.
	Filters []string

	// Set of name prefixes for metrics to be sent as distributions instead of
	// as histograms.
	DistributionPrefixes []string

	// UseDistributions True indicates to send histograms with `d` type instead of `h` type
	// https://docs.datadoghq.com/developers/dogstatsd/datagram_shell?tab=metrics#the-dogstatsd-protocol
	UseDistributions bool
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

	if config.Filters == nil {
		config.Filters = DefaultFilters
	}

	if config.DistributionPrefixes == nil {
		config.DistributionPrefixes = DefaultDistributionPrefixes
	}

	// transform filters from array to map
	filterMap := make(map[string]struct{})
	for _, f := range config.Filters {
		filterMap[f] = struct{}{}
	}

	c := &Client{
		serializer: serializer{
			filters:          filterMap,
			distPrefixes:     config.DistributionPrefixes,
			useDistributions: config.UseDistributions,
		},
	}

	conn, bufferSize, err := dial(config.Address, config.BufferSize)
	if err != nil {
		log.Printf("stats/datadog: %s", err)
	}

	c.conn, c.err, c.bufferSize = conn, err, bufferSize
	c.buffer.BufferSize = bufferSize
	c.buffer.Serializer = &c.serializer
	log.Printf("stats/datadog: sending metrics with a buffer of size %d B", bufferSize)
	return c
}

// HandleMeasures satisfies the stats.Handler interface.
func (c *Client) HandleMeasures(time time.Time, measures ...stats.Measure) {
	c.buffer.HandleMeasures(time, measures...)
}

// Flush satisfies the stats.Flusher interface.
func (c *Client) Flush() {
	c.buffer.Flush()
}

// Write satisfies the io.Writer interface.
func (c *Client) Write(b []byte) (int, error) {
	return c.serializer.Write(b)
}

// Close flushes and closes the client, satisfies the io.Closer interface.
func (c *Client) Close() error {
	c.Flush()
	c.close()
	return c.err
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
