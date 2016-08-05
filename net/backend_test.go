package net_stats

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestSplitNetworkAddress(t *testing.T) {
	tests := []struct {
		input   string
		network string
		address string
	}{
		{
			input:   "",
			network: "",
			address: "",
		},
		{
			input:   "tcp://",
			network: "tcp",
			address: "",
		},
		{
			input:   "localhost",
			network: "",
			address: "localhost",
		},
		{
			input:   "tcp://localhost",
			network: "tcp",
			address: "localhost",
		},
	}

	for _, test := range tests {
		network, address := SplitNetworkAddress(test.input)

		if network != test.network {
			t.Errorf("%s: invalid network returned: %#v != %#v", test.input, test.network, network)
		}

		if address != test.address {
			t.Errorf("%s: invalid address returned: %#v != %#v", test.input, test.address, address)
		}
	}
}

func TestMakeFailFunc(t *testing.T) {
	b := &bytes.Buffer{}
	f := makeFailFunc(b)
	f(errors.New("A"))

	if s := b.String(); s != "stats: A\n" {
		t.Errorf("invalid content written by the default error handler: %#v", s)
	}
}

func TestPrintStack(t *testing.T) {
	b := &bytes.Buffer{}
	printStack(b)

	if b.Len() == 0 {
		t.Error("printStack did not output any content")
	}
}

func TestPrintPanic(t *testing.T) {
	b := &bytes.Buffer{}
	printPanic(b, "oops!")

	if s := b.String(); s != "panic: oops! [recovered]\n" {
		t.Errorf("printStack did not output any content: %s", s)
	}
}

func TestHandlePanic(t *testing.T) {
	b := &bytes.Buffer{}
	defer func() {
		if s := b.String(); !strings.HasPrefix(s, "panic: oops! [recovered]\n") {
			t.Errorf("handlePanic did not output the right content: %s", s)
		}
	}()
	defer handlePanic(b)
	panic("oops!")
}

func TestHandleError(t *testing.T) {
	e1 := errors.New("")
	e2 := error(nil)

	handleError(e1, &Config{
		Fail: func(err error) { e2 = err },
	})

	if e1 != e2 {
		t.Errorf("%s != %s", e1, e2)
	}
}

func TestFlushSuccess(t *testing.T) {
	c := &testConn{}
	b := &bytes.Buffer{}
	b.WriteString("Hello World!")

	if flush(c, b, &Config{}) != c {
		t.Error("flush should return the connection on success")
	}

	if s := c.String(); s != "Hello World!" {
		t.Errorf("flush did not output the right content: %s", s)
	}
}

func TestFlushFailure(t *testing.T) {
	e := error(nil)
	c := &testConn{err: testError}
	b := &bytes.Buffer{}
	b.WriteString("Hello World!")

	if flush(c, b, &Config{
		Fail: func(err error) { e = err },
	}) != nil {
		t.Error("flush should return nil on failure")
	}

	if e != c.err {
		t.Errorf("the error handler was called with an invalid error: %s != %s", c.err, e)
	}
}

func TestWriteSuccessNoFlush(t *testing.T) {
	c := &testConn{}
	b := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
		write:  set,
	}

	if write(c, b, j, &Config{
		Protocol:   testProto{},
		BufferSize: 512,
		SampleRate: 1,
	}) != c {
		t.Error("write should return the connection on success")
	}

	if s := c.String(); len(s) != 0 {
		t.Errorf("write shouldn't have flushed when there was enough room in the buffer: %#v", s)
	}

	if s := b.String(); s != "set:test:1/1\n" {
		t.Errorf("the connection buffer contains invalid data: %s", s)
	}
}

func TestWriteSuccessFlush(t *testing.T) {
	c := &testConn{}
	b := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  10.0,
		write:  set,
	}

	b.WriteString("set:test:0/1\n")

	if write(c, b, j, &Config{
		Protocol:   testProto{},
		BufferSize: 20,
		SampleRate: 1,
	}) != c {
		t.Error("write should return the connection on success")
	}

	if s := c.String(); s != "set:test:0/1\n" {
		t.Error("write should have flushed the initial buffer to the connection")
	}

	if s := b.String(); s != "set:test:10/1\n" {
		t.Errorf("the connection buffer contains invalid data: %s", s)
	}
}

func TestWriteSuccessNoBuffer(t *testing.T) {
	c := &testConn{}
	b := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  10.0,
		write:  set,
	}

	if write(c, b, j, &Config{
		Protocol:   testProto{},
		BufferSize: 10,
		SampleRate: 1,
	}) != c {
		t.Error("write should return the connection on success")
	}

	if s := c.String(); s != "set:test:10/1\n" {
		t.Error("write should have flushed directly to the connection")
	}

	if s := b.String(); len(s) != 0 {
		t.Error("the connection buffer should be empty")
	}
}

