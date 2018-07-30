package procstats

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestProcMetrics(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		testProcMetrics(t, os.Getpid())
	})
	t.Run("child", func(t *testing.T) {
		cmd := exec.Command("yes")
		cmd.Stdin = os.Stdin
		cmd.Stdout = ioutil.Discard
		cmd.Stderr = ioutil.Discard

		cmd.Start()
		time.Sleep(200 * time.Millisecond)
		testProcMetrics(t, cmd.Process.Pid)
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
	})
}

func testProcMetrics(t *testing.T, pid int) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	proc := NewProcMetricsWith(e, pid)

	for i := 0; i != 10; i++ {
		t.Logf("collect number %d", i)
		proc.Collect()

		if len(h.Measures()) == 0 {
			t.Error("no measures were reported by the stats collector")
		}

		for _, m := range h.Measures() {
			t.Log(m)
		}

		h.Clear()
		time.Sleep(10 * time.Millisecond)
	}
}
