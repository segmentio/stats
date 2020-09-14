package datadog

import "net"

type udpWriter struct {
	conn net.Conn
}

// newUDPWriter returns a pointer to a new newUDPWriter given a socket file path as addr.
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

// Write data to the UDP connection
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
