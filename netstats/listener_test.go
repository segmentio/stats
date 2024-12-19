package netstats

import (
	"net"
	"reflect"
	"testing"

	stats "github.com/segmentio/stats/v5"
	"github.com/segmentio/stats/v5/statstest"
	"github.com/segmentio/stats/v5/version"
)

func TestListener(t *testing.T) {
	initValue := stats.GoVersionReportingEnabled
	stats.GoVersionReportingEnabled = false
	defer func() { stats.GoVersionReportingEnabled = initValue }()
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

	measures := h.Measures()
	t.Run("CheckGoVersionEmitted", func(t *testing.T) {
		if version.DevelGoVersion() {
			t.Skip("No metrics emitted if compiled with Go devel version")
		}
		measurePassed := false
		for _, measure := range measures {
			if measure.Name != "go_version" {
				continue
			}
			for _, tag := range measure.Tags {
				if tag.Name != "go_version" {
					continue
				}
				if tag.Value == version.GoVersion() {
					measurePassed = true
				}
			}
		}
		if !measurePassed {
			t.Errorf("did not find correct 'go_version' tag for measure: %#v\n", measures)
		}
	})
	var foundMetric stats.Measure
	for i := range measures {
		if measures[i].Name == "netstats.test.conn.error" {
			foundMetric = measures[i]
			break
		}
	}
	if foundMetric.Name == "" {
		t.Errorf("did not find netstats metric: %v", measures)
	}

	expected := stats.Measure{
		Name:   "netstats.test.conn.error",
		Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
		Tags:   []stats.Tag{stats.T("operation", "accept"), stats.T("protocol", "tcp")},
	}
	if !reflect.DeepEqual(expected, foundMetric) {
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
