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
			Name: "request|foo",
			Fields: []stats.Field{
				stats.MakeField("count", 5, stats.Counter),
			},
		},
		s: `request_foo.count:5|c
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
				stats.MakeField("count", 5, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("ans|wer:blah", "also|pipe:colon,comma"),
				stats.T("hello", "world"),
			},
		},
		s: `request.count:5|c|#ans_wer_blah:also_pipe_colon_comma,hello:world
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

func TestAppendSanitizedMetricName(t *testing.T) {
	long := strings.Repeat("x", 300) // longer than maxLen
	cases := []struct {
		prefix   string // existing data in buffer
		in, want string
	}{
		// Test with empty prefix (original behavior)
		{"", "cpu.load", "cpu.load"},
		{"", "abc_DEF-123", "abc_DEF-123"},

		// Test with existing data preservation
		{"myapp.", "cpu.load", "myapp.cpu.load"},
		{"prefix_", "abc_DEF-123", "prefix_abc_DEF-123"},
		{"server1.", "memory.usage", "server1.memory.usage"},

		// spaces / punctuation
		{"", "CPU Load %", "CPU_Load"},
		{"prefix.", "foo|bar:baz@2", "prefix.foo_bar_baz_2"},
		{"app_", "a/b\\c*d?e", "app_a_b_c_d_e"},

		// leading / trailing rubbish
		{"", "__bad__", "bad"},
		{"", "----abc", "abc"},
		{"", "abc---", "abc"},
		{"", "...abc..def...", "abc..def"},
		{"prefix_", "..trimmed..", "prefix_trimmed"},

		// consecutive illegal chars collapse
		{"", "foo!!!@@@###bar", "foo_bar"},
		{"app.", "foo!!!@@@###bar", "app.foo_bar"},

		// Unicode / accent folding - existing cases
		{"", "🐳docker.stats", "docker.stats"},
		{"", "résumé.clicks", "resume.clicks"},
		{"prefix_", "café.orders", "prefix_cafe.orders"},

		// New accent test cases covering various languages
		{"", "naïve.users", "naive.users"},           // ï -> i
		{"", "señor.requests", "senor.requests"},     // ñ -> n
		{"", "München.traffic", "Munchen.traffic"},   // ü -> u
		{"", "Zürich.latency", "Zurich.latency"},     // ü -> u
		{"", "Åse.connections", "Ase.connections"},   // Å -> A
		{"", "Björk.plays", "Bjork.plays"},           // ö -> o
		{"", "François.logins", "Francois.logins"},   // ç -> c, ç -> c
		{"", "Athènes.visits", "Athenes.visits"},     // è -> e
		{"", "São.Paulo.errors", "Sao.Paulo.errors"}, // ã -> a
		{"", "Malmö.requests", "Malmo.requests"},     // ö -> o
		{"", "Ølberg.metrics", "Olberg.metrics"},     // Ø -> O
		{"", "Reykjavík.data", "Reykjavik.data"},     // í -> i
		{"", "Kraków.sessions", "Krakow.sessions"},   // ó -> o
		{"", "Torshavn.bytes", "Torshavn.bytes"},     // þ -> t (if you had Þórshavn)

		// Mixed accents with existing data
		{"db_", "entrée.créée", "db_entree.creee"},     // é -> e, é -> e, é -> e
		{"api.", "piñata.fiesta", "api.pinata.fiesta"}, // ñ -> n
		{"web_", "crème.brûlée", "web_creme.brulee"},   // è -> e, û -> u, é -> e

		// Multi-byte UTF-8 sequences (emojis, symbols)
		{"", "🤡circus.metrics", "circus.metrics"},       // 4-byte emoji -> single _
		{"", "ci🤡rcus.metrics", "ci_rcus.metrics"},      // 4-byte emoji -> single _
		{"", "hello🌍world", "hello_world"},              // emoji in middle
		{"", "perf🚀🎯ormance", "perf_ormance"},           // multiple emojis -> single _
		{"", "chinese.测试.metrics", "chinese._.metrics"}, // Chinese characters -> _
		{"", "cyrillic.тест.data", "cyrillic._.data"},   // Cyrillic -> _
		{"", "arabic.العربية.stats", "arabic._.stats"},  // Arabic -> _
		{"", "🤡🎭🎪", "_truncated_"},                      // only emojis -> single _
		{"prefix_", "🤡test", "prefix_test"},             // emoji at start with prefix
		{"", "test🤡end", "test_end"},                    // emoji in middle
		{"", "emoji🤡and🎯more", "emoji_and_more"},        // multiple emojis mixed

		// Mixed Latin-1 accents with multi-byte UTF-8
		{"", "café🤡résumé", "cafe_resume"},           // Latin-1 accents + emoji
		{"", "naïve🌍test", "naive_test"},             // ï -> i, emoji -> _
		{"", "umlauts.🤡München", "umlauts._Munchen"}, // emoji + German umlauts

		// empty or only illegal
		{"", "", "_unnamed_"},
		{"", "!!!", "_truncated_"},
		{"prefix_", "", "prefix_"},
		{"prefix_", "!!!", "prefix__truncated_"},

		// over-long → truncated (but preserve prefix if it fits)
		{"", long, strings.Repeat("x", maxLen)},
		{"short_", long, "short_" + strings.Repeat("x", maxLen-6)}, // 6 = len("short_")

		// Test edge case where prefix + content exceeds maxLen
		{strings.Repeat("x", 240), "content.data.here", strings.Repeat("x", 240) + "content.da"}, // Should truncate at maxLen=250

	}

	for _, c := range cases {
		// Start with prefix data in buffer
		buf := []byte(c.prefix)
		originalLen := len(buf)

		// Append sanitized metric name
		buf = appendSanitizedMetricName(buf, c.in)
		got := string(buf)

		if got != c.want {
			t.Fatalf("prefix=%q in=%q  want=%q  got=%q", c.prefix, c.in, c.want, got)
		}

		// Verify prefix is preserved
		if len(c.prefix) > 0 && !strings.HasPrefix(got, c.prefix) {
			t.Errorf("prefix %q not preserved in result %q", c.prefix, got)
		}

		// Verify length constraints
		if len(buf) > maxLen {
			t.Errorf("result %q length=%d exceeds maxLen=%d", got, len(buf), maxLen)
		}

		// Verify we only modified the buffer from the original length onward
		if originalLen > 0 && originalLen <= len(buf) {
			originalPart := string(buf[:originalLen])
			if originalPart != c.prefix {
				t.Errorf("original buffer data corrupted: want %q, got %q", c.prefix, originalPart)
			}
		}
	}

	// Additional test: verify behavior with various buffer capacities
	t.Run("BufferReuse", func(t *testing.T) {
		buf := make([]byte, 0, 100) // pre-allocated capacity
		buf = append(buf, "test_"...)
		buf = appendSanitizedMetricName(buf, "café.métrics")
		expected := "test_cafe.metrics"
		if string(buf) != expected {
			t.Errorf("buffer reuse failed: want %q, got %q", expected, string(buf))
		}
	})
}
