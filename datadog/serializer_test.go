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
		s: `request.count:5|c|#ans_wer_blah:also|pipe:colon_comma,hello:world
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

	// Test lenient tag values - URLs, paths, colons, pipes
	{
		m: stats.Measure{
			Name: "api.request",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("url", "http://api.example.com/v1/users"),
				stats.T("path", "/api/v1/users"),
				stats.T("env", "prod:us-east-1"),
			},
		},
		s: `api.request.count:1|c|#url:http://api.example.com/v1/users,path:/api/v1/users,env:prod:us-east-1
`,
		dp: []string{},
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
		{"", "0_ 0", "0__0"}, // legitimate underscore followed by space -> two underscores (one legit, one replacement)

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

		// Unmapped Latin-1 Supplement characters (regression test for panic)
		{"", "test÷metric", "test_metric"},                 // ÷ (division sign, U+00F7, byte 247)
		{"", "value×count", "value_count"},                 // × (multiplication sign, U+00D7, byte 215)
		{"", "price¤amount", "price_amount"},               // ¤ (currency sign, U+00A4)
		{"prefix_", "data÷by×time", "prefix_data_by_time"}, // multiple unmapped chars with prefix

		// Edge cases for UTF-8 continuation byte handling
		{"", "test\xC3\xA9\x80suffix", "teste_suffix"},   // é followed by orphaned continuation byte
		{"", "metric\xE2\x82", "metric"},                 // incomplete 3-byte sequence (skips continuation byte, no replacement at end)
		{"", "test\xF0\x9F", "test"},                     // incomplete 4-byte emoji (no continuation bytes, no replacement at end)
		{"", "test\xC0\x80", "test"},                     // overlong encoding of NULL (invalid, skips continuation byte)
		{"", "\x80\x81\x82\xBF", "_truncated_"},          // all invalid continuation bytes
		{"", "valid\xC3\xA9\xC3\xA8text", "valideetext"}, // adjacent 2-byte sequences (é è)
		{"", "test\xED\xA0\x80invalid", "test_invalid"},  // invalid surrogate half (UTF-16 artifact)
		{"", "hello\xF4\x90\x80\x80", "hello"},           // codepoint beyond U+10FFFF (invalid, skips all continuation bytes)

		// Latin-1 Supplement boundary characters
		{"", "À", "A"},                     // U+00C0 (first mapped character)
		{"", "ÿ", "y"},                     // U+00FF (last mapped character)
		{"", "\u00C0test\u00FF", "Atesty"}, // boundaries with text

		// Mixed valid and invalid sequences
		{"", "café\xE2\x82résumé", "cafe_resume"},               // valid + incomplete + valid
		{"", "test\xF0\x9F\x98\x80end", "test_end"},             // valid 4-byte emoji (should become single _)
		{"", "\xF0\x9F\x98\x80\xF0\x9F\x98\x81", "_truncated_"}, // two valid 4-byte emojis (all invalid, only punctuation)

		// empty or only illegal
		{"", "", "_unnamed_"},
		{"", "!!!", "_truncated_"},
		{"prefix_", "", "prefix_"},
		{"prefix_", "!!!", "prefix__truncated_"},

		// only trim characters (dots, underscores, dashes) - fast path edge case
		{"", "...", "_truncated_"},
		{"", "___", "_truncated_"},
		{"", "---", "_truncated_"},
		{"", "._-._-", "_truncated_"},
		{"", "......", "_truncated_"},
		{"", "______", "_truncated_"},
		{"", "------", "_truncated_"},
		{"prefix_", "...", "prefix__truncated_"},
		{"prefix_", "___", "prefix__truncated_"},
		{"prefix_", "._-", "prefix__truncated_"},

		// over-long → truncated (but preserve prefix if it fits)
		{"", long, strings.Repeat("x", maxLen)},
		{"short_", long, "short_" + strings.Repeat("x", maxLen)}, // 6 = len("short_")

		{strings.Repeat("x", 240), "content.data.here", strings.Repeat("x", 240) + "content.data.here"},
	}

	for _, c := range cases {
		// Start with prefix data in buffer
		buf := []byte(c.prefix)
		originalLen := len(buf)

		// Append sanitized metric name
		buf = appendSanitizedMetricName(buf, c.in)
		got := string(buf)

		if got != c.want {
			t.Fatalf("prefix=%q in=%q want=%q (len %v) got=%q (len %v)", c.prefix, c.in, c.want, len(c.want), got, len(got))
		}

		// Verify prefix is preserved
		if len(c.prefix) > 0 && !strings.HasPrefix(got, c.prefix) {
			t.Errorf("prefix %q not preserved in result %q", c.prefix, got)
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

// FuzzAppendSanitizedMetricName performs fuzz testing to discover edge cases
// in metric name sanitization, particularly around UTF-8 handling.
func FuzzAppendSanitizedMetricName(f *testing.F) {
	// Seed corpus with interesting test cases
	f.Add("")
	f.Add("simple")
	f.Add("café")
	f.Add("🤡test")
	f.Add("\xff\xfe")
	f.Add("test\xC3\xA9\x80suffix")
	f.Add("\xF0\x9F\x98\x80") // valid emoji
	f.Add("\xED\xA0\x80")     // invalid surrogate
	f.Add("naïve.users")
	f.Add(strings.Repeat("x", 300)) // over maxLen

	f.Fuzz(func(t *testing.T, input string) {
		buf := appendSanitizedMetricName(nil, input)
		result := string(buf)

		// Invariant 1: Output must be valid UTF-8
		// (Note: The serializer.Write() method also validates, but sanitization should produce valid UTF-8)
		if !utf8.ValidString(result) {
			t.Errorf("Output is not valid UTF-8 for input %q: got %q", input, result)
		}

		// Invariant 2: Output length must not exceed maxLen (except special fallback cases)
		if len(result) > maxLen && result != "_unnamed_" && result != "_truncated_" {
			t.Errorf("Output exceeds maxLen: %d > %d for input %q", len(result), maxLen, input)
		}

		// Invariant 3: Output must not start/end with trim chars (except special cases)
		if len(result) > 0 && result != "_unnamed_" && result != "_truncated_" {
			if isTrim(result[0]) {
				t.Errorf("Output starts with trim char for input %q: %q", input, result)
			}
			if isTrim(result[len(result)-1]) {
				t.Errorf("Output ends with trim char for input %q: %q", input, result)
			}
		}

		// Invariant 4: Every ASCII byte in output must be valid or replacement
		for i := 0; i < len(result); i++ {
			b := result[i]
			// Only check ASCII range (< 128), as valid UTF-8 multi-byte sequences are allowed
			if b < 128 && !valid[b] && b != replacement {
				t.Errorf("Invalid ASCII byte in output at position %d: 0x%02X for input %q", i, b, input)
			}
		}

		// Invariant 5: Empty input should produce "_unnamed_"
		if input == "" && result != "_unnamed_" {
			t.Errorf("Empty input should produce '_unnamed_', got %q", result)
		}
	})
}

// TestSanitizationPreservesUTF8Validity ensures that the full pipeline
// (sanitize → Write → output) maintains UTF-8 validity, even with problematic inputs.
func TestSanitizationPreservesUTF8Validity(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"normal metric", "normal.metric"},
		{"Latin-1 accents", "café.résumé"},
		{"orphaned continuation byte", "test\xC3\xA9\x80suffix"},
		{"incomplete 3-byte", "metric\xE2\x82"},
		{"valid emoji", string([]byte{0xF0, 0x9F, 0x98, 0x80})},
		{"incomplete emoji", string([]byte{0xF0, 0x9F})},
		{"overlong encoding", "test\xC0\x80"},
		{"invalid surrogate", "test\xED\xA0\x80invalid"},
		{"mixed valid/invalid", "café\xE2\x82résumé"},
		{"all continuation bytes", "\x80\x81\x82\xBF"},
		{"multiple emojis", "test🤡and🎯more"},
		{"Chinese characters", "测试.metrics"},
		{"Cyrillic text", "тест.data"},
		{"mixed scripts", "café测试résumé🤡тест"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockBuffer := &MockWriteCloser{Buffer: &bytes.Buffer{}}
			s := &serializer{
				conn:       mockBuffer,
				bufferSize: MaxBufferSize,
			}

			measure := stats.Measure{
				Name: tc.input,
				Fields: []stats.Field{
					stats.MakeField("count", 1, stats.Counter),
				},
				Tags: []stats.Tag{
					stats.T("tagname", tc.input),
				},
			}

			buf := s.AppendMeasure(nil, measure)
			_, err := s.Write(buf)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			output := mockBuffer.String()
			if !utf8.ValidString(output) {
				t.Errorf("Final output is not valid UTF-8: %q", output)
			}

			// Additional check: ensure output doesn't contain raw invalid bytes
			for i, r := range output {
				if r == utf8.RuneError {
					_, size := utf8.DecodeRuneInString(output[i:])
					if size == 1 {
						// This is an actual decoding error, not the valid U+FFFD character
						t.Errorf("Output contains invalid UTF-8 at position %d", i)
					}
				}
			}
		})
	}
}

func TestAppendSanitizedTagValue(t *testing.T) {
	long := strings.Repeat("x", 300) // longer than maxLen
	cases := []struct {
		prefix   string // existing data in buffer
		in, want string
	}{
		// Basic cases - tag values should be more lenient
		{"", "simple", "simple"},
		{"", "with-dashes_underscores.dots", "with-dashes_underscores.dots"},

		// Tag values can contain colons and pipes (unlike the protocol separators)
		{"", "http://example.com", "http://example.com"},
		{"", "path/to/resource", "path/to/resource"},
		{"", "key:value:pair", "key:value:pair"},
		{"", "pipe|separated|values", "pipe|separated|values"},
		{"", "mixed:pipe|slash/colon", "mixed:pipe|slash/colon"},

		// Commas must be sanitized (they separate tags in the protocol)
		{"", "value,with,commas", "value_with_commas"},
		{"", "item1,item2,item3", "item1_item2_item3"},

		// Special characters that should be sanitized
		{"", "value@sign#hash", "value_sign_hash"},
		{"", "brackets[test]", "brackets_test"},
		{"", "parens(test)", "parens_test"},

		// Accented characters should be normalized
		{"", "café", "cafe"},
		{"", "naïve", "naive"},
		{"", "señor", "senor"},

		// Leading/trailing special chars should be trimmed
		{"", "-leading-dash", "leading-dash"},
		{"", "trailing-dash-", "trailing-dash"},
		{"", "...dots...", "dots"},
		{"", "__underscores__", "underscores"},

		// Empty string handling
		{"", "", ""},
		{"prefix:", "", "prefix:"},

		// Multiple consecutive special chars collapse
		{"", "foo!!!bar", "foo_bar"},
		{"", "test@@@value", "test_value"},

		// Mixed valid and invalid characters
		{"", "env:prod|region:us-east-1", "env:prod|region:us-east-1"},
		{"", "url:http://api.example.com/v1", "url:http://api.example.com/v1"},
		{"", "list:item1,item2,item3", "list:item1_item2_item3"},

		// Emojis and other multi-byte sequences
		{"", "test🤡emoji", "test_emoji"},
		{"", "hello🌍world", "hello_world"},
		{"", "测试值", "_truncated_"}, // Chinese -> _truncated_ (all invalid)

		// Over-long values should be truncated
		{"", long, strings.Repeat("x", maxLen)},

		// With prefix
		{"tagname:", "http://example.com", "tagname:http://example.com"},
		{"key:", "value,with,comma", "key:value_with_comma"},
		{"env:", "production", "env:production"},
	}

	for _, c := range cases {
		// Start with prefix data in buffer
		buf := []byte(c.prefix)
		originalLen := len(buf)

		// Append sanitized tag value
		buf = appendSanitizedTagValue(buf, c.in)
		got := string(buf)

		if got != c.want {
			t.Fatalf("prefix=%q in=%q want=%q (len %v) got=%q (len %v)", c.prefix, c.in, c.want, len(c.want), got, len(got))
		}

		// Verify prefix is preserved
		if len(c.prefix) > 0 && !strings.HasPrefix(got, c.prefix) {
			t.Errorf("prefix %q not preserved in result %q", c.prefix, got)
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
		buf = append(buf, "tag:"...)
		buf = appendSanitizedTagValue(buf, "http://café.com/path")
		expected := "tag:http://cafe.com/path"
		if string(buf) != expected {
			t.Errorf("buffer reuse failed: want %q, got %q", expected, string(buf))
		}
	})
}

// BenchmarkAppendSanitizedMetricName measures performance of metric name sanitization
// across different input types to ensure the implementation is efficient.
func BenchmarkAppendSanitizedMetricName(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{"simple ASCII", "simple.metric.name"},
		{"with dashes and underscores", "my_app-server.request_count"},
		{"Latin-1 accents", "café.résumé.münchen"},
		{"mixed accents", "naïve.señor.Zürich"},
		{"emoji in middle", "performance🚀metrics🎯data"},
		{"Chinese characters", "测试.metrics.数据"},
		{"Cyrillic text", "тест.данные.метрики"},
		{"mixed scripts", "app.测试.тест.café"},
		{"invalid UTF-8", "test\xC3\xA9\x80\xE2\x82invalid"},
		{"over maxLen", strings.Repeat("x", 300)},
		{"special chars", "foo!!!@@@###bar|||:::"},
		{"mostly trim chars", "...___---...___---"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			buf := make([]byte, 0, 512)
			b.ResetTimer()
			for b.Loop() {
				buf = appendSanitizedMetricName(buf[:0], bm.input)
			}
		})
	}

	// Benchmark with prefix (common use case)
	b.Run("with prefix", func(b *testing.B) {
		buf := make([]byte, 0, 512)
		b.ResetTimer()
		for b.Loop() {
			buf = buf[:0]
			buf = append(buf, "myapp."...)
			buf = appendSanitizedMetricName(buf, "café.metrics")
		}
	})
}

// BenchmarkTrimComparison compares manual trimming vs bytes.TrimFunc.
func BenchmarkTrimComparison(b *testing.B) {
	testCases := []struct {
		name   string
		prefix string
		input  string
	}{
		{"no trim needed", "prefix.", "simple.metric"},
		{"trim leading", "prefix.", "...trimmed"},
		{"trim trailing", "prefix.", "trimmed..."},
		{"trim both", "prefix.", "...trimmed..."},
		{"trim all", "prefix.", "......"},
		{"mixed trim chars", "prefix.", "._-test._-"},
		{"typical metric", "myapp.", "http.server.duration"},
	}

	for _, tc := range testCases {
		// Benchmark current manual approach
		b.Run(tc.name+"/manual", func(b *testing.B) {
			buf := make([]byte, 0, 512)
			b.ResetTimer()
			for b.Loop() {
				buf = buf[:0]
				buf = append(buf, tc.prefix...)
				origLen := len(buf)

				// Copy input
				buf = append(buf, tc.input...)

				// Manual trim (current approach)
				start, end := origLen, len(buf)
				for start < end && isTrim(buf[start]) {
					start++
				}
				for end > start && isTrim(buf[end-1]) {
					end--
				}

				if start > origLen || end < len(buf) {
					copy(buf[origLen:], buf[start:end])
					buf = buf[:origLen+(end-start)]
				}
			}
		})

		// Benchmark bytes.Trim* approach
		b.Run(tc.name+"/bytesTrim", func(b *testing.B) {
			buf := make([]byte, 0, 512)
			cutset := "._-"
			b.ResetTimer()
			for b.Loop() {
				buf = buf[:0]
				buf = append(buf, tc.prefix...)
				origLen := len(buf)

				// Copy input
				buf = append(buf, tc.input...)

				// bytes.Trim approach (TrimLeft then TrimRight)
				trimmed := bytes.Trim(buf[origLen:], cutset)
				buf = append(buf[:origLen], trimmed...)
			}
		})
	}
}
