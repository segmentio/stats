package prometheus

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
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

func TestEnqueueSuccess(t *testing.T) {
	c := make(chan job, 1)
	j := job{
		metric: stats.NewGauge(stats.MakeOpts("test")),
		value:  1.0,
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
	}

	close(c)
	enqueue(j, c, func(err error) { e = err })

	if e == nil {
		t.Errorf("no error reported by an enqueue operation that should have failed")
	}
}

func TestRunComplete(t *testing.T) {
	now := time.Now()

	jobs := make(chan job, 3)
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	close(jobs)

	join := &sync.WaitGroup{}
	join.Add(1)

	store := newMetricStore()

	go run(jobs, store, join, &Config{
		ExpireTimeout: time.Second,
		Fail:          func(err error) { t.Error(err) },
		Now:           func() time.Time { return now },
	})

	join.Wait()

	if metrics := store.snapshot(); !reflect.DeepEqual(metrics, []Metric{
		Metric{
			Name:  "metric_1",
			Type:  "counter",
			Value: 3,
			Time:  now,
			key:   "metric_1",
			sum:   3,
			count: 3,
		},
	}) {
		t.Errorf("invalid metric snapshot: %v", metrics)
	}
}

func TestRunExpire(t *testing.T) {
	jobs := make(chan job, 3)
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}

	join := &sync.WaitGroup{}
	join.Add(1)

	store := newMetricStore()

	go run(jobs, store, join, &Config{
		ExpireTimeout: 100 * time.Microsecond,
		Fail:          func(err error) { t.Error(err) },
		Now:           time.Now,
	})

	time.Sleep(time.Millisecond)
	close(jobs)

	join.Wait()

	if metrics := store.snapshot(); !reflect.DeepEqual(metrics, []Metric{}) {
		t.Errorf("invalid metric snapshot: %v", metrics)
	}
}

func TestRunError(t *testing.T) {
	now := time.Now()

	jobs := make(chan job, 3)
	jobs <- job{
		metric: stats.NewGauge(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	jobs <- job{
		metric: stats.NewCounter(stats.Opts{Name: "metric_1"}),
		value:  1,
	}
	close(jobs)

	join := &sync.WaitGroup{}
	join.Add(1)

	store := newMetricStore()
	e := error(nil)

	go run(jobs, store, join, &Config{
		ExpireTimeout: time.Second,
		Fail:          func(err error) { e = err },
		Now:           func() time.Time { return now },
	})

	join.Wait()

	if e == nil {
		t.Error("no error was reported")
	}

	if metrics := store.snapshot(); !reflect.DeepEqual(metrics, []Metric{
		Metric{
			Name:  "metric_1",
			Type:  "gauge",
			Value: 1,
			Time:  now,
			key:   "metric_1",
			sum:   1,
			count: 1,
		},
	}) {
		t.Errorf("invalid metric snapshot: %v", metrics)
	}
}

func TestListenSuccess(t *testing.T) {
	done := make(chan struct{})
	close(done)

	lstn := listen(done, &Config{
		RetryAfterMin: time.Microsecond,
		RetryAfterMax: time.Microsecond,
		Address:       ":0",
	})

	if lstn == nil {
		t.Error("failed to open a network listener")
	} else {
		lstn.Close()
	}
}

func TestListenFailure(t *testing.T) {
	e := error(nil)

	done := make(chan struct{})
	close(done)

	lstn := listen(done, &Config{
		RetryAfterMin: time.Microsecond,
		RetryAfterMax: time.Microsecond,
		Address:       "...",
		Fail:          func(err error) { e = err },
	})

	if e == nil {
		t.Error("expected error when listen fails but non was returned")
	}

	if lstn != nil {
		lstn.Close()
		t.Error("unexpected non-nil listener returned when listen should have failed")
	}
}

func TestListenPreset(t *testing.T) {
	done := make(chan struct{})
	close(done)

	preset, _ := net.Listen("tcp", ":0")
	defer preset.Close()

	if lstn := listen(done, &Config{
		Listener: preset,
	}); lstn != preset {
		t.Error("an invalid listener was returned when one was preset in the configuration")
	}
}

func TestSetConfigDefaults(t *testing.T) {
	if config := setConfigDefaults(Config{}); reflect.DeepEqual(config, Config{}) {
		t.Error("setting config defaults didn't change the config value")
	}
}

func TestBackend(t *testing.T) {
	now := time.Unix(1, 0)

	a := "127.0.0.1:12345"
	b := NewBackendWith(Config{
		Address: a,
		Now:     func() time.Time { return now },
	})
	defer b.Close()

	b.Set(stats.NewGauge(stats.MakeOpts("metric_1")), 1)
	b.Add(stats.NewCounter(stats.MakeOpts("metric_2")), 2)
	b.Observe(stats.NewHistogram(stats.MakeOpts("metric_3")), time.Second)

	// give some time to the backend to start
	time.Sleep(10 * time.Millisecond)

	res, err := http.Get("http://" + a + "/metrics")
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}

	if s := string(content); s != `# TYPE metric_1 gauge
metric_1 1 1000

# TYPE metric_2 counter
metric_2 2 1000

# TYPE metric_3 histogram
metric_3_count 1 1000
metric_3_sum 1 1000
` {
		t.Errorf("invalid response received from backend:\n%s", s)
	}
}

func TestBackendClose(t *testing.T) {
	NewBackend(":0").Close()
}
