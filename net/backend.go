package net_stats

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

type Protocol interface {
	WriteSet(w io.Writer, m stats.Metric, v float64) error

	WriteAdd(w io.Writer, m stats.Metric, v float64) error

	WriteObserve(w io.Writer, m stats.Metric, v time.Duration) error
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
	Dial          func(string, string) (net.Conn, error)
	Fail          func(error)
}

func NewBackendWith(config Config) stats.Backend {
	config = setConfigDefaults(config)

	j := make(chan job, config.QueueSize)
	b := &backend{
		jobs: j,
		fail: config.Fail,
	}

	b.join.Add(1)
	go run(j, &b.join, &config)

	return b
}

func setConfigDefaults(config Config) Config {
	if config.BufferSize == 0 {
		config.BufferSize = 512
	}

	if config.QueueSize == 0 {
		config.QueueSize = 1000
	}

	if config.Dial == nil {
		config.Dial = net.Dial
	}

	if config.Fail == nil {
		config.Fail = makeFailFunc(os.Stderr)
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

	return config
}

type writer func(Protocol, io.Writer, stats.Metric, interface{}) error

type job struct {
	metric stats.Metric
	value  interface{}
	write  writer
}

type backend struct {
	join sync.WaitGroup
	jobs chan<- job
	fail func(error)
}

func (b *backend) Close() (err error) {
	defer b.join.Wait()
	defer func() { recover() }()
	close(b.jobs)
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

func set(p Protocol, w io.Writer, m stats.Metric, v interface{}) error {
	return p.WriteSet(w, m, v.(float64))
}

func add(p Protocol, w io.Writer, m stats.Metric, v interface{}) error {
	return p.WriteAdd(w, m, v.(float64))
}

func observe(p Protocol, w io.Writer, m stats.Metric, v interface{}) error {
	return p.WriteObserve(w, m, v.(time.Duration))
}

func run(jobs <-chan job, join *sync.WaitGroup, config *Config) {
	var conn net.Conn

	defer join.Done()
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	cbuf := bufio.NewWriterSize(nil, config.BufferSize)
	mbuf := &bytes.Buffer{}
	mbuf.Grow(config.BufferSize)

	timer := time.NewTicker(config.FlushTimeout)
	defer timer.Stop()

	for {
		if conn == nil {
			conn = connect(config)
			cbuf.Reset(conn)
		}

		select {
		case job, open := <-jobs:
			if !open {
				conn = flush(conn, cbuf, config)
				return
			}

			mbuf.Reset()
			conn = write(conn, cbuf, mbuf, job, config)

		case <-timer.C:
			conn = flush(conn, cbuf, config)
		}
	}
}

func connect(config *Config) (conn net.Conn) {
	retryAfter := config.RetryAfterMin
	for {
		if conn = dial(config); conn == nil {
			retryAfter = sleep(retryAfter, config.RetryAfterMax)
		} else {
			return
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

func write(conn net.Conn, cbuf *bufio.Writer, mbuf *bytes.Buffer, job job, config *Config) net.Conn {
	var err error

	if err = job.write(config.Protocol, mbuf, job.metric, job.value); err != nil {
		handleError(err, config)
		return conn
	}

	if err = conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout)); err == nil {
		if (mbuf.Len() + cbuf.Buffered()) > config.BufferSize {
			conn = flush(conn, cbuf, config)
		}

		if mbuf.Len() >= config.BufferSize {
			_, err = conn.Write(mbuf.Bytes())
		} else {
			_, err = cbuf.Write(mbuf.Bytes())
		}
	}

	if err != nil {
		conn = reset(conn, cbuf)
		handleError(err, config)
	}

	return conn
}

func flush(conn net.Conn, cbuf *bufio.Writer, config *Config) net.Conn {
	if conn != nil {
		var err error

		if err = conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout)); err == nil {
			err = cbuf.Flush()
		}

		if err != nil {
			conn = reset(conn, cbuf)
			handleError(err, config)
		}
	}

	return conn
}

func reset(conn net.Conn, cbuf *bufio.Writer) net.Conn {
	if conn != nil {
		cbuf.Reset(nil)
		conn.Close()
	}
	return nil
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
