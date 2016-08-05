package net_stats

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

type Protocol interface {
	WriteSet(w io.Writer, m stats.Metric, v float64, r float64) error

	WriteAdd(w io.Writer, m stats.Metric, v float64, r float64) error

	WriteObserve(w io.Writer, m stats.Metric, v time.Duration, r float64) error
}

type Config struct {
	Protocol      Protocol
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

func SplitNetworkAddress(addr string) (network string, address string) {
	if index := strings.Index(addr, "://"); index >= 0 {
		network, addr = addr[:index], addr[index+3:]
	}
	address = addr
	return
}

func NewBackendWith(config Config) stats.Backend {
	config = setConfigDefaults(config)

	jobs := make(chan job, config.QueueSize)
	done := make(chan struct{})

	b := &backend{
		fail: config.Fail,
		jobs: jobs,
		done: done,
	}

	b.join.Add(1)
	go run(done, jobs, &b.join, &config)

	return b
}

func setConfigDefaults(config Config) Config {
	if config.BufferSize == 0 {
		config.BufferSize = 512
	}

	if config.QueueSize == 0 {
		config.QueueSize = 1000
	}

	if config.RetryAfterMin == 0 {
		config.RetryAfterMin = 100 * time.Millisecond
	}

	if config.RetryAfterMax == 0 {
		config.RetryAfterMax = 15 * time.Second
	}

	if config.FlushTimeout == 0 {
		config.FlushTimeout = 5 * time.Second
	}

	if config.WriteTimeout == 0 {
		config.WriteTimeout = 1 * time.Second
	}

	if config.SampleRate == 0 {
		config.SampleRate = 1
	}

	if config.Dial == nil {
		config.Dial = net.Dial
	}

	if config.Fail == nil {
		config.Fail = makeFailFunc(os.Stderr)
	}

	return config
}

type writer func(Protocol, io.Writer, stats.Metric, interface{}, float64) error

type job struct {
	metric stats.Metric
	value  interface{}
	write  writer
}

type backend struct {
	join sync.WaitGroup
	jobs chan<- job
	done chan struct{}
	fail func(error)
}

func (b *backend) Close() (err error) {
	defer b.join.Wait()
	defer func() { recover() }()
	close(b.jobs)
	close(b.done)
	return
}

func (b *backend) Set(m stats.Metric, v float64) { b.enqueue(m, v, set) }

func (b *backend) Add(m stats.Metric, v float64) { b.enqueue(m, v, add) }

func (b *backend) Observe(m stats.Metric, v time.Duration) { b.enqueue(m, v, observe) }

func (b *backend) enqueue(m stats.Metric, v interface{}, w writer) {
	enqueue(job{
		metric: m,
		value:  v,
		write:  w,
	}, b.jobs, b.fail)
}

func enqueue(job job, jobs chan<- job, fail func(error)) {
	defer func() {
		if x := recover(); x != nil {
			fail(fmt.Errorf("discarding %s because the metric queue was closed", job.metric.Name()))
		}
	}()
	select {
	case jobs <- job:
	default:
		fail(fmt.Errorf("discarding %s because the metric queue is full", job.metric.Name()))
	}
}

func set(p Protocol, w io.Writer, m stats.Metric, v interface{}, r float64) error {
	return p.WriteSet(w, m, v.(float64), r)
}

func add(p Protocol, w io.Writer, m stats.Metric, v interface{}, r float64) error {
	return p.WriteAdd(w, m, v.(float64), r)
}

func observe(p Protocol, w io.Writer, m stats.Metric, v interface{}, r float64) error {
	return p.WriteObserve(w, m, v.(time.Duration), r)
}

func run(done <-chan struct{}, jobs <-chan job, join *sync.WaitGroup, config *Config) {
	var conn net.Conn

	defer join.Done()
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	buf := &bytes.Buffer{}
	buf.Grow(config.BufferSize)

	timer := time.NewTicker(config.FlushTimeout)
	defer timer.Stop()

	for {
		if conn == nil {
			conn = connect(done, config)
		}

		select {
		case job, open := <-jobs:
			if !open {
				conn = flush(conn, buf, config)
				return
			}

			if config.SampleRate == 1 || config.SampleRate > rand.Float64() {
				conn = write(conn, buf, job, config)
			}

		case <-timer.C:
			conn = flush(conn, buf, config)
		}
	}
}

func connect(done <-chan struct{}, config *Config) (conn net.Conn) {
	retryAfter := config.RetryAfterMin

	for {
		if conn = dial(config); conn != nil {
			return
		}

		select {
		case <-done:
			return
		default:
			retryAfter = sleep(retryAfter, config.RetryAfterMax)
		}
	}
}

func dial(config *Config) (conn net.Conn) {
	var err error

	if conn, err = config.Dial(config.Network, config.Address); err != nil {
		handleError(err, config)
	}

	return
}

func sleep(d time.Duration, max time.Duration) time.Duration {
	time.Sleep(d)
	return backoff(d, max)
}

func backoff(d time.Duration, max time.Duration) time.Duration {
	if d += d; d > max {
		d = max
	}
	return d
}

func write(conn net.Conn, buf *bytes.Buffer, job job, config *Config) net.Conn {
	n1 := buf.Len()

	if err := job.write(config.Protocol, buf, job.metric, job.value, config.SampleRate); err != nil {
		handleError(err, config)
		return conn
	}

	if n2 := buf.Len(); n2 >= config.BufferSize {
		if n1 == 0 {
			n1 = n2
		}
		conn = flushN(conn, buf, config, n1)
	}

	return conn
}

func flush(conn net.Conn, buf *bytes.Buffer, config *Config) net.Conn {
	return flushN(conn, buf, config, buf.Len())
}

func flushN(conn net.Conn, buf *bytes.Buffer, config *Config, n int) net.Conn {
	if conn != nil {
		var err error

		if err = conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout)); err == nil {
			_, err = conn.Write(buf.Next(n))
		}

		if err != nil {
			conn.Close()
			conn = nil
			handleError(err, config)
		}
	}

	return conn
}

func handleError(err error, config *Config) {
	defer handlePanic(os.Stderr)
	config.Fail(err)
}

func handlePanic(w io.Writer) {
	if v := recover(); v != nil {
		printPanic(w, v)
		printStack(w)
	}
}

func printPanic(w io.Writer, v interface{}) {
	fmt.Fprintf(w, "panic: %v [recovered]\n", v)
}

func printStack(w io.Writer) {
	stack := make([]byte, 32768)
	w.Write(stack[:runtime.Stack(stack, false)])
}

func makeFailFunc(w io.Writer) func(error) {
	return func(err error) { fmt.Fprintf(w, "stats: %s\n", err) }
}
