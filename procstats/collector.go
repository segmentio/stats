package procstats

import (
	"io"
	"runtime"
	"sync"
	"time"
)

type Collector interface {
	Collect()
}

type CollectorFunc func()

func (f CollectorFunc) Collect() { f() }

type Config struct {
	Collector       Collector
	CollectInterval time.Duration
}

func MultiCollector(collectors ...Collector) Collector {
	return CollectorFunc(func() {
		for _, c := range collectors {
			c.Collect()
		}
	})
}

func StartCollector(collector Collector) io.Closer {
	return StartCollectorWith(Config{Collector: collector})
}

func StartCollectorWith(config Config) io.Closer {
	config = setConfigDefaults(config)

	stop := make(chan struct{})
	join := make(chan struct{})

	go func() {
		// Locks the OS thread, stats collection heavily relies on blocking
		// syscalls, letting other goroutines execute on the same thread
		// increases the chance for the Go runtime to detected that the thread
		// is blocked and schedule a new one.
		runtime.LockOSThread()

		defer runtime.UnlockOSThread()
		defer close(join)

		ticker := time.NewTicker(config.CollectInterval)
		config.Collector.Collect()
		for {
			select {
			case <-ticker.C:
				config.Collector.Collect()
			case <-stop:
				return
			}
		}
	}()

	return &closer{stop: stop, join: join}
}

func setConfigDefaults(config Config) Config {
	if config.CollectInterval == 0 {
		config.CollectInterval = 15 * time.Second
	}

	if config.Collector == nil {
		config.Collector = MultiCollector()
	}

	return config
}

type closer struct {
	once sync.Once
	stop chan<- struct{}
	join <-chan struct{}
}

func (c *closer) Close() error {
	c.once.Do(c.close)
	return nil
}

func (c *closer) close() {
	close(c.stop)
	<-c.join
}
