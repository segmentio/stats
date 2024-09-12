package stdoutstats

import (
	"bytes"
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
	want := "stdoutstats.test.compression_ratio:0.3|d|#algorithm:jwt256,file_size_bucket:bucket_name\n"
	if !strings.HasSuffix(bufstr, want) {
		t.Errorf("stdoutstats: got %v want %v", bufstr, want)
	}
}
