package prometheus

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/netstats"
)

type Config struct {
	Network       string
	Address       string
	QueueSize     int
	RetryAfterMin time.Duration
	RetryAfterMax time.Duration
	ExpireTimeout time.Duration
	Listener      net.Listener
	Fail          func(error)
	Now           func() time.Time
}

func NewBackend(addr string) stats.Backend {
	network, address := netstats.SplitNetworkAddress(addr)
	return NewBackendWith(Config{
		Network: network,
		Address: address,
	})
}

func NewBackendWith(config Config) stats.Backend {
	config = setConfigDefaults(config)

	store := newMetricStore()
	done := make(chan struct{})
	jobs := make(chan job, config.QueueSize)

	b := &backend{
		done: done,
		jobs: jobs,
		fail: config.Fail,
	}

	b.join.Add(1)
	go serve(done, store, &b.join, &config)

	b.join.Add(1)
	go run(jobs, store, &b.join, &config)

	return b
}

func setConfigDefaults(config Config) Config {
	if len(config.Address) == 0 {
		config.Address = ":9000"
	}

	if config.QueueSize == 0 {
		config.QueueSize = 1000
	}

	if config.ExpireTimeout == 0 {
		config.ExpireTimeout = 10 * time.Minute
	}

	if config.Fail == nil {
		config.Fail = makeFailFunc(os.Stderr)
	}

	if config.Now == nil {
		config.Now = time.Now
	}

	return config
}

type job struct {
	metric stats.Metric
	value  float64
	time   time.Time
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

func (b *backend) Set(m stats.Metric, v float64, t time.Time) { b.enqueue(m, v, t) }

func (b *backend) Add(m stats.Metric, v float64, t time.Time) { b.enqueue(m, v, t) }

func (b *backend) Observe(m stats.Metric, v float64, t time.Time) { b.enqueue(m, v, t) }

func (b *backend) enqueue(m stats.Metric, v float64, t time.Time) {
	enqueue(job{
		metric: m,
		value:  v,
		time:   t,
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

func serve(done <-chan struct{}, store *metricStore, join *sync.WaitGroup, config *Config) {
	defer join.Done()

	if lstn := listen(done, config); lstn != nil {
		defer lstn.Close()

		go http.Serve(lstn, NewHttpHandler(HandlerFunc(func(res ResponseWriter, req *http.Request) {
			for _, m := range store.snapshot() {
				res.WriteMetric(m)
			}
		})))

		<-done
	}
}

func listen(done <-chan struct{}, config *Config) net.Listener {
	retryAfter := config.RetryAfterMin

	if config.Listener != nil {
		return config.Listener
	}

	for {
		if lstn, err := net.Listen("tcp", config.Address); err == nil {
			return lstn
		} else {
			handleError(err, config)
		}

		retryAfter = sleep(done, retryAfter, config.RetryAfterMax)
		select {
		default:
		case <-done:
			return nil
		}
	}
}

func run(jobs <-chan job, store *metricStore, join *sync.WaitGroup, config *Config) {
	defer join.Done()

	timer := time.NewTicker(config.ExpireTimeout / 2)
	defer timer.Stop()

	for {
		select {
		case job, open := <-jobs:
			if !open {
				return
			}

			if err := store.insert(makeMetric(job.metric, job.value, job.time)); err != nil {
				handleError(err, config)
			}

		case <-timer.C:
			store.expire(config.Now().Add(-config.ExpireTimeout))
		}
	}
}

func sleep(done <-chan struct{}, d time.Duration, max time.Duration) time.Duration {
	select {
	case <-done:
	case <-time.After(d):
	}
	return backoff(d, max)
}

func backoff(d time.Duration, max time.Duration) time.Duration {
	if d += d; d > max {
		d = max
	}
	return d
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
