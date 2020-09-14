package datadog

import (
	"io/ioutil"
	"net"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestUDSReconnectWhenConnRefused(t *testing.T) {
	dir, err := ioutil.TempDir("", "socket")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	socketPath := filepath.Join(dir, "dsd.socket")
	count := int32(0)
	closerServer1 := startUDSTestServerWithSocketFile(t, socketPath, HandlerFunc(func(m Metric, _ net.Addr) {
		atomic.AddInt32(&count, 1)
	}))
	defer closerServer1.Close()

	client := NewClientWith(ClientConfig{
		Address:    "unixgram://" + socketPath,
		BufferSize: 1, // small buffer to force write to unix socket for each measure
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
		t.Errorf("invalid error expected none, got %v", err)
	}
	// restart UDS server with same socket file
	closerServer2 := startUDSTestServerWithSocketFile(t, socketPath, HandlerFunc(func(m Metric, _ net.Addr) {
		atomic.AddInt32(&count, 1)
	}))

	defer closerServer2.Close()

	_, err = client.Write([]byte(measure))
	if err != nil {
		t.Errorf("unable to write data but should be able to as the client should reconnect %v", err)
	}

}
