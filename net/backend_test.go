package net_stats

import (
	"bufio"
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

func TestReset(t *testing.T) {
	c := &testConn{}
	b := bufio.NewWriter(c)

	if reset(c, b) != nil {
		t.Error("reset should always return nil")
	}
}

func TestFlushSuccess(t *testing.T) {
	c := &testConn{}
	b := bufio.NewWriter(c)
	b.WriteString("Hello World!")

	if flush(c, b, nil) != c {
		t.Error("flush should return the connection on success")
	}

	if s := c.String(); s != "Hello World!" {
		t.Errorf("flush did not output the right content: %s", s)
	}
}

func TestFlushFailure(t *testing.T) {
	e := error(nil)
	c := &testConn{err: io.EOF}
	b := bufio.NewWriter(c)
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
	const N = 512

	c := &testConn{}
	b := bufio.NewWriterSize(c, N)
	m := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  1.0,
		write:  set,
	}

	if write(c, b, m, j, &Config{
		Protocol:   testProto{},
		BufferSize: N,
	}) != c {
		t.Error("write should return the connection on success")
	}

	if s := c.String(); len(s) != 0 {
		t.Errorf("write shouldn't have flushed when there was enough room in the buffer: %#v", s)
	}

	if s := m.String(); s != "set:test:1\n" {
		t.Error("write should have written to the message buffer")
	}

	if n := b.Buffered(); n != 11 {
		t.Errorf("the connection buffer contains a wrong number of bytes: %d", n)
	}
}

func TestWriteSuccessFlush(t *testing.T) {
	const N = 20

	c := &testConn{}
	b := bufio.NewWriterSize(c, N)
	m := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  10.0,
		write:  set,
	}

	b.WriteString("set:test:0\n")

	if write(c, b, m, j, &Config{
		Protocol:   testProto{},
		BufferSize: N,
	}) != c {
		t.Error("write should return the connection on success")
	}

	if s := c.String(); s != "set:test:0\n" {
		t.Error("write should have flushed the initial buffer to the connection")
	}

	if s := m.String(); s != "set:test:10\n" {
		t.Error("write should have written to the message buffer")
	}

	if n := b.Buffered(); n != 12 {
		t.Errorf("the connection buffer contains a wrong number of bytes: %d", n)
	}
}

func TestWriteSuccessNoBuffer(t *testing.T) {
	const N = 10

	c := &testConn{}
	b := bufio.NewWriterSize(c, N)
	m := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  10.0,
		write:  set,
	}

	if write(c, b, m, j, &Config{
		Protocol:   testProto{},
		BufferSize: N,
	}) != c {
		t.Error("write should return the connection on success")
	}

	if s := c.String(); s != "set:test:10\n" {
		t.Error("write should have flushed directly to the connection")
	}

	if s := m.String(); s != "set:test:10\n" {
		t.Error("write should have written to the message buffer")
	}

	if n := b.Buffered(); n != 0 {
		t.Error("the connection buffer should be empty")
	}
}

func TestWriteFailureProtocol(t *testing.T) {
	const N = 10

	e := error(nil)
	c := &testConn{}
	b := bufio.NewWriterSize(c, N)
	m := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  10.0,
		write:  set,
	}

	if write(c, b, m, j, &Config{
		Protocol:   testProto{err: io.EOF},
		BufferSize: N,
		Fail:       func(err error) { e = err },
	}) != c {
		t.Error("write should return the connection on protocol failures")
	}

	if e != io.EOF {
		t.Errorf("the wrong error was reported to the error handler: %s", e)
	}
}

