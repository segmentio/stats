package netstats

import "github.com/segmentio/stats"

type Metrics struct {
	BytesIn  stats.Histogram
	BytesOut stats.Histogram
	Errors   stats.Counter
}

func NewMetrics(client stats.Client, tags ...stats.Tag) *Metrics {
	return &Metrics{
		BytesIn:  client.Histogram("conn.read.bytes", tags...),
		BytesOut: client.Histogram("conn.write.bytes", tags...),
		Errors:   client.Counter("conn.errors.count", tags...),
	}
}
