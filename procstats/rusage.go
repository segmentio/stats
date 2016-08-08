// +build !windows

package procstats

import (
	"syscall"
	"time"

	"github.com/segmentio/stats"
)

type RusageStats struct {
	Utime    stats.Counter
	Stime    stats.Counter
	Maxrss   stats.Gauge
	Ixrss    stats.Gauge
	Idrss    stats.Gauge
	Isrss    stats.Gauge
	Minflt   stats.Counter
	Majflt   stats.Counter
	Nswap    stats.Counter
	Inblock  stats.Counter
	Oublock  stats.Counter
	Msgsnd   stats.Counter
	Msgrcv   stats.Counter
	Nsignals stats.Counter
	Nvcsw    stats.Counter
	Nivcsw   stats.Counter

	lastUtime    time.Duration
	lastStime    time.Duration
	lastMinflt   int64
	lastMajflt   int64
	lastNswap    int64
	lastInblock  int64
	lastOublock  int64
	lastMsgsnd   int64
	lastMsgrcv   int64
	lastNsignals int64
	lastNvcsw    int64
	lastNivcsw   int64
}

func NewRusageStats(client stats.Client) *RusageStats {
	return &RusageStats{
		Utime:    client.Counter("rusage.utime.duration"),
		Stime:    client.Counter("rusage.stime.duration"),
		Maxrss:   client.Gauge("rusage.maxrss.bytes"),
		Ixrss:    client.Gauge("rusage.ixrss.bytes"),
		Idrss:    client.Gauge("rusage.idrss.bytes"),
		Isrss:    client.Gauge("rusage.isrss.bytes"),
		Minflt:   client.Counter("rusage.minftl.count"),
		Majflt:   client.Counter("rusage.majflt.count"),
		Nswap:    client.Counter("rusage.nswap.count"),
		Inblock:  client.Counter("rusage.inblock.count"),
		Oublock:  client.Counter("rusage.oublock.count"),
		Msgsnd:   client.Counter("rusage.msgsnd.count"),
		Msgrcv:   client.Counter("rusage.msgrcv.count"),
		Nsignals: client.Counter("rusage.nsignals.count"),
		Nvcsw:    client.Counter("rusage.nvcsw.count"),
		Nivcsw:   client.Counter("rusage.nivcsw.count"),
	}
}

func (ru *RusageStats) Collect() {
	rusage := &syscall.Rusage{}
	syscall.Getrusage(syscall.RUSAGE_SELF, rusage)

	utime := timevalToDuration(rusage.Utime)
	stime := timevalToDuration(rusage.Stime)

	ru.Utime.Add(float64(utime - ru.lastUtime))
	ru.Stime.Add(float64(stime - ru.lastStime))
	ru.Maxrss.Set(float64(rusage.Maxrss))
	ru.Ixrss.Set(float64(rusage.Ixrss))
	ru.Idrss.Set(float64(rusage.Idrss))
	ru.Isrss.Set(float64(rusage.Isrss))
	ru.Minflt.Add(float64(rusage.Minflt - ru.lastMinflt))
	ru.Majflt.Add(float64(rusage.Majflt - ru.lastMajflt))
	ru.Nswap.Add(float64(rusage.Nswap - ru.lastNswap))
	ru.Inblock.Add(float64(rusage.Inblock - ru.lastInblock))
	ru.Oublock.Add(float64(rusage.Oublock - ru.lastOublock))
	ru.Msgsnd.Add(float64(rusage.Msgsnd - ru.lastMsgsnd))
	ru.Msgrcv.Add(float64(rusage.Msgrcv - ru.lastMsgrcv))
	ru.Nsignals.Add(float64(rusage.Nsignals - ru.lastNsignals))
	ru.Nvcsw.Add(float64(rusage.Nvcsw - ru.lastNvcsw))
	ru.Nivcsw.Add(float64(rusage.Nivcsw - ru.lastNivcsw))

	ru.lastUtime = utime
	ru.lastStime = stime
	ru.lastMinflt = rusage.Minflt
	ru.lastMajflt = rusage.Majflt
	ru.lastNswap = rusage.Nswap
	ru.lastInblock = rusage.Inblock
	ru.lastOublock = rusage.Oublock
	ru.lastMsgsnd = rusage.Msgsnd
	ru.lastMsgrcv = rusage.Msgrcv
	ru.lastNsignals = rusage.Nsignals
	ru.lastNvcsw = rusage.Nvcsw
	ru.lastNivcsw = rusage.Nivcsw
}

func timevalToDuration(t syscall.Timeval) time.Duration {
	return (time.Duration(t.Sec) * time.Second) + (time.Duration(t.Usec) * time.Microsecond)
}
