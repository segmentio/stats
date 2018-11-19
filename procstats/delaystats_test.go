package procstats_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/procstats"
	"github.com/segmentio/stats/statstest"
)

func TestProcMetrics(t *testing.T) {
	u, err := user.Current()
	if err != nil || u.Uid != "0" {
		t.Log("test needs to be run as root")
		t.Skip()
	}

	rand.Seed(time.Now().UnixNano())

	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	proc := procstats.NewDelayMetricsWith(e, os.Getpid())

	for i := 0; i != 10; i++ {
		tmpfile, err := ioutil.TempFile("", "delaystats_test")
		if err != nil {
			t.Fatal(err)
		}
		defer tmpfile.Close()
		defer os.Remove(tmpfile.Name())

		b := make([]byte, rand.Int31n(1000000))

		if _, err := rand.Read(b); err != nil {
			t.Fatal(err)
		}
		if _, err := tmpfile.Write(b); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Sync(); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		t.Logf("collect number %d", i)

		// Work around issues testing on Docker containers that don't use host networking
		defer func() {
			if r := recover(); r != nil {
				if r1, ok := r.(error); ok && strings.HasPrefix(r1.Error(), "Failed to communicate with taskstats Netlink family") {
					t.Skip()
					return
				}
				panic(r)
			}
		}()
		proc.Collect()

		if len(h.Measures()) == 0 {
			t.Error("no measures were reported by the stats collector")
		}

		for _, m := range h.Measures() {
			t.Log(m)
		}

		h.Clear()
	}
}
