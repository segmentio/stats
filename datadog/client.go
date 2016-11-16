package datadog

import (
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

const (
	// DefaultAddress is the default address to which clients connection to.
	DefaultAddress = "localhost:8125"

	// DefaultBufferSize is the default size of the client buffer.
	DefaultBufferSize = 65507

	// DefaultFlushTimeout is the default interval at which clients flush
	// metrics from their stats engine.
	DefaultFlushInterval = 1 * time.Second
)

// The ClientConfig type is used to configure datadog clients.
type ClientConfig struct {
	// Engine is the stats engine that the datadog client will be reading
	// metrics from.
	// If Engine is nil the default stats engine is used.
	Engine *stats.Engine

	// Address of the dogstatsd agent to send metrics to.
	Address string

	// BufferSize is the size of the output buffer used by the client.
	BufferSize int

	// Output, if not nil, is a writer where the client will output the metrics
	// it collected.
	// If Output is nil the client will open a new UDP socket to Address.
	Output io.WriteCloser

	// FlushInterval configures how often the client reads metrics from the
	// stats engine and sends them to the dogstatsd agent.
	FlushInterval time.Duration
}

// Client represents a datadog client that pulls metrics from a stats engine and
// forward them to a dogstatsd agent.
type Client struct {
	once sync.Once
	done chan struct{}
	join chan struct{}
}

// NewDefaultClient creates and returns a new datadog client with a default
// configuration.
func NewDefaultClient() *Client {
	return NewClient(ClientConfig{})
}

// NewClient creates and returns a new datadog client configured with config.
func NewClient(config ClientConfig) *Client {
	if len(config.Address) == 0 {
		config.Address = DefaultAddress
	}

	if config.BufferSize == 0 {
		config.BufferSize = DefaultBufferSize
	}

	if config.FlushInterval == 0 {
		config.FlushInterval = DefaultFlushInterval
	}

	cli := &Client{
		done: make(chan struct{}),
		join: make(chan struct{}),
	}

	go run(config, time.NewTicker(config.FlushInterval), cli.done, cli.join)

	runtime.SetFinalizer(cli, func(c *Client) { c.Close() })
	return cli
}

// Close stops the client's internal timer and releases all allocated resources.
func (c *Client) Close() error {
	c.once.Do(c.close)
	return nil
}

func (c *Client) close() {
	close(c.done)
	<-c.join
}

func run(c ClientConfig, tick *time.Ticker, done <-chan struct{}, join chan<- struct{}) {
	defer close(join)
	defer tick.Stop()

	if c.Output == nil {
		var err error
		if c.Output, err = net.Dial("udp", c.Address); err != nil {
			log.Print(err)
			return
		}
	}

	defer c.Output.Close()

	var state []stats.Metric
	var changes []stats.Metric
	var cache = make(map[string]stats.Metric)

	b1 := make([]byte, 0, 1024)
	b2 := make([]byte, 0, c.BufferSize)

	// On each tick, fetch the sttate of the engine and write the metrics that
	// have changed since the last loop iteration.
mainLoop:
	for {
		select {
		case <-done:
			break mainLoop

		case <-tick.C:
			state, changes = diff(state, c.Engine.State(), cache, changes[:0])
			write(c.Output, b1, b2, changes)
		}
	}

	// Flush the engine state one last time before existing, this helps prevent
	// data loss when the program is shutting down and the engine had a couple
	// of pending changes.
	state, changes = diff(state, c.Engine.State(), cache, changes[:0])
	write(c.Output, b1, b2, changes)
}

func write(w io.Writer, b1 []byte, b2 []byte, changes []stats.Metric) {
	// Write all changed metrics to the client buffer in order to send
	// it to the datadog agent.
	for _, m := range changes {
		b1 = appendMetric(b1[:0], m)

		if len(b1) > cap(b2) {
			// The metric is too large to fit in the output buffer, we
			// simply write it straight to the output and hope for the
			// best (it'll likely be discarded because it's bigger than
			// what a UDP datagram can carry).
			w.Write(b1)
			continue
		}

		if (len(b1) + len(b2)) > cap(b2) {
			// The output buffer is full, flushing to the writer.
			w.Write(b2)
			b2 = b2[:0]
		}

		b2 = append(b2, b1...)
	}

	// Flush any remaining data in the output buffer.
	if len(b2) != 0 {
		w.Write(b2)
	}
}

// The diff function takes an old and new engine state and computes the
// differences between them, returing a list of metrics that have been
// changed.
func diff(old []stats.Metric, new []stats.Metric, cache map[string]stats.Metric, changed []stats.Metric) ([]stats.Metric, []stats.Metric) {
	// Populate the cache with all old metrics.
	for _, m := range old {
		cache[m.Key] = m
	}

	// Look for metrics that have changed since the last tick.
	for _, m := range new {
		if n, ok := cache[m.Key]; !ok || m.Sample != n.Sample {
			switch m.Type {
			case stats.CounterType:
				// For counters we need to report the difference since the last
				// tick and discards rate sampling.
				m.Value -= n.Value
				m.Sample = 0

			case stats.GaugeType:
				// For gages we already have the correct value, no need to do
				// rate sampling.
				m.Sample = 0
			}
			changed = append(changed, m)
		}
		delete(cache, m.Key)
	}

	// Clear the cache so it can be reused.
	for k := range cache {
		delete(cache, k)
	}

	return new, changed
}
