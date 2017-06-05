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
		{mtype: histogram, name: "C", value: 0.1},
		{mtype: gauge, name: "B", value: 1, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: counter, name: "A", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: histogram, name: "C", value: 0.1},
		{mtype: gauge, name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: histogram, name: "C", value: 0.5},
		{mtype: histogram, name: "C", value: 10},
	}

	store := metricStore{}

	for _, m := range input {
		store.update(m, []float64{0.25, 0.5, 0.75, 1.0})
	}

	metrics := store.collect(nil)
	sort.Sort(byNameAndLabels(metrics))

	expects := []metric{
		{mtype: counter, name: "A", value: 3, labels: labels{}},
		{mtype: counter, name: "A", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: gauge, name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: histogram, name: "C_bucket", value: 2, labels: labels{{"le", "0.25"}}},
		{mtype: histogram, name: "C_bucket", value: 1, labels: labels{{"le", "0.5"}}},
		{mtype: histogram, name: "C_bucket", value: 0, labels: labels{{"le", "0.75"}}},
		{mtype: histogram, name: "C_bucket", value: 0, labels: labels{{"le", "1"}}},
		{mtype: histogram, name: "C_count", value: 4, labels: labels{}},
		{mtype: histogram, name: "C_sum", value: 10.7, labels: labels{}},
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad metrics:")
		t.Logf("expected: %v", expects)
		t.Logf("found:    %v", metrics)
	}
}
