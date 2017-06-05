package prometheus

import (
	"reflect"
	"sort"
	"testing"
)

func TestMetricStore(t *testing.T) {
	input := []metric{
		{mtype: counter, name: "A", value: 1},
		{mtype: counter, name: "A", value: 2},
		{mtype: gauge, name: "B", value: 1, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: counter, name: "A", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: gauge, name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
	}

	store := metricStore{}

	for _, m := range input {
		store.update(m)
	}

	metrics := store.collect(nil)
	sort.Slice(metrics, func(i int, j int) bool {
		m1 := &metrics[i]
		m2 := &metrics[j]
		return m1.name < m2.name || (m1.name == m2.name && m1.labels.less(m2.labels))
	})

	expects := []metric{
		{mtype: counter, name: "A", value: 3, labels: labels{}},
		{mtype: counter, name: "A", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: gauge, name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad metrics:")
		t.Logf("expected: %v", expects)
		t.Logf("found:    %v", metrics)
	}
}
