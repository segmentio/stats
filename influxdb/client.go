package influxdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/objconv/json"
	"github.com/segmentio/stats"
)

const (
	// DefaultAddress is the default address to which the InfluxDB client tries
	// to connect to.
	DefaultAddress = "localhost:8086"

	// DefaultDatabase is the default database used by the InfluxDB client.
	DefaultDatabase = "stats"

	// DefaultBufferSize is the default size for batches of metrics sent to
	// InfluxDB.
	DefaultBufferSize = 2 * 1024 * 1024 // 2 MB

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
	BufferSize int

	// Maximum amount of time that requests to InfluxDB may take.
	Timeout time.Duration

	// Transport configures the HTTP transport used by the client to send
	// requests to InfluxDB. By default http.DefaultTransport is used.
	Transport http.RoundTripper
}

// Client represents an InfluxDB client that implements the stats.Handler
// interface.
type Client struct {
	serializer
	buffer stats.Buffer
}

// NewClient creates and returns a new InfluxDB client publishing metrics to the
// server running at addr.
func NewClient(addr string) *Client {
	return NewClientWith(ClientConfig{
		Address: addr,
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

	if config.BufferSize == 0 {
		config.BufferSize = DefaultBufferSize
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	c := &Client{
		serializer: serializer{
			url:  makeURL(config.Address, config.Database),
			done: make(chan struct{}),
			http: http.Client{
				Timeout:   config.Timeout,
				Transport: config.Transport,
			},
		},
	}

	c.buffer.BufferSize = config.BufferSize
	c.buffer.Serializer = &c.serializer
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

	r, err := c.http.Post(u.String(), "application/x-www-form-urlencoded", strings.NewReader(
		fmt.Sprintf("q=CREATE DATABASE %q", db),
	))
	if err != nil {
		return err
	}
	return readResponse(r)
}

// HandleMetric satisfies the stats.Handler interface.
func (c *Client) HandleMeasures(time time.Time, measures ...stats.Measure) {
	c.buffer.HandleMeasures(time, measures...)
}

// Flush satisfies the stats.Flusher interface.
func (c *Client) Flush() {
	c.buffer.Flush()
}

// Close flushes and closes the client, satisfies the io.Closer interface.
func (c *Client) Close() error {
	c.once.Do(func() { close(c.done) })
	c.Flush()
	return nil
}

type serializer struct {
	url  *url.URL
	http http.Client
	once sync.Once
	done chan struct{}
}

func (*serializer) AppendMeasures(b []byte, time time.Time, measures ...stats.Measure) []byte {
	for _, m := range measures {
		b = AppendMeasure(b, time, m)
	}
	return b
}

func (s *serializer) Write(b []byte) (n int, err error) {
	for attempt := 0; attempt != 10; attempt++ {
		var res *http.Response

		if attempt != 0 {
			select {
			case <-time.After(s.http.Timeout):
			case <-s.done:
				err = context.Canceled
				return
			}
		}

		req, _ := http.NewRequest("POST", s.url.String(), bytes.NewReader(b))
		res, err = s.http.Do(req)
		if err != nil {
			log.Print("stats/influxdb:", err)
			continue
		}

		if err = readResponse(res); err != nil {
			log.Printf("stats/influxdb: POST %s: %d %s: %s", s.url, res.StatusCode, res.Status, err)
			continue
		}

		break
	}

	n = len(b)
	return
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