func TestWriteFailureProtocol(t *testing.T) {
	e := error(nil)
	c := &testConn{}
	b := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  10.0,
		write:  set,
	}

	if write(c, b, j, &Config{
		Protocol:   testProto{err: testError},
		BufferSize: 10,
		SampleRate: 1,
		Fail:       func(err error) { e = err },
	}) != c {
		t.Error("write should return the connection on protocol failures")
	}

	if e != testError {
		t.Errorf("the wrong error was reported to the error handler: %s", e)
	}
}

func TestWriteFailureConn(t *testing.T) {
	e := error(nil)
	c := &testConn{err: testError}
	b := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  10.0,
		write:  set,
	}

	if write(c, b, j, &Config{
		Protocol:   testProto{},
		BufferSize: 10,
		SampleRate: 1,
		Fail:       func(err error) { e = err },
	}) != nil {
		t.Error("write should return nil on connection failure")
	}

	if e != testError {
		t.Errorf("the wrong error was reported to the error handler: %s", e)
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		in  time.Duration
		out time.Duration
		max time.Duration
	}{
		{
			in:  1 * time.Second,
			out: 2 * time.Second,
			max: 15 * time.Second,
		},
		{
			in:  8 * time.Second,
			out: 15 * time.Second,
			max: 15 * time.Second,
		},
	}

	for _, test := range tests {
		if d := backoff(test.in, test.max); d != test.out {
			t.Errorf("backoff(%s, %s): %s != %s", test.in, test.max, test.out, d)
		}
	}
}

func TestSleep(t *testing.T) {
	tests := []struct {
		in  time.Duration
		out time.Duration
		max time.Duration
	}{
		{
			in:  1 * time.Microsecond,
			out: 2 * time.Microsecond,
			max: 15 * time.Microsecond,
		},
		{
			in:  8 * time.Microsecond,
			out: 15 * time.Microsecond,
			max: 15 * time.Microsecond,
		},
	}

	for _, test := range tests {
		if d := sleep(nil, test.in, test.max); d != test.out {
			t.Errorf("sleep(%s, %s): %s != %s", test.in, test.max, test.out, d)
		}
	}

}

func TestDialSuccess(t *testing.T) {
	conn := &testConn{}

	if c := dial(&Config{
		Network: "tcp",
		Address: "localhost",
		Dial: func(network string, address string) (net.Conn, error) {
			if network != "tcp" {
				t.Errorf("dial passed an invalid network: %s", network)
			}
			if address != "localhost" {
				t.Errorf("dial passed an invalid address: %s", address)
			}
			return conn, nil
		},
	}); c != conn {
		t.Errorf("dial returned an invalid connection: %v", c)
	}
}

func TestDialFailure(t *testing.T) {
	e := error(nil)

	if c := dial(&Config{
		Dial: func(_ string, _ string) (net.Conn, error) { return nil, testError },
		Fail: func(err error) { e = err },
	}); c != nil {
		t.Errorf("dial returned an invalid connection: %v", c)
	}

	if e != testError {
		t.Errorf("the error handler was called with an invalid error: %s", e)
	}
}

func TestConnectComplete(t *testing.T) {
	conn := &testConn{}
	done := make(chan struct{})

	dialSuccess := func(_ string, _ string) (net.Conn, error) { return conn, nil }
	dialFailure := func(_ string, _ string) (net.Conn, error) { return nil, testError }

	config := &Config{
		RetryAfterMin: time.Microsecond,
		RetryAfterMax: time.Microsecond,
		Dial:          dialFailure,
	}
	config.Fail = func(err error) {
		if err != testError {
			t.Errorf("the error handler was called with an invalid error: %s", err)
		}
		config.Dial = dialSuccess
	}

	if c := connect(done, config); c != conn {
		t.Errorf("connect returned an invalid connection: %v", c)
	}
}

func TestConnectAbort(t *testing.T) {
	done := make(chan struct{})
	close(done)

	config := &Config{
		RetryAfterMin: time.Microsecond,
		RetryAfterMax: time.Microsecond,
		Dial:          func(_ string, _ string) (net.Conn, error) { return nil, testError },
		Fail:          func(err error) {},
	}

	if c := connect(done, config); c != nil {
		t.Errorf("connect returned an invalid connection: %v", c)
	}
}

