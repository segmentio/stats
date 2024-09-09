package datadog

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestUDSReconnectsWhenConnRefused(t *testing.T) {
	dir, err := os.MkdirTemp("", "socket")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(dir)

	socketPath := filepath.Join(dir, "dsd.socket")
	closerServer1 := startUDSTestServerWithSocketFile(t, socketPath, HandlerFunc(func(_ Metric, _ net.Addr) {}))
	defer closerServer1.Close()

	client := NewClientWith(ClientConfig{
		Address:    "unixgram://" + socketPath,
		BufferSize: 1, // small buffer to force write to unix socket for each measure written
	})

	measure := `main.http.error.count:0|c|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity
`

	_, err = client.Write([]byte(measure))
	if err != nil {
		t.Errorf("unable to write data %v", err)
	}

	closerServer1.Close()

	_, err = client.Write([]byte(measure))
	if err == nil {
		t.Errorf("got no error but expected one as the connection should be refused as we closed the server")
	}
	// restart UDS server with same socket file
	closerServer2 := startUDSTestServerWithSocketFile(t, socketPath, HandlerFunc(func(_ Metric, _ net.Addr) {}))
	defer closerServer2.Close()

	_, err = client.Write([]byte(measure))
	if err != nil {
		t.Errorf("unable to write data but should be able to as the client should reconnect %v", err)
	}
}
