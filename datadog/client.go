package datadog

import (
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	stats "github.com/segmentio/stats/v5"

	"golang.org/x/sys/unix"
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
	// UDP: host:port (default)
	// UDS: unixgram://dir/file.ext
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

	w, err := newWriter(config.Address)
	if err != nil {
		log.Printf("stats/datadog: %s", err)
		c.err = err
		w = &noopWriter{}
	}

	newBufSize, err := w.CalcBufferSize(config.BufferSize)
	if err != nil {
		log.Printf("stats/datadog: unable to calc writer's buffer size. Defaulting to a buffer of size %d - %v\n", DefaultBufferSize, err)
		newBufSize = DefaultBufferSize
	}

	c.bufferSize = newBufSize
	c.buffer.Serializer = &c.serializer
	c.buffer.BufferSize = newBufSize
	c.serializer.conn = w
	log.Printf("stats/datadog: sending metrics with a buffer of size %d B", newBufSize)
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

func bufSizeFromFD(f *os.File, sizehint int) (bufsize int, err error) {
	fd := int(f.Fd())

	// The kernel refuses to send UDP datagrams that are larger than the size of
	// the size of the socket send buffer. To maximize the number of metrics
	// sent in one batch we attempt to attempt to adjust the kernel buffer size
	// to accept larger datagrams, or fallback to the default socket buffer size
	// if it failed.
	if bufsize, err = unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF); err != nil {
		return
	}

	// The kernel applies a 2x factor on the socket buffer size, only half of it
	// is available to write datagrams from user-space, the other half is used
	// by the kernel directly.
	bufsize /= 2

	for sizehint > bufsize && sizehint > 0 {
		if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, sizehint); err == nil {
			bufsize = sizehint
			break
		}
		sizehint /= 2
	}

	// Even tho the buffer agrees to support a bigger size it shouldn't be
	// possible to send datagrams larger than 65 KB on an IPv5 socket, so let's
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
	_ = unix.SetNonblock(fd, true)
	return
}

type ddWriter interface {
	io.WriteCloser
	CalcBufferSize(desiredBufSize int) (int, error)
}

func newWriter(addr string) (ddWriter, error) {
	if strings.HasPrefix(addr, "unixgram://") ||
		strings.HasPrefix(addr, "udp://") {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		switch u.Scheme {
		case "unixgram":
			return newUDSWriter(u.Path)
		case "udp":
			return newUDPWriter(u.Path)
		}
	}
	// default assume addr host:port to use UDP
	return newUDPWriter(addr)
}

// noopWriter is a writer that does nothing.
type noopWriter struct{}

// Write writes nothing.
func (w *noopWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

// Close is a noop close.
func (w *noopWriter) Close() error {
	return nil
}

// CalcBufferSize returns the sizehint.
func (w *noopWriter) CalcBufferSize(sizehint int) (int, error) {
	return sizehint, nil
}
