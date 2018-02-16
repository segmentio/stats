package netstats

import (
	"net"
	"reflect"
	"testing"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestListener(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("netstats.test", h)

	lstn := NewListenerWith(e, testLstn{})

	conn, err := lstn.Accept()
	if err != nil {
		t.Error(err)
		return
	}

	conn.Close()
	lstn.Close()

	expected := []stats.Measure{
		{
			Name:   "netstats.test.conn.open",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("protocol", "tcp")},
		},
		{
			Name:   "netstats.test.conn.close",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("protocol", "tcp")},
		},
	}

	if !reflect.DeepEqual(expected, h.Measures()) {
		t.Error("bad measures:")
		t.Logf("expected: %v", expected)
		t.Logf("found:    %v", h.Measures())
	}
}

func TestListenerError(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("netstats.test", h)

	lstn := NewListenerWith(e, testLstn{err: errTest})

	_, err := lstn.Accept()
	if err != errTest {
		t.Error(err)
		return
	}

	lstn.Close()

	expected := []stats.Measure{
		{
			Name:   "netstats.test.conn.error",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("operation", "accept"), stats.T("protocol", "tcp")},
		},
	}

	if !reflect.DeepEqual(expected, h.Measures()) {
		t.Error("bad measures:")
		t.Logf("expected: %v", expected)
		t.Logf("found:    %v", h.Measures())
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
