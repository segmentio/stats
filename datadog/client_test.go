package datadog

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vertoforce/stats"
)

func TestClient(t *testing.T) {
	client := NewClient(DefaultAddress)

	for i := 0; i != 1000; i++ {
		client.HandleMeasures(time.Time{}, stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				{Name: "count", Value: stats.ValueOf(5)},
				{Name: "rtt", Value: stats.ValueOf(100 * time.Millisecond)},
			},
			Tags: []stats.Tag{
				stats.T("answer", "42"),
				stats.T("hello", "world"),
			},
		})
	}

	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

func TestClientWithDistributionPrefixes(t *testing.T) {
	client := NewClientWith(ClientConfig{
		Address:              DefaultAddress,
		DistributionPrefixes: []string{"dist_"},
	})

	client.HandleMeasures(time.Time{}, stats.Measure{
		Name: "request",
		Fields: []stats.Field{
			{Name: "count", Value: stats.ValueOf(5)},
			stats.MakeField("dist_rtt", stats.ValueOf(100*time.Millisecond), stats.Histogram),
		},
		Tags: []stats.Tag{
			stats.T("answer", "42"),
			stats.T("hello", "world"),
		},
	})

	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

func TestClientWithUseDistributions(t *testing.T) {

	// Start a goroutine listening for packets and giving them back on packets chan
	packets := make(chan []byte)
	addr, closer := startUDPListener(t, packets)
	defer closer.Close()

	client := NewClientWith(ClientConfig{
		Address:          addr,
		UseDistributions: true,
	})

	testMeassure := stats.Measure{
		Name: "request",
		Fields: []stats.Field{
			{Name: "count", Value: stats.ValueOf(5)},
			stats.MakeField("dist_rtt", stats.ValueOf(100*time.Millisecond), stats.Histogram),
		},
		Tags: []stats.Tag{
			stats.T("answer", "42"),
			stats.T("hello", "world"),
		},
	}
	client.HandleMeasures(time.Time{}, testMeassure)
	client.Flush()

	expectedPacket1 := "request.count:5|c|#answer:42,hello:world\nrequest.dist_rtt:0.1|d|#answer:42,hello:world\n"
	assert.EqualValues(t, expectedPacket1, string(<-packets))

	client.useDistributions = false
	client.HandleMeasures(time.Time{}, testMeassure)
	client.Flush()

	expectedPacket2 := "request.count:5|c|#answer:42,hello:world\nrequest.dist_rtt:0.1|h|#answer:42,hello:world\n"
	assert.EqualValues(t, expectedPacket2, string(<-packets))

	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

func TestClientWriteLargeMetrics(t *testing.T) {
	const data = `main.http.error.count:0|c|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity
main.http.message.count:1|c|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,operation:read,type:request
main.http.message.header.size:2|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,operation:read,type:request
main.http.message.header.bytes:240|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,operation:read,type:request
main.http.message.body.bytes:0|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,operation:read,type:request
main.http.message.count:1|c|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,http_res_content_charset:,http_res_content_endoing:,http_res_content_type:application/json,http_res_protocol:HTTP/1.1,http_res_server:,http_res_transfer_encoding:identity,http_res_upgrade:,http_status:200,http_status_bucket:2xx,operation:write,type:response
main.http.message.header.size:1|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,http_res_content_charset:,http_res_content_endoing:,http_res_content_type:application/json,http_res_protocol:HTTP/1.1,http_res_server:,http_res_transfer_encoding:identity,http_res_upgrade:,http_status:200,http_status_bucket:2xx,operation:write,type:response
main.http.message.header.bytes:70|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,http_res_content_charset:,http_res_content_endoing:,http_res_content_type:application/json,http_res_protocol:HTTP/1.1,http_res_server:,http_res_transfer_encoding:identity,http_res_upgrade:,http_status:200,http_status_bucket:2xx,operation:write,type:response
main.http.message.body.bytes:839|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,http_res_content_charset:,http_res_content_endoing:,http_res_content_type:application/json,http_res_protocol:HTTP/1.1,http_res_server:,http_res_transfer_encoding:identity,http_res_upgrade:,http_status:200,http_status_bucket:2xx,operation:write,type:response
main.http.rtt.seconds:0.001215296|h|#http_req_content_charset:,http_req_content_endoing:,http_req_content_type:,http_req_host:localhost:3011,http_req_method:GET,http_req_protocol:HTTP/1.1,http_req_transfer_encoding:identity,http_res_content_charset:,http_res_content_endoing:,http_res_content_type:application/json,http_res_protocol:HTTP/1.1,http_res_server:,http_res_transfer_encoding:identity,http_res_upgrade:,http_status:200,http_status_bucket:2xx,operation:write,type:response
`

	count := int32(0)
	expect := int32(strings.Count(data, "\n"))

	addr, closer := startTestServer(t, HandlerFunc(func(m Metric, _ net.Addr) {
		atomic.AddInt32(&count, 1)
	}))
	defer closer.Close()

	client := NewClient(addr)

	if _, err := client.Write([]byte(data)); err != nil {
		t.Error(err)
	}

	time.Sleep(100 * time.Millisecond)

	if n := atomic.LoadInt32(&count); n != expect {
		t.Error("bad metric count:", n)
	}
}

func BenchmarkClient(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	for _, N := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("write a batch of %d measures to a client", N), func(b *testing.B) {
			client := NewClientWith(ClientConfig{
				Address:    DefaultAddress,
				BufferSize: MaxBufferSize,
			})

			measures := make([]stats.Measure, N)

			for i := range measures {
				measures[i] = stats.Measure{
					Name: "benchmark.test.metric",
					Fields: []stats.Field{
						{Name: "value", Value: stats.ValueOf(42)},
					},
					Tags: []stats.Tag{
						stats.T("answer", "42"),
						stats.T("hello", "world"),
					},
				}
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					client.HandleMeasures(time.Time{}, measures...)
				}
			})
		})
	}
}

// startUDPListener starts a goroutine listening for UDP packets on 127.0.0.1 and an available port.
// The address listened to is returned as `addr`. The payloads of packets received are copied to `packets`
func startUDPListener(t *testing.T, packets chan []byte) (addr string, closer io.Closer) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0") // :0 chooses an available port
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			packetBytes := make([]byte, 1024)
			n, _, err := conn.ReadFrom(packetBytes)
			if n > 0 {
				packets <- packetBytes[:n]
			}

			if err != nil {
				t.Log(err)
				return
			}
		}
	}()

	return conn.LocalAddr().String(), conn
}
