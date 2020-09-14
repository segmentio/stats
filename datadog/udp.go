package datadog

import "net"

// udsWriter is an internal class wrapping around management of UDS connection
type udpWriter struct {
	conn net.Conn
}

// newUDSWriter returns a pointer to a new udpWriter given a socket file path as addr.
func newUDPWriter(addr string) (*udpWriter, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}
	return &udpWriter{conn: conn}, nil

}

// Write data to the UDS connection with write timeout and minimal error handling:
// create the connection if nil, and destroy it if the statsd server has disconnected
func (w *udpWriter) Write(data []byte) (int, error) {
	return w.conn.Write(data)
}

func (w *udpWriter) Close() error {
	return w.conn.Close()
}

func (w *udpWriter) CalcBufferSize(sizehint int) (int, error) {
	f, err := w.conn.(*net.UDPConn).File()
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return bufSizeFromFD(f, sizehint)
}
