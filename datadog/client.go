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

	// DefaultFlushInterval is the default interval at which clients flush
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

	if config.Engine == nil {
		config.Engine = stats.DefaultEngine
	}

	cli := &Client{
		done: make(chan struct{}),
		join: make(chan struct{}),
	}

	engineConfig := config.Engine.Config()
	metricTimeout := 2 * engineConfig.MetricTimeout
	go run(config, time.NewTicker(config.FlushInterval), cli.done, cli.join, metricTimeout)

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

func run(c ClientConfig, tick *time.Ticker, done <-chan struct{}, join chan<- struct{}, timeout time.Duration) {
	defer close(join)
	defer tick.Stop()

	if c.Output == nil {
		var err error
		if c.Output, err = net.Dial("udp", c.Address); err != nil {
			log.Printf("stats/datadog: %s", err)
			return
		}
	}

	defer c.Output.Close()

	var version uint64                      // last version seen by the client
	var counters = make(map[string]counter) // cache of previous counter values
	var b1 = make([]byte, 0, 1024)
	var b2 = make([]byte, 0, c.BufferSize)

	// On each tick, fetch the state of the engine and write the metrics that
	// have changed since the last loop iteration.
mainLoop:
	for {
		select {
		case <-done:
			break mainLoop

		case now := <-tick.C:
			var state []stats.Metric
			state, version = c.Engine.State(version)
			write(c.Output, b1, b2, metrics(state, counters, now))

			for k, c := range counters { // clear expired counters
				if now.After(c.modTime.Add(timeout)) {
					delete(counters, k)
				}
			}
		}
	}

	// Flush the engine state one last time before exiting, this helps prevent
	// data loss when the program is shutting down and the engine had a couple
	// of pending changes.
	state, _ := c.Engine.State(version)
	write(c.Output, b1, b2, metrics(state, counters, time.Now()))
}

func write(w io.Writer, b1 []byte, b2 []byte, changes []Metric) {
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

type counter struct {
	value   float64
	modTime time.Time
}

func metrics(state []stats.Metric, counters map[string]counter, now time.Time) []Metric {
	// List of datadog metrics computed from the state.
	metrics := make([]Metric, 0, len(state))

	// Aggregation of histograms into a single value.
	histograms := make(map[string]Metric)

	for _, m := range state {
		switch m.Type {
		case stats.CounterType:
			// For counters the datadog client needs to report the difference of
			// value between now and the last time the counter was reported.
			value := m.Value - counters[m.Key].value

			// If the value is negative then we have an outdated entry in the
			// counter cache, we simply overwrite it with the new value.
			if value < 0 {
				value = m.Value
			}

			counters[m.Key] = counter{
				value:   m.Value,
				modTime: now,
			}

			metrics = append(metrics, Metric{
				Type:      Counter,
				Name:      m.Name,
				Value:     value,
				Tags:      m.Tags,
				Namespace: m.Namespace,
			})

		case stats.GaugeType:
			// Gauge always have the right value, we just place them in the
			// result list of metrics.
			metrics = append(metrics, Metric{
				Type:      Gauge,
				Name:      m.Name,
				Value:     m.Value,
				Tags:      m.Tags,
				Namespace: m.Namespace,
			})

		case stats.HistogramType:
			// Histograms need to be aggregated to report average values.
			h, ok := histograms[m.Key]

			if !ok {
				h = Metric{
					Type:      Histogram,
					Name:      m.Name,
					Tags:      m.Tags,
					Namespace: m.Namespace,
				}
			}

			h.Value += m.Value
			h.Rate += 1 // reuse the field to accumulate the number of samples
			histograms[m.Key] = h
		}
	}

	// Compute the average values and set the sample rate on histograms.
	for _, h := range histograms {
		h.Value /= h.Rate   // average value
		h.Rate = 1 / h.Rate // sample rate
		metrics = append(metrics, h)
	}

	return metrics
}
