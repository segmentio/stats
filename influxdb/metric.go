package influxdb

import (
	"io"
	"strconv"
	"sync/atomic"

	"github.com/segmentio/stats"
)

func appendMetric(b []byte, m *stats.Metric) []byte {
	if len(m.Namespace) != 0 {
		b = append(b, m.Namespace...)
		b = append(b, '.')
	}

	b = append(b, m.Name...)

	for _, tag := range m.Tags {
		b = append(b, ',')
		b = append(b, tag.Name...)
		b = append(b, '=')
		b = append(b, tag.Value...)
	}

	b = append(b, " value="...)
	b = strconv.AppendFloat(b, m.Value, 'g', -1, 64)

	b = append(b, ' ')
	b = strconv.AppendInt(b, m.Time.UnixNano(), 10)

	return append(b, '\n')
}

type metrics struct {
	lines  [][]byte
	length int32
	size   int32
	remain int32
}

func newMetrics(capacity int) *metrics {
	return &metrics{
		lines:  make([][]byte, capacity),
		remain: int32(capacity),
	}
}

func (m *metrics) reset() {
	m.length = 0
	m.size = 0
	m.remain = int32(len(m.lines))
}

func (m *metrics) len() int {
	i := int(atomic.LoadInt32(&m.length))
	n := len(m.lines)

	if i > n {
		i = n
	}

	return i
}

func (m *metrics) append(metric *stats.Metric) (flush bool, added bool) {
	i := int(atomic.AddInt32(&m.length, 1)) - 1
	n := len(m.lines)

	if i >= n {
		return
	}

	line := appendMetric(m.lines[i][:0], metric)
	m.lines[i] = line
	atomic.AddInt32(&m.size, int32(len(line)))

	flush = atomic.AddInt32(&m.remain, -1) == 0
	added = true
	return
}

type metricsReader struct {
	lines  [][]byte
	index  int
	offset int
}

func newMetricsReader(m *metrics) *metricsReader {
	return &metricsReader{
		lines: m.lines[:m.len()],
	}
}

func (m *metricsReader) Close() error {
	m.index = len(m.lines)
	m.offset = 0
	return nil
}

func (m *metricsReader) Read(b []byte) (n int, err error) {
	for c := -1; c != 0 && n < len(b); n += c {
		c = m.fill(b[n:])
	}
	if n == 0 && m.index == len(m.lines) {
		err = io.EOF
	}
	return
}

func (m *metricsReader) fill(b []byte) int {
	if m.index == len(m.lines) {
		return 0
	}

	l := m.lines[m.index][m.offset:]
	n := copy(b, l)

	if n == len(l) {
		m.index++
		m.offset = 0
	} else {
		m.offset += n
	}

	return n
}

func (m *metricsReader) WriteTo(w io.Writer) (n int64, err error) {
	for err == nil && m.index != len(m.lines) {
		var c int
		var l = m.lines[m.index][m.offset:]

		c, err = w.Write(l)
		if c > 0 {
			n += int64(c)

			if c == len(l) {
				m.index++
				m.offset = 0
			} else {
				m.offset += c
			}
		}
	}
	return
}