func TestRunComplete(t *testing.T) {
	conn := &testConn{}
	done := make(chan struct{})
	jobs := make(chan job, 3)
	jobs <- job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
		write:  set,
	}
	jobs <- job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  2.0,
		write:  add,
	}
	jobs <- job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
		write:  observe,
	}

	join := &sync.WaitGroup{}
	join.Add(1)

	go run(done, jobs, join, &Config{
		Protocol:      testProto{},
		RetryAfterMin: time.Second,
		RetryAfterMax: time.Second,
		FlushTimeout:  100 * time.Microsecond,
		SampleRate:    1,
		Dial:          func(_ string, _ string) (net.Conn, error) { return conn, nil },
	})

	time.AfterFunc(time.Millisecond, func() {
		close(jobs)
		close(done)
	})
	join.Wait()

	if s := conn.String(); s != "set:test:1/1\nadd:test:2/1\nobserve:test:1/1\n" {
		t.Errorf("run flushed invalid data to the connection: %s", s)
	}
}

func TestRunNoConnect(t *testing.T) {
	conn := &testConn{}
	done := make(chan struct{})
	jobs := make(chan job, 3)

	join := &sync.WaitGroup{}
	join.Add(1)

	close(jobs)
	close(done)

	go run(done, jobs, join, &Config{
		Protocol:      testProto{},
		RetryAfterMin: time.Second,
		RetryAfterMax: time.Second,
		FlushTimeout:  100 * time.Microsecond,
		SampleRate:    1,
		Dial:          func(_ string, _ string) (net.Conn, error) { return conn, nil },
	})

	join.Wait()
}

func TestSetConfigDefaults(t *testing.T) {
	if config := setConfigDefaults(Config{}); reflect.DeepEqual(config, Config{}) {
		t.Error("setting config defaults didn't change the config value")
	}
}

func TestEnqueueSuccess(t *testing.T) {
	c := make(chan job, 1)
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
		write:  nil,
	}

	enqueue(j, c, nil)

	if x := <-c; !reflect.DeepEqual(j, x) {
		t.Errorf("invalid job found after enqueing in channel: %#v", x)
	}
}

func TestEnqueueFailureFull(t *testing.T) {
	e := error(nil)
	c := make(chan job, 0)
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
		write:  nil,
	}

	enqueue(j, c, func(err error) { e = err })

	if e == nil {
		t.Errorf("no error reported by an enqueue operation that should have failed")
	}
}

func TestEnqueueFailureClosed(t *testing.T) {
	e := error(nil)
	c := make(chan job, 1)
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
		write:  nil,
	}

	close(c)
	enqueue(j, c, func(err error) { e = err })

	if e == nil {
		t.Errorf("no error reported by an enqueue operation that should have failed")
	}
}

func TestBackend(t *testing.T) {
	conn := &testConn{}

	b := NewBackendWith(Config{
		Protocol:      testProto{},
		RetryAfterMin: time.Second,
		RetryAfterMax: time.Second,
		FlushTimeout:  100 * time.Microsecond,
		Dial:          func(_ string, _ string) (net.Conn, error) { return conn, nil },
	})

	b.Set(stats.NewGauge(stats.MakeOpts("test")), 1)
	b.Add(stats.NewCounter(stats.MakeOpts("test")), 2)
	b.Observe(stats.NewHistogram(stats.MakeOpts("test")), 1)

	time.Sleep(time.Millisecond)

	b.Close()

	if s := conn.String(); s != "set:test:1/1\nadd:test:2/1\nobserve:test:1/1\n" {
		t.Errorf("run flushed invalid data to the connection: %s", s)
	}
}

type testProto struct {
	err error
}

func (p testProto) WriteSet(w io.Writer, m stats.Metric, v float64, r float64) error {
	return p.write("set", w, m, v, r)
}

func (p testProto) WriteAdd(w io.Writer, m stats.Metric, v float64, r float64) error {
	return p.write("add", w, m, v, r)
}

func (p testProto) WriteObserve(w io.Writer, m stats.Metric, v float64, r float64) (err error) {
	return p.write("observe", w, m, v, r)
}

func (p testProto) write(s string, w io.Writer, m stats.Metric, v float64, r float64) (err error) {
	if p.err != nil {
		return p.err
	}
	_, err = fmt.Fprintf(w, "%s:%s:%g/%g\n", s, m.Name(), v, r)
	return
}

type testConn struct {
	bytes.Buffer
	err error
}

func (c *testConn) Read(b []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.Buffer.Read(b)
}

func (c *testConn) Write(b []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.Buffer.Write(b)
}

func (c *testConn) Close() error { return nil }

func (c *testConn) LocalAddr() net.Addr { return testAddr{} }

func (c *testConn) RemoteAddr() net.Addr { return testAddr{} }

func (c *testConn) SetDeadline(_ time.Time) error { return nil }

func (c *testConn) SetReadDeadline(_ time.Time) error { return nil }

func (c *testConn) SetWriteDeadline(_ time.Time) error { return nil }

type testAddr struct{}

func (_ testAddr) Network() string { return "tcp" }

func (_ testAddr) String() string { return "localhost" }

var testError = errors.New("test")
