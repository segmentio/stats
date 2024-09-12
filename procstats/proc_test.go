package procstats

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/segmentio/stats/v5"
	"github.com/segmentio/stats/v5/statstest"
)

func TestProcMetrics(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		testProcMetrics(t, os.Getpid())
	})
	t.Run("child", func(t *testing.T) {
		cmd := exec.Command("yes")
		cmd.Stdin = os.Stdin
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(200 * time.Millisecond)
		testProcMetrics(t, cmd.Process.Pid)
		cmd.Process.Signal(os.Interrupt)
		waitErr := cmd.Wait()
		//revive:disable-next-line
		if exitErr, ok := waitErr.(*exec.ExitError); ok && exitErr.Error() == "signal: interrupt" {
			// This is expected from stopping the process
		} else {
			t.Fatal(waitErr)
		}
	})
}

func testProcMetrics(t *testing.T, pid int) {
	t.Helper()
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	proc := NewProcMetricsWith(e, pid)

	// for darwin - catch the "can't collect child metrics" error before
	// starting the test
	_, err := CollectProcInfo(proc.pid)
	var o *OSUnsupportedError
	if errors.As(err, &o) {
		t.Skipf("can't run test because current OS is unsupported: %v", runtime.GOOS)
	}

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
