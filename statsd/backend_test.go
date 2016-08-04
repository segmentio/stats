package statsd

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestSetConfigDefaults(t *testing.T) {
	config := setConfigDefaults(Config{})

	if config.Network != "udp" {
		t.Error("invalid default network:", config.Network)
	}

	if config.Address != "localhost:8125" {
		t.Error("invalid default address:", config.Address)
	}
}

func TestBackend(t *testing.T) {
	packets := []string{}

	server, _ := net.ListenUDP("udp4", &net.UDPAddr{
		IP: net.IPv4(127, 0, 0, 1),
	})
	defer server.Close()
	addr := server.LocalAddr()
	join := make(chan struct{})

	go func() {
		defer close(join)
		var b [512]byte

		if n, err := server.Read(b[:]); err != nil {
			t.Error(err)
		} else {
			packets = append(packets, string(b[:n]))
		}
	}()

	a := addr.Network() + "://" + addr.String()
	c := stats.NewClient("statsd", NewBackend(a))
	c.Gauge(stats.Opts{Name: "events", Unit: "level"}).Set(1)
	c.Counter(stats.Opts{Name: "events", Unit: "count"}).Add(1)
	c.Histogram(stats.Opts{Name: "events", Unit: "duration"}).Observe(time.Second)
	c.Close()

	select {
	case <-join:
	case <-time.After(1 * time.Second):
		t.Error("timeout!")
	}

	if !reflect.DeepEqual(packets, []string{
		`statsd.events.level:1|g
statsd.events.count:1|c
statsd.events.duration:1000|h
`,
	}) {
		t.Errorf("invalid packets transmitted by the statsd client: %#v", packets)
	}
}
