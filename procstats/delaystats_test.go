package procstats_test

import (
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ncw/directio"
	"github.com/segmentio/stats"
	"github.com/segmentio/stats/procstats"
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

	proc := procstats.NewDelayMetricsWith(e, pid)

	for i := 0; i != 10; i++ {
		tmpfile, err := ioutil.TempFile("", "delaystats_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		b := make([]byte, rand.Int31n(1000000))
		if _, err := rand.Read(b); err != nil {
			t.Fatal(err)
		}
		if _, err := tmpfile.Write(b); err != nil {
			t.Fatal(err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatal(err)
		}

		in, err := directio.OpenFile(tmpfile.Name(), os.O_RDONLY, 0666)
		block := directio.AlignedBlock(directio.BlockSize)
		if _, err := io.ReadFull(in, block); err != nil {
			t.Fatal(err)
		}

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
