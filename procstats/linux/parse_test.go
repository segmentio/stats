// +build linux

package linux

import (
	"reflect"
	"testing"
)

func TestForEachLine(t *testing.T) {
	tests := []struct {
		text  string
		lines []string
	}{
		{
			text:  "",
			lines: []string{},
		},
		{
			text:  "1\n2\n3\nHello \n World!",
			lines: []string{"1", "2", "3", "Hello", "World!"},
		},
	}

	for _, test := range tests {
		lines := []string{}
		forEachLine(test.text, func(line string) { lines = append(lines, line) })

		if !reflect.DeepEqual(lines, test.lines) {
			t.Error(lines)
		}
	}
}

func TestForEachColumn(t *testing.T) {
	tests := []struct {
		text    string
		columns []string
	}{
		{
			text:    "",
			columns: []string{},
		},
		{
			text:    "Hello World!  A  B  C",
			columns: []string{"Hello World!", "A", "B", "C"},
		},
		{
			text:    "Max cpu time              unlimited            unlimited            seconds",
			columns: []string{"Max cpu time", "unlimited", "unlimited", "seconds"},
		},
	}

	for _, test := range tests {
		columns := []string{}
		forEachColumn(test.text, func(column string) { columns = append(columns, column) })

		if !reflect.DeepEqual(columns, test.columns) {
			t.Error(columns)
		}
	}
}

func TestForEachProperty(t *testing.T) {
	type KV struct {
		K string
		V string
	}

	tests := []struct {
		text string
		kv   []KV
	}{
		{
			text: "",
			kv:   []KV{},
		},
		{
			text: "A: 1\nB: 2\nC: 3\nD",
			kv:   []KV{{"A", "1"}, {"B", "2"}, {"C", "3"}, {"", "D"}},
		},
	}

	for _, test := range tests {
		kv := []KV{}
		forEachProperty(test.text, func(k string, v string) { kv = append(kv, KV{k, v}) })

		if !reflect.DeepEqual(kv, test.kv) {
			t.Error(kv)
		}
	}
}
