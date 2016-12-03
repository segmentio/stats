package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	engine := NewEngine(EngineConfig{
		Prefix: "test",
		Tags:   []Tag{{"hello", "world"}},
	})
	defer engine.Close()

	now := time.Now()

	a := MakeCounter(engine, "A")
	b := MakeGauge(engine, "B")
	c := MakeCounter(engine, "C", Tag{"context", "test"})
	d := MakeTimer(engine, "D").StartAt(now)

	a.Incr()
	b.Set(2)
	c.Add(3)

	d.StampAt("lap", now.Add(1*time.Second))
	d.StampAt("lap", now.Add(2*time.Second))
	d.StopAt(now.Add(3 * time.Second))

	// Give a bit of time for the engine to update its state.
	time.Sleep(10 * time.Millisecond)

	metrics, version := engine.State(0)
	sortMetrics(metrics)

	if version != 6 {
		t.Error("bad engine version:", version)
	}

	expects := []Metric{
		Metric{
			Type:  CounterType,
			Key:   "A?",
			Name:  "A",
			Tags:  nil,
			Value: 1,
			Namespace: Namespace{
				Name: "test",
				Tags: []Tag{{"hello", "world"}},
			},
		},
		Metric{
			Type:  GaugeType,
			Key:   "B?",
			Name:  "B",
			Tags:  nil,
			Value: 2,
			Namespace: Namespace{
				Name: "test",
				Tags: []Tag{{"hello", "world"}},
			},
		},
		Metric{
			Type:  CounterType,
			Key:   "C?context=test",
			Name:  "C",
			Tags:  []Tag{{"context", "test"}},
			Value: 3,
			Namespace: Namespace{
				Name: "test",
				Tags: []Tag{{"hello", "world"}},
			},
		},
		Metric{
			Type:  HistogramType,
			Key:   "D?&stamp=lap",
			Name:  "D",
			Tags:  []Tag{{"stamp", "lap"}},
			Value: 1,
			Namespace: Namespace{
				Name: "test",
				Tags: []Tag{{"hello", "world"}},
			},
		},
		Metric{
			Type:  HistogramType,
			Key:   "D?&stamp=lap",
			Name:  "D",
			Tags:  []Tag{{"stamp", "lap"}},
			Value: 1,
			Namespace: Namespace{
				Name: "test",
				Tags: []Tag{{"hello", "world"}},
			},
		},
		Metric{
			Type:  HistogramType,
			Key:   "D?&stamp=total",
			Name:  "D",
			Tags:  []Tag{{"stamp", "total"}},
			Value: 1,
			Namespace: Namespace{
				Name: "test",
				Tags: []Tag{{"hello", "world"}},
			},
		},
	}

	for i := range metrics {
		metrics[i].Time = time.Time{} // reset because we can't predict that value
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad engine state:")

		for i := range metrics {
			m := metrics[i]
			e := expects[i]

			if !reflect.DeepEqual(m, e) {
				t.Logf("unexpected metric at index %d:\n<<< %#v\n>>> %#v", i, m, e)
			}
		}
	}
}
