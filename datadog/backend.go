package datadog

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/netstats"
)

type Config struct {
	Network       string
	Address       string
	BufferSize    int
	QueueSize     int
	RetryAfterMin time.Duration
	RetryAfterMax time.Duration
	FlushTimeout  time.Duration
	WriteTimeout  time.Duration
	SampleRate    float64
	Dial          func(string, string) (net.Conn, error)
	Fail          func(error)
}

func NewBackend(addr string) stats.Backend {
	network, address := netstats.SplitNetworkAddress(addr)
	return NewBackendWith(Config{
		Network: network,
		Address: address,
	})
}

func NewBackendWith(config Config) stats.Backend {
	config = setConfigDefaults(config)
	return netstats.NewBackendWith(netstats.Config{
		Protocol:      protocol{},
		Network:       config.Network,
		Address:       config.Address,
		BufferSize:    config.BufferSize,
		QueueSize:     config.QueueSize,
		RetryAfterMin: config.RetryAfterMin,
		RetryAfterMax: config.RetryAfterMax,
		FlushTimeout:  config.FlushTimeout,
		WriteTimeout:  config.WriteTimeout,
		SampleRate:    config.SampleRate,
		Dial:          config.Dial,
		Fail:          config.Fail,
	})
}

func setConfigDefaults(config Config) Config {
	if len(config.Network) == 0 {
		config.Network = "udp"
	}

	if len(config.Address) == 0 {
		config.Address = "localhost"
	}

	if _, port, _ := net.SplitHostPort(config.Address); len(port) == 0 {
		config.Address = net.JoinHostPort(config.Address, "8125")
	}

	return config
}

type protocol struct{}

func (p protocol) WriteSet(w io.Writer, m stats.Metric, v float64, r float64) error {
	return p.write("g", w, m, v, r)
}

func (p protocol) WriteAdd(w io.Writer, m stats.Metric, v float64, r float64) error {
	return p.write("c", w, m, v, r)
}

func (p protocol) WriteObserve(w io.Writer, m stats.Metric, v float64, r float64) error {
	return p.write("h", w, m, v, r)
}

func (p protocol) write(s string, w io.Writer, m stats.Metric, v float64, r float64) (err error) {
	_, err = fmt.Fprintf(w, "%s:%g|%s%v%v\n", sanitize(m.Name()), v, s, sample(r), tags(m.Tags()))
	return
}

type sample float64

func (sample sample) Format(f fmt.State, _ rune) {
	if sample != 1 {
		fmt.Fprintf(f, "|@%g", float64(sample))
	}
}

type tags stats.Tags

func (tags tags) Format(f fmt.State, _ rune) {
	if len(tags) != 0 {
		io.WriteString(f, "|#")

		for i, t := range tags {
			if i != 0 {
				io.WriteString(f, ",")
			}
			io.WriteString(f, sanitize(t.Name))
			io.WriteString(f, ":")
			io.WriteString(f, sanitize(t.Value))
		}
	}
}

func sanitize(s string) string {
	s = replace(s, ",")
	s = replace(s, ":")
	s = replace(s, "|")
	s = replace(s, "@")
	s = replace(s, "#")
	return s
}

func replace(s string, b string) string {
	if strings.IndexByte(s, b[0]) >= 0 {
		s = strings.Replace(s, b, "_", -1)
	}
	return s
}
