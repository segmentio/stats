package netstats

import (
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestConn(t *testing.T) {
	now := time.Now()

	backend := &stats.EventBackend{}
	client := stats.NewClientWith(stats.Config{
		Backend: backend,
		Scope:   "test",
		Now:     func() time.Time { return now },
	})
	defer client.Close()

	conn := NewConn(&testConn{}, client)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 10))
	conn.Read(make([]byte, 10))
	conn.Close()

	events := []stats.Event{
		// Write
		{
			Type:   "histogram",
			Name:   "test.conn.iops",
			Value:  12,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "write"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.bytes.count",
			Value:  12,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "write"}},
			Time:   now,
		},

		// Read
		{
			Type:   "histogram",
			Name:   "test.conn.iops",
			Value:  10,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "read"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.bytes.count",
			Value:  10,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "read"}},
			Time:   now,
		},

		// Read
		{
			Type:   "histogram",
			Name:   "test.conn.iops",
			Value:  2,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "read"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.bytes.count",
			Value:  2,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "read"}},
			Time:   now,
		},
	}

	if !reflect.DeepEqual(backend.Events, events) {
		t.Errorf("\n- %#v\n- %#v", events, backend.Events)
	}
}

func TestConnError(t *testing.T) {
	now := time.Now()

	backend := &stats.EventBackend{}
	client := stats.NewClientWith(stats.Config{
		Backend: backend,
		Scope:   "test",
		Now:     func() time.Time { return now },
	})
	defer client.Close()

	conn := NewConn(&testConn{err: testError}, client)
	conn.SetDeadline(now)
	conn.SetReadDeadline(now)
	conn.SetWriteDeadline(now)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 10))
	conn.Read(make([]byte, 10))
	conn.Close()

	events := []stats.Event{
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "set-timeout"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "set-read-timeout"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "set-write-timeout"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "write"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "read"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "read"}},
			Time:   now,
		},
		{
			Type:   "counter",
			Name:   "test.conn.errors.count",
			Value:  1,
			Sample: 1,
			Tags:   stats.Tags{{"protocol", "tcp"}, {"local_address", "127.0.0.1:2121"}, {"remote_address", "127.0.0.1:4242"}, {"operation", "close"}},
			Time:   now,
		},
	}

	if !reflect.DeepEqual(backend.Events, events) {
		t.Errorf("\n- %#v\n- %#v", events, backend.Events)
	}
}

func TestRootError(t *testing.T) {
	e1 := &net.OpError{Err: io.EOF}
	e2 := rootError(e1)

	if e2 != io.EOF {
		t.Errorf("bad root error: %s", e2)
	}
}
