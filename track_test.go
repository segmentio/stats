package stats

import (
	"errors"
	"reflect"
	"testing"
)

func TestHandlerFunc(t *testing.T) {
	var m0 = NewMetric("test")
	var v0 = Count(1)

	var m1 Metric
	var v1 Value

	x := HandlerFunc(func(m Metric, v Value) error {
		m1 = m
		v1 = v
		return nil
	})

	if err := x.Handle(m0, v0); err != nil {
		t.Errorf("stats handler returned an error: %s", err)
	}

	if !reflect.DeepEqual(m0, m1) {
		t.Errorf("invalid metric seen by stats handler: %#v != %#v", m0, m1)
	}

	if !reflect.DeepEqual(v0, v1) {
		t.Errorf("invalid value seen by stats handler: %#v != %#v", v0, v1)
	}
}

func TestTrackerFunc(t *testing.T) {
	var m0 = NewMetric("test")
	var v0 = Count(1)

	var m1 Metric
	var v1 Value

	x := TrackerFunc(func(m Metric, v Value) {
		m1 = m
		v1 = v
	})

	x.Track(m0, v0)

	if !reflect.DeepEqual(m0, m1) {
		t.Errorf("invalid metric seen by stats tracker: %#v != %#v", m0, m1)
	}

	if !reflect.DeepEqual(v0, v1) {
		t.Errorf("invalid value seen by stats tracker: %#v != %#v", v0, v1)
	}
}

func TestTracker(t *testing.T) {
	var m0 = NewMetric("test")
	var v0 = Count(1)

	var m1 Metric
	var v1 Value

	x := NewTracker("test", HandlerFunc(func(m Metric, v Value) error {
		m1 = m
		v1 = v
		return nil
	}), func(m Metric, v Value, e error) {
		t.Errorf("%#v:%#v: %s", m, v, e)
	})

	x.Track(m0, v0)

	if !reflect.DeepEqual(m0, m1) {
		t.Errorf("invalid metric seen by stats tracker: %#v != %#v", m0, m1)
	}

	if !reflect.DeepEqual(v0, v1) {
		t.Errorf("invalid value seen by stats tracker: %#v != %#v", v0, v1)
	}
}

func TestTrackerError(t *testing.T) {
	var m0 = NewMetric("test")
	var v0 = Count(1)
	var e0 = errors.New("Bad")
	var e1 error

	x := NewTracker("test", HandlerFunc(func(m Metric, v Value) error {
		return e0
	}), func(m Metric, v Value, e error) {
		e1 = e
	})

	x.Track(m0, v0)

	if !reflect.DeepEqual(e0, e1) {
		t.Errorf("invalid error seen by stats tracker: %#v != %#v", e0, e1)
	}
}

func TestTrackerPanic(t *testing.T) {
	defer func() { recover() }()

	NewTracker("test", HandlerFunc(func(m Metric, v Value) error { return nil }), nil)
	t.Error("the expected panic wasn't raised")
}

func TestDispatcher(t *testing.T) {
	const N = 3

	var m0 = NewMetric("test")
	var v0 = Count(1)

	var ms [N]Metric
	var vs [N]Value
	var ts [N]Tracker

	for i := 0; i != N; i++ {
		offset := i
		ts[i] = TrackerFunc(func(m Metric, v Value) {
			ms[offset] = m
			vs[offset] = v
		})
	}

	x := NewDispatcher(ts[:]...)
	x.Track(m0, v0)

	for i := 0; i != N; i++ {
		if !reflect.DeepEqual(m0, ms[i]) {
			t.Errorf("invalid metric seen by stats tracker: %#v != %#v", m0, ms[i])
		}

		if !reflect.DeepEqual(v0, vs[i]) {
			t.Errorf("invalid value seen by stats tracker: %#v != %#v", v0, vs[i])
		}
	}
}
