package netstats

import "github.com/segmentio/stats"

type Metrics struct {
	Reads    stats.Histogram
	Writes   stats.Histogram
	BytesIn  stats.Counter
	BytesOut stats.Counter
	Errors   stats.Counter
}

func NewMetrics(client stats.Client, tags ...stats.Tag) *Metrics {
	m := &Metrics{
		Errors: client.Counter("conn.errors.count", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	// read
	tags[n] = stats.Tag{"operation", "read"}
	m.Reads = client.Histogram("conn.iops", tags...)
	m.BytesIn = client.Counter("conn.bytes.count", tags...)

	// write
	tags[n] = stats.Tag{"operation", "write"}
	m.Writes = client.Histogram("conn.iops", tags...)
	m.BytesOut = client.Counter("conn.bytes.count", tags...)

	return m
}
