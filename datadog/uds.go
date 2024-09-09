package datadog

import (
	"net"
	"sync"
	"time"
)

// UDSTimeout holds the default timeout for UDS socket writes, as they can get
// blocking when the receiving buffer is full.
// same value as in official datadog client: https://github.com/DataDog/datadog-go/blob/master/statsd/uds.go#L13
const defaultUDSTimeout = 1 * time.Millisecond

// udsWriter is an internal class wrapping around management of UDS connection
// credits to Datadog team: https://github.com/DataDog/datadog-go/blob/master/statsd/uds.go
type udsWriter struct {
	// Address to send metrics to, needed to allow reconnection on error
	addr net.Addr

	// Established connection object, or nil if not connected yet
	conn   net.Conn
	connMu sync.RWMutex // so that we can replace the failing conn on error

	// write timeout
	writeTimeout time.Duration
}

// newUDSWriter returns a pointer to a new udsWriter given a socket file path as addr.
func newUDSWriter(addr string) (*udsWriter, error) {
	udsAddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		return nil, err
	}
	// Defer connection to first read/write
	writer := &udsWriter{addr: udsAddr, conn: nil, writeTimeout: defaultUDSTimeout}
	return writer, nil
}

// Write data to the UDS connection with write timeout and minimal error handling:
// create the connection if nil, and destroy it if the statsd server has disconnected.
func (w *udsWriter) Write(data []byte) (int, error) {
	conn, err := w.ensureConnection()
	if err != nil {
		return 0, err
	}

	if err = conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)); err != nil {
		return 0, err
	}

	n, err := conn.Write(data)
	// var netErr net.Error
	//	if err != nil && (!errors.As(err, &netErr) || !err.()) {
	if err, isNetworkErr := err.(net.Error); err != nil && (!isNetworkErr || !err.Timeout()) {
		// Statsd server disconnected, retry connecting at next packet
		w.unsetConnection()
		return 0, err
	}
	return n, err
}

func (w *udsWriter) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

func (w *udsWriter) CalcBufferSize(sizehint int) (int, error) {
	conn, err := w.ensureConnection()
	if err != nil {
		return 0, err
	}
	f, err := conn.(*net.UnixConn).File()
	if err != nil {
		w.unsetConnection()
		return 0, err
	}
	defer f.Close()

	return bufSizeFromFD(f, sizehint)
}

func (w *udsWriter) ensureConnection() (net.Conn, error) {
	// Check if we've already got a socket we can use
	w.connMu.RLock()
	currentConn := w.conn
	w.connMu.RUnlock()

	if currentConn != nil {
		return currentConn, nil
	}

	// Looks like we might need to connect - try again with write locking.
	w.connMu.Lock()
	defer w.connMu.Unlock()
	if w.conn != nil {
		return w.conn, nil
	}

	newConn, err := net.Dial(w.addr.Network(), w.addr.String())
	if err != nil {
		return nil, err
	}
	w.conn = newConn
	return newConn, nil
}

func (w *udsWriter) unsetConnection() {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	w.conn = nil
}
