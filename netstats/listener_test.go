package netstats

import (
	"net"
	"reflect"
	"testing"

	"github.com/segmentio/stats"
)

func TestListener(t *testing.T) {
	h := &handler{}
	e := stats.NewDefaultEngine()
	e.Register(h)

	lstn := NewListenerEngine(e, testLstn{})

	conn, err := lstn.Accept()
	if err != nil {
		t.Error(err)
		return
	}

	conn.Close()
	lstn.Close()

	if !reflect.DeepEqual(h.metrics, []stats.Metric{
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.open.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     1,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.close.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestListenerError(t *testing.T) {
	h := &handler{}
	e := stats.NewDefaultEngine()
	e.Register(h)

	lstn := NewListenerEngine(e, testLstn{err: errTest})

	_, err := lstn.Accept()
	if err != errTest {
		t.Error(err)
		return
	}

	lstn.Close()

	if !reflect.DeepEqual(h.metrics, []stats.Metric{
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "accept"}},
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

type testLstn struct {
	conn testConn
	err  error
}

func (lstn testLstn) Accept() (net.Conn, error) {
	if lstn.err != nil {
		return nil, lstn.err
	}
	return &lstn.conn, nil
}

func (lstn testLstn) Close() error {
	return lstn.err
}

func (lstn testLstn) Addr() net.Addr {
	return testLocalAddr
}
