package datadog

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

// ConnConfig carries the configuration options that can be set when creating a
// connection.
type ConnConfig struct {
	Address    string
	BufferSize int
}

// A Conn represents a UDP connection to a dogstatsd server.
type Conn struct {
	m sync.Mutex
	c net.Conn
	b []byte
}

// Dial opens a new dogstatsd connection to address.
func Dial(address string) (conn *Conn, err error) {
	return DialConfig(ConnConfig{
		Address: address,
	})
}

// DialConfig opens a new dogstatsd connection using config.
func DialConfig(config ConnConfig) (conn *Conn, err error) {
	var c net.Conn
	var n int

	if len(config.Address) == 0 {
		config.Address = DefaultAddress
	}

	if config.BufferSize == 0 {
		config.BufferSize = DefaultBufferSize
	}

	if c, n, err = dial(config.Address, config.BufferSize); err != nil {
		return
	}

	conn = NewConn(c, make([]byte, 0, n))
	return
}

// NewConn creates a new dogstatsd connection with conn and buff.
func NewConn(conn net.Conn, buff []byte) *Conn {
	return &Conn{
		c: conn,
		b: buff,
	}
}

// Close satisfies the net.Conn interface.
func (c *Conn) Close() (err error) {
	err = c.Flush()
	c.c.Close()
	return
}

// Read satisfies the net.Conn interface.
func (c *Conn) Read(b []byte) (int, error) {
	return 0, io.EOF
}

// Write satisfies the net.Conn interface.
func (c *Conn) Write(b []byte) (n int, err error) {
	c.m.Lock()

	if n = len(b); n > cap(c.b) {
		c.m.Unlock()
		return 0, fmt.Errorf("discarded because it doesn't fit in the output buffer (size = %d, max = %d)", n, cap(c.b))
	}

	if n > (cap(c.b) - len(c.b)) {
		err = c.flush()
	}

	c.b = append(c.b, b...)
	c.m.Unlock()
	return
}

// LocalAddr satisfies the net.Conn interface.
func (c *Conn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

// RemoteAddr satisfies the net.Conn interface.
func (c *Conn) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

// SetDeadline satisfies the net.Conn interface.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

// SetReadDeadline satisfies the net.Conn interface.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.c.SetReadDeadline(t)
}

// SetWriteDeadline satisfies the net.Conn interface.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}

// Flush sends a UDP datagram containing all buffered data.
func (c *Conn) Flush() (err error) {
	c.m.Lock()
	err = c.flush()
	c.m.Unlock()
	return
}

func (c *Conn) flush() (err error) {
	if len(c.b) != 0 {
		_, err = c.c.Write(c.b)
		c.b = c.b[:0]
	}
	return
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
