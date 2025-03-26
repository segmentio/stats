package datadog

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	stats "github.com/segmentio/stats/v5"
)

var testMeasures = []struct {
	m  stats.Measure
	s  string
	dp []string
}{
	{
		m: stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				stats.MakeField("count", 5, stats.Counter),
			},
		},
		s: `request.count:5|c
`,
		dp: []string{},
	},

	{
		m: stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				stats.MakeField("count", 5, stats.Counter),
				stats.MakeField("rtt", 100*time.Millisecond, stats.Histogram),
			},
			Tags: []stats.Tag{
				stats.T("answer", "42"),
				stats.T("hello", "world"),
			},
		},
		s: `request.count:5|c|#answer:42,hello:world
request.rtt:0.1|h|#answer:42,hello:world
`,
		dp: []string{},
	},

	{
		m: stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				stats.MakeField("dist_rtt", 100*time.Millisecond, stats.Histogram),
			},
			Tags: []stats.Tag{
				stats.T("answer", "42"),
				stats.T("hello", "world"),
			},
		},
		s: `request.dist_rtt:0.1|d|#answer:42,hello:world
`,
		dp: []string{"dist_"},
	},
}

func TestAppendMeasure(t *testing.T) {
	client := NewClient(DefaultAddress)
	for _, test := range testMeasures {
		t.Run(test.s, func(t *testing.T) {
			client.distPrefixes = test.dp
			if s := string(client.AppendMeasure(nil, test.m)); s != test.s {
				t.Error("bad metric representation:")
				t.Log("expected:", test.s)
				t.Log("found:   ", s)
			}
		})
	}
}

var (
	testDistNames = []struct {
		n string
		d bool
	}{
		{
			n: "name",
			d: false,
		},
		{
			n: "",
			d: false,
		},
		{
			n: "dist_name",
			d: true,
		},
		{
			n: "distname",
			d: false,
		},
	}
	distPrefixes = []string{"dist_"}
)

func TestSendDist(t *testing.T) {
	client := NewClientWith(ClientConfig{DistributionPrefixes: distPrefixes})
	for _, test := range testDistNames {
		t.Run(test.n, func(t *testing.T) {
			a := client.sendDist(test.n)
			if a != test.d {
				t.Error("distribution name detection incorrect:")
				t.Log("expected:", test.d)
				t.Log("found:   ", a)
			}
		})
	}
}

type MockWriteCloser struct {
	*bytes.Buffer
	closed bool
}

func (mwc *MockWriteCloser) Close() error {
	if mwc.closed {
		return errors.New("mock writer already closed")
	}
	mwc.closed = true
	return nil
}

var testSerializerByteArrays = []struct {
	name           string
	data           string
	expectedOutput string
	expectedBytes  int
}{
	{
		name:           "Data with valid UTF",
		data:           "metric.name:1|c|#tag1:value1,tag2:value2",
		expectedOutput: "metric.name:1|c|#tag1:value1,tag2:value2",
		expectedBytes:  40, // The length of the expected output
	},
	{
		name:           "Data with invalid UTF in metric name, value & tags",
		data:           "\xff\xfe.metric.name:\xe2\x26|h|#tag1:\xffvalue;tag2:value2\xff",
		expectedOutput: "\uFFFD.metric.name:\uFFFD\x26|h|#tag1:\uFFFDvalue;tag2:value2\uFFFD",
		expectedBytes:  52, // The length of the expected output
	},
	{
		name:           "Multiline data with valid UTF",
		data:           "metric.name:1|c|#tag1:value1,tag2:value2\nmetric.name:1|c|#tag1:value1,tag2:value2",
		expectedOutput: "metric.name:1|c|#tag1:value1,tag2:value2\nmetric.name:1|c|#tag1:value1,tag2:value2",
		expectedBytes:  81, // The length of the expected output
	},
	{
		name:           "Multiline data with invalid UTF in metric name, value & tags",
		data:           "\xff\xfe.metric.name:\xe2\x26|h|#tag1:\xffvalue;tag2:value2\xff\n\xff\xfe.metric.name:\xe2\x26|h|#tag1:\xffvalue;tag2:value2\xff",
		expectedOutput: "\uFFFD.metric.name:\uFFFD\x26|h|#tag1:\uFFFDvalue;tag2:value2\uFFFD\n\uFFFD.metric.name:\uFFFD\x26|h|#tag1:\uFFFDvalue;tag2:value2\uFFFD",
		expectedBytes:  105, // The length of the expected output
	},
	{
		name:           "Data with emojis & invalid UTF in metric name, value & tags",
		data:           "🙂.metric.name:💥\x26|h|#tag1:\xffvalue;tag2:value2🧨",
		expectedOutput: "🙂.metric.name:💥\x26|h|#tag1:\uFFFDvalue;tag2:value2🧨",
		expectedBytes:  55, // The length of the expected output
	},
}

func TestSerializerWriteWithInvalidUnicode(t *testing.T) {
	for _, test := range testSerializerByteArrays {
		t.Run(test.name, func(t *testing.T) {
			// Mock connection
			mockBuffer := &MockWriteCloser{Buffer: &bytes.Buffer{}}
			s := &serializer{
				conn:       mockBuffer,
				bufferSize: MaxBufferSize,
			}
			bytesWritten, err := s.Write([]byte(test.data))
			if err != nil {
				t.Errorf("Write failed: %v", err)
			}
			// Check the number of bytes written
			if bytesWritten != test.expectedBytes {
				t.Errorf("Expected %d bytes to be written, got %d", test.expectedBytes, bytesWritten)
			}
			// Validate the final buffer output
			finalOutput := mockBuffer.String()
			if !strings.Contains(finalOutput, test.expectedOutput) {
				t.Errorf("Input data: %q; Got output: %q; Expected output: %q", test.data, finalOutput, test.expectedOutput)
			}
			// Check for UTF-8 validity in output
			if !utf8.ValidString(finalOutput) {
				t.Error("Output contains invalid UTF-8 sequences")
			}
		})
	}
}
