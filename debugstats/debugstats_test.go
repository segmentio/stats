package debugstats

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/segmentio/stats/v5"
)

func TestStdout(t *testing.T) {
	var buf bytes.Buffer
	s := &Client{Dst: &buf}
	stats.Register(s)
	stats.Set("blah", 7)
	stats.Observe("compression_ratio", 0.3, stats.T("file_size_bucket", "bucket_name"), stats.T("algorithm", "jwt256"))
	bufstr := buf.String()
	want := "debugstats.test.compression_ratio:0.3|d|#algorithm:jwt256,file_size_bucket:bucket_name\n"
	if !strings.HasSuffix(bufstr, want) {
		t.Errorf("debugstats: got %v want %v", bufstr, want)
	}
}

func TestStdoutGrepMatch(t *testing.T) {
	var buf bytes.Buffer
	s := &Client{
		Dst:  &buf,
		Grep: regexp.MustCompile(`compression_ratio`),
	}
	eng := stats.NewEngine("prefix", s)

	// Send measures that match and don't match the Grep pattern
	eng.Set("compression_ratio", 0.3)
	eng.Set("other_metric", 42)
	eng.Flush()

	bufstr := buf.String()

	// Check that only the matching measure is output
	if !strings.Contains(bufstr, "compression_ratio:0.3") {
		t.Errorf("debugstats: expected output to contain 'compression_ratio:0.3', but it did not. Output: %s", bufstr)
	}

	if strings.Contains(bufstr, "other_metric") {
		t.Errorf("debugstats: expected output not to contain 'other_metric', but it did. Output: %s", bufstr)
	}
}
