package netstats

import (
	"net"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestListener(t *testing.T) {
	engine := stats.NewDefaultEngine()
	defer engine.Close()

	lstn := NewListener(engine, testLstn{}, stats.Tag{"test", "listener"})

	conn, err := lstn.Accept()
	if err != nil {
		t.Error(err)
		return
	}

	conn.Close()

	// Give time to the engine to process the metrics.
	time.Sleep(10 * time.Millisecond)

	metrics := engine.State()
	sort.Sort(stats.MetricsByKey(metrics))

	for i := range metrics {
		metrics[i].Time = time.Time{} // reset because we can't predict that value
	}

	expects := []stats.Metric{
		stats.Metric{
			Type:   stats.CounterType,
			Key:    "netstats.test.conn.close.count?protocol=tcp&test=listener",
			Name:   "netstats.test.conn.close.count",
			Tags:   []stats.Tag{{"protocol", "tcp"}, {"test", "listener"}},
			Value:  1,
			Sample: 1,
		},
		stats.Metric{
			Type:   stats.CounterType,
			Key:    "netstats.test.conn.open.count?protocol=tcp&test=listener",
			Name:   "netstats.test.conn.open.count",
			Tags:   []stats.Tag{{"protocol", "tcp"}, {"test", "listener"}},
			Value:  1,
			Sample: 1,
		},
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad engine state:")

		for i := range metrics {
			m := metrics[i]
			e := expects[i]

			if !reflect.DeepEqual(m, e) {
				t.Logf("unexpected metric at index %d:\n<<< %#v\n>>> %#v", i, m, e)
			}
		}
	}
}

func TestListenerError(t *testing.T) {
	engine := stats.NewDefaultEngine()
	defer engine.Close()

	lstn := NewListener(engine, testLstn{err: errTest}, stats.Tag{"test", "listener"})

	conn, err := lstn.Accept()

	if conn != nil {
		t.Error(conn)
		return
	}

	if err != errTest {
		t.Error(err)
		return
	}

	lstn.Close()

	// Give time to the engine to process the metrics.
	time.Sleep(10 * time.Millisecond)

	metrics := engine.State()
	sort.Sort(stats.MetricsByKey(metrics))

	for i := range metrics {
		metrics[i].Time = time.Time{} // reset because we can't predict that value
	}

	expects := []stats.Metric{
		stats.Metric{
			Type:   stats.CounterType,
			Key:    "netstats.test.lstn.errors.count?operation=accept&protocol=tcp&test=listener",
			Name:   "netstats.test.lstn.errors.count",
			Tags:   []stats.Tag{{"operation", "accept"}, {"protocol", "tcp"}, {"test", "listener"}},
			Value:  1,
			Sample: 1,
		},
		stats.Metric{
			Type:   stats.CounterType,
			Key:    "netstats.test.lstn.errors.count?operation=close&protocol=tcp&test=listener",
			Name:   "netstats.test.lstn.errors.count",
			Tags:   []stats.Tag{{"operation", "close"}, {"protocol", "tcp"}, {"test", "listener"}},
			Value:  1,
			Sample: 1,
		},
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad engine state:")

		for i := range metrics {
			m := metrics[i]
			e := expects[i]

			if !reflect.DeepEqual(m, e) {
				t.Logf("unexpected metric at index %d:\n<<< %#v\n>>> %#v", i, m, e)
			}
		}
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