func TestWriteFailureConn(t *testing.T) {
	const N = 10

	e := error(nil)
	c := &testConn{err: io.EOF}
	b := bufio.NewWriterSize(c, N)
	m := &bytes.Buffer{}
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  10.0,
		write:  set,
	}

	if write(c, b, m, j, &Config{
		Protocol:   testProto{},
		BufferSize: N,
		Fail:       func(err error) { e = err },
	}) != nil {
		t.Error("write should return nil on connection failure")
	}

	if e != io.EOF {
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
		if d := sleep(test.in, test.max); d != test.out {
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
		Dial: func(_ string, _ string) (net.Conn, error) { return nil, io.EOF },
		Fail: func(err error) { e = err },
	}); c != nil {
		t.Errorf("dial returned an invalid connection: %v", c)
	}

	if e != io.EOF {
		t.Errorf("the error handler was called with an invalid error: %s", e)
	}
}

func TestConnect(t *testing.T) {
	conn := &testConn{}

	dialSuccess := func(_ string, _ string) (net.Conn, error) { return conn, nil }
	dialFailure := func(_ string, _ string) (net.Conn, error) { return nil, io.EOF }

	config := &Config{
		RetryAfterMin: time.Microsecond,
		RetryAfterMax: time.Microsecond,
		Dial:          dialFailure,
	}
	config.Fail = func(err error) {
		if err != io.EOF {
			t.Errorf("the error handler was called with an invalid error: %s", err)
		}
		config.Dial = dialSuccess
	}

	if c := connect(config); c != conn {
		t.Errorf("connect returned an invalid connection: %v", c)
	}
}

func TestRun(t *testing.T) {
	conn := &testConn{}

	jobs := make(chan job, 3)
	jobs <- job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  1.0,
		write:  set,
	}
	jobs <- job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  2.0,
		write:  add,
	}
	jobs <- job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
		value:  time.Second,
		write:  observe,
	}

	join := &sync.WaitGroup{}
	join.Add(1)

	go run(jobs, join, &Config{
		Protocol:      testProto{},
		RetryAfterMin: time.Second,
		RetryAfterMax: time.Second,
		FlushTimeout:  100 * time.Microsecond,
		Dial:          func(_ string, _ string) (net.Conn, error) { return conn, nil },
	})

	time.AfterFunc(time.Millisecond, func() { close(jobs) })
	join.Wait()

	if s := conn.String(); s != "set:test:1\nadd:test:2\nobserve:test:1s\n" {
		t.Errorf("run flushed invalid data to the connection: %s", s)
	}
}

func TestSetConfigDefaults(t *testing.T) {
	if config := setConfigDefaults(Config{}); reflect.DeepEqual(config, Config{}) {
		t.Error("setting config defaults didn't change the config value")
	}
}

func TestEnqueueSuccess(t *testing.T) {
	c := make(chan job, 1)
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
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
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
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
		metric: stats.NewGauge(stats.MakeOpts("test", ""), nil),
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

	b.Set(stats.NewGauge(stats.MakeOpts("test", ""), nil), 1)
	b.Add(stats.NewGauge(stats.MakeOpts("test", ""), nil), 2)
	b.Observe(stats.NewGauge(stats.MakeOpts("test", ""), nil), time.Second)

	time.Sleep(time.Millisecond)

	b.Close()

	if s := conn.String(); s != "set:test:1\nadd:test:2\nobserve:test:1s\n" {
		t.Errorf("run flushed invalid data to the connection: %s", s)
	}
}

type testProto struct {
	err error
}

func (p testProto) WriteSet(w io.Writer, m stats.Metric, v float64) error {
	return p.write("set", w, m, v)
}

func (p testProto) WriteAdd(w io.Writer, m stats.Metric, v float64) error {
	return p.write("add", w, m, v)
}

func (p testProto) WriteObserve(w io.Writer, m stats.Metric, v time.Duration) (err error) {
	return p.write("observe", w, m, v)
}

func (p testProto) write(s string, w io.Writer, m stats.Metric, v interface{}) (err error) {
	if p.err != nil {
		return p.err
	}
	_, err = fmt.Fprintf(w, "%s:%s:%v\n", s, m.Name(), v)
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
