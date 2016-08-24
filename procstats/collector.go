package procstats

import "time"

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

func Start(collector Collector) func() {
	return StartWith(Config{
		Collector: collector,
	})
}

func StartWith(config Config) func() {
	config = setConfigDefaults(config)

	stop := make(chan struct{})
	join := make(chan struct{})

	go func() {
		defer close(join)

		ticker := time.NewTicker(config.CollectInterval)

		for {
			select {
			case <-ticker.C:
				config.Collector.Collect()
			case <-stop:
				return
			}
		}
	}()

	return func() {
		close(stop)
		<-join
	}
}

func setConfigDefaults(config Config) Config {
	if config.CollectInterval == 0 {
		config.CollectInterval = 5 * time.Second
	}

	if config.Collector == nil {
		config.Collector = MultiCollector()
	}

	return config
}
