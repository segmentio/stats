package topk

import (
	"errors"
	"time"

	"github.com/segmentio/stats/v4"
)

// todo: add more configuration
type topkConfig struct {
	engine        *stats.Engine
	k             uint
	epsilon       float64
	delta         float64
	flushInterval time.Duration
}

func newTopkConfig() topkConfig {
	return topkConfig{
		engine:        stats.DefaultEngine,
		k:             5,
		epsilon:       0.001,
		delta:         0.99,
		flushInterval: 10 * time.Second,
	}
}

type Opt func(config *topkConfig) error

func WithEngine(engine *stats.Engine) Opt {
	return func(config *topkConfig) error {
		if engine == nil {
			return errors.New("engine is nil")
		}
		config.engine = engine
		return nil
	}
}

func WithK(k uint) Opt {
	return func(config *topkConfig) error {
		if k <= 0 {
			return errors.New("k must be positive")
		}
		config.k = k
		return nil
	}
}

func WithEpsilon(epsilon float64) Opt {
	return func(config *topkConfig) error {
		if epsilon <= 0 {
			return errors.New("epsilon must be positive")
		}
		config.epsilon = epsilon
		return nil
	}
}

func WithDelta(delta float64) Opt {
	return func(config *topkConfig) error {
		if delta <= 0 {
			return errors.New("delta must be positive")
		}
		config.delta = delta
		return nil
	}
}

func WithFlushInterval(interval time.Duration) Opt {
	return func(config *topkConfig) error {
		if interval <= 0 {
			return errors.New("flush interval must be positive")
		}
		config.flushInterval = interval
		return nil
	}
}
