package influxdb

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/segmentio/stats"
)

const (
	// DefaultAddress is the default address to which the InfluxDB client tries
	// to connect to.
	DefaultAddress = "localhost:8086"

	// DefaultDatabase is the default database used by the InfluxDB client.
	DefaultDatabase = "stats"

	// DefaultBatchSize is the default size for batches of metrics sent to
	// InfluxDB.
	DefaultBatchSize = 1000

	// DefaultFlushInterval is the default value used to configure the interval
	// at which batches of metrics are flushed to InfluxDB.
	DefaultFlushInterval = 10 * time.Second

	// DefaultTimeout is the default timeout value used when sending requests to
	// InfluxDB.
	DefaultTimeout = 5 * time.Second
)

// The ClientConfig type is used to configure InfluxDB clients.
type ClientConfig struct {
	// Address of the InfluxDB database to send metrics to.
	Address string

	// Name of the InfluxDB database to send metrics to.
	Database string

	// Maximum size of batch of events sent to InfluxDB.
	BatchSize int

	FlushInterval time.Duration

	// Maximum amount of time that requests to InfluxDB may take.
	Timeout time.Duration

	// Transport configures the HTTP transport used by the client to send
	// requests to InfluxDB. By default http.DefaultTransport is used.
	Transport http.RoundTripper
}

// Client represents an InfluxDB client that implements the stats.Handler
// interface.
type Client struct {
	url        *url.URL
	httpClient http.Client
	metrics    unsafe.Pointer
	pool       sync.Pool
	join       sync.WaitGroup
	once       sync.Once
	done       chan struct{}
	flushedAt  int64
}

// NewClient creates and returns a new InfluxDB client publishing metrics to the
// server running at addr.
func NewClient(addr string) *Client {
	return NewClientWith(ClientConfig{
		Address:       addr,
		FlushInterval: DefaultFlushInterval,
	})
}

// NewClientWith creates and returns a new InfluxDB client configured with the
// given config.
func NewClientWith(config ClientConfig) *Client {
	if len(config.Address) == 0 {
		config.Address = DefaultAddress
	}

	if len(config.Database) == 0 {
		config.Database = DefaultDatabase
	}

	if config.BatchSize == 0 {
		config.BatchSize = DefaultBatchSize
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	c := &Client{
		url: makeURL(config.Address, config.Database),
		httpClient: http.Client{
			Timeout:   config.Timeout,
			Transport: config.Transport,
		},
		pool: sync.Pool{New: func() interface{} { return newMetrics(config.BatchSize) }},
		done: make(chan struct{}),
	}

	if config.FlushInterval != 0 {
		go c.run(config.FlushInterval)
	}

	return c
}

// CreateDB creates a database named db in the InfluxDB server that the client
// was configured to send metrics to.
func (c *Client) CreateDB(db string) error {
	u := *c.url
	q := u.Query()
	q.Del("db")
	u.Path = "/query"
	u.RawQuery = q.Encode()

	r, err := c.httpClient.Post(u.String(), "application/x-www-form-urlencoded", strings.NewReader(
		fmt.Sprintf("q=CREATE DATABASE %q", db),
	))
	if err != nil {
		return err
	}
	return readResponse(r)
}

// HandleMetric satisfies the stats.Handler interface.
func (c *Client) HandleMetric(m *stats.Metric) {
	if !stats.TagsAreSorted(m.Tags) {
		stats.SortTags(m.Tags)
	}

	var mptr *metrics
	var flush bool
	var added bool
handleMetric:
	mptr = c.loadMetrics()

	for mptr == nil {
		mptr = c.acquireMetrics()
		if c.compareAndSwapMetrics(nil, mptr) {
			break
		}
		c.releaseMetrics(mptr)
		mptr = nil
	}

	flush, added = mptr.append(m)

	if !added {
		c.compareAndSwapMetrics(mptr, nil)
		goto handleMetric
	}

	if flush {
		c.compareAndSwapMetrics(mptr, nil)
		c.sendAsync(mptr)
	}
}

// Flush satisfies the stats.Flusher interface.
func (c *Client) Flush() {
	c.flush()
	c.join.Wait()
}

// Close closes the client, flushing all buffered metrics and releasing internal
// iresources.
func (c *Client) Close() error {
	c.flush()
	c.once.Do(func() { close(c.done) })
	c.join.Wait()
	return nil
}

func (c *Client) flush() {
	for {
		mptr := c.loadMetrics()
		if mptr == nil {
			break
		}
		if c.compareAndSwapMetrics(mptr, nil) {
			c.sendAsync(mptr)
			break
		}
	}
}

func (c *Client) sendAsync(m *metrics) {
	c.setLastFlush(time.Now())
	c.join.Add(1)
	go c.send(m)
}

func (c *Client) send(m *metrics) {
	defer c.join.Done()
	defer c.releaseMetrics(m)

	for attempt := 0; attempt != 10; attempt++ {
		if attempt != 0 {
			select {
			case <-time.After(c.httpClient.Timeout):
			case <-c.done:
				return
			}
		}

		r, err := c.httpClient.Do(&http.Request{
			Method:        "POST",
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			URL:           c.url,
			Body:          newMetricsReader(m),
			ContentLength: int64(m.size),
		})

		if err != nil {
			log.Print("stats/influxdb:", err)
			continue
		}

		if err := readResponse(r); err != nil {
			log.Printf("stats/influxdb: POST %s: %d %s: %s", c.url, r.StatusCode, r.Status, err)
			continue
		}

		break
	}
}

func (c *Client) acquireMetrics() *metrics {
	return c.pool.Get().(*metrics)
}

func (c *Client) releaseMetrics(m *metrics) {
	m.reset()
	c.pool.Put(m)
}

func (c *Client) loadMetrics() *metrics {
	return (*metrics)(atomic.LoadPointer(&c.metrics))
}

func (c *Client) compareAndSwapMetrics(old *metrics, new *metrics) bool {
	return atomic.CompareAndSwapPointer(&c.metrics,
		unsafe.Pointer(old),
		unsafe.Pointer(new),
	)
}

func (c *Client) setLastFlush(t time.Time) {
	atomic.StoreInt64(&c.flushedAt, time.Now().UnixNano())
}

func (c *Client) lastFlush() time.Time {
	return time.Unix(0, atomic.LoadInt64(&c.flushedAt))
}

func (c *Client) run(flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case now := <-ticker.C:
			if now.Sub(c.lastFlush()) >= flushInterval {
				c.flush()
			}
		}
	}
}

func makeURL(address string, database string) *url.URL {
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}

	u, err := url.Parse(address)
	if err != nil {
		panic(err)
	}

	if len(u.Scheme) == 0 {
		u.Scheme = "http"
	}

	if len(u.Path) == 0 {
		u.Path = "/write"
	}

	q := u.Query()

	if _, ok := q["db"]; !ok {
		q.Set("db", database)
		u.RawQuery = q.Encode()
	}

	return u
}

func readResponse(r *http.Response) error {
	if r.StatusCode < 300 {
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
		return nil
	}

	info := &influxError{}
	err := json.NewDecoder(r.Body).Decode(info)
	r.Body.Close()

	if err != nil {
		return err
	}

	return info
}

type influxError struct {
	Err string `json:"error"`
}

func (e *influxError) Error() string {
	return e.Err
}
