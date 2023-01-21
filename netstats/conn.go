package netstats

import (
	"io"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/vpcinfo"
	"github.com/vertoforce/stats"
)

func init() {
	stats.Buckets.Set("conn.read:bytes",
		1e2, // 100 B
		1e3, // 1 KB
		1e4, // 10 KB
		1e5, // 100 KB
		math.Inf(+1),
	)

	stats.Buckets.Set("conn.write:bytes",
		1e2, // 100 B
		1e3, // 1 KB
		1e4, // 10 KB
		1e5, // 100 KB
		math.Inf(+1),
	)
}

// NewConn returns a net.Conn object that wraps c and produces metrics on the
// default engine.
func NewConn(c net.Conn) net.Conn {
	return NewConnWith(stats.DefaultEngine, c)
}

// NewConnWith returns a net.Conn object that wraps c and produces metrics on eng.
func NewConnWith(eng *stats.Engine, c net.Conn) net.Conn {
	nc := &conn{Conn: c, eng: eng}

	proto := c.LocalAddr().Network()
	nc.r.metrics.protocol = proto
	nc.w.metrics.protocol = proto

	eng.Incr("conn.open.count", stats.T("protocol", proto))
	return nc
}

type conn struct {
	net.Conn
	eng  *stats.Engine
	once sync.Once

	r struct {
		sync.Mutex
		metrics struct {
			count      int    `metric:"count" type:"counter"`
			bytes      int    `metric:"bytes" type:"histogram"`
			protocol   string `tag:"protocol"`
			inZone     string `tag:"in_zone"`
			targetZone string `tag:"target_zone"`
			sourceZone string `tag:"source_zone"`
		} `metric:"conn.read"`
	}

	w struct {
		sync.Mutex
		metrics struct {
			count      int    `metric:"count" type:"counter"`
			bytes      int    `metric:"bytes" type:"histogram"`
			protocol   string `tag:"protocol"`
			inZone     string `tag:"in_zone"`
			targetZone string `tag:"target_zone"`
			sourceZone string `tag:"source_zone"`
		} `metric:"conn.write"`
	}
}

func (c *conn) BaseConn() net.Conn {
	return c.Conn
}

func (c *conn) Close() (err error) {
	err = c.Conn.Close()
	c.once.Do(func() {
		c.eng.Incr("conn.close.count", stats.T("protocol", c.w.metrics.protocol))
		if err != nil {
			c.error("close", err)
		}
	})
	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	sourceZone := zoneOf(c.Conn)
	targetZone := currentZone()

	c.r.Lock()
	c.r.metrics.count = 1
	c.r.metrics.bytes = n
	c.r.metrics.sourceZone = sourceZone
	c.r.metrics.targetZone = targetZone
	c.r.metrics.inZone = inZone(sourceZone, targetZone)
	c.eng.Report(&c.r)
	c.r.Unlock()

	if err != nil && err != io.EOF {
		c.error("read", err)
	}

	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	sourceZone := currentZone()
	targetZone := zoneOf(c.Conn)

	c.w.Lock()
	c.w.metrics.count = 1
	c.w.metrics.bytes = n
	c.w.metrics.sourceZone = sourceZone
	c.w.metrics.targetZone = targetZone
	c.w.metrics.inZone = inZone(sourceZone, targetZone)
	c.eng.Report(&c.w)
	c.w.Unlock()

	if err != nil {
		c.error("write", err)
	}

	return
}

func (c *conn) SetDeadline(t time.Time) (err error) {
	if err = c.Conn.SetDeadline(t); err != nil {
		c.error("set-deadline", err)
	}
	return
}

func (c *conn) SetReadDeadline(t time.Time) (err error) {
	if err = c.Conn.SetReadDeadline(t); err != nil {
		c.error("set-read-deadline", err)
	}
	return
}

func (c *conn) SetWriteDeadline(t time.Time) (err error) {
	if err = c.Conn.SetWriteDeadline(t); err != nil {
		c.error("set-write-deadline", err)
	}
	return
}

func (c *conn) error(op string, err error) {
	switch err = rootError(err); err {
	case io.EOF, io.ErrClosedPipe, io.ErrUnexpectedEOF:
		// this is expected to happen when connections are closed
	default:
		// only report serious errors, others should be handled gracefully
		if !isTemporary(err) {
			c.eng.Incr("conn.error.count",
				stats.T("operation", op),
				stats.T("protocol", c.w.metrics.protocol),
			)
		}
	}
}

func rootError(err error) error {
searchRootError:
	for i := 0; i != 10; i++ { // protect against cyclic errors
		switch e := err.(type) {
		case *net.OpError:
			err = e.Err
		default:
			break searchRootError
		}
	}
	return err
}

func normalizeZone(zone string) string {
	if zone == "" {
		return "N/A"
	}
	return zone
}

func currentZone() string {
	zone, _ := vpcinfo.LookupZone()
	return normalizeZone(zone.String())
}

func zoneOf(conn net.Conn) string {
	subnets, _ := vpcinfo.LookupSubnets()
	return normalizeZone(subnets.LookupAddr(conn.RemoteAddr()).Zone)
}

func inZone(source, target string) string {
	if source == "N/A" || target == "N/A" {
		return "false"
	}
	return strconv.FormatBool(source == target)
}
