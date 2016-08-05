package prometheus

import (
	"reflect"
	"testing"

	"github.com/segmentio/stats"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "",
			out: ``,
		},
		{
			in:  "hello",
			out: `hello`,
		},
		{
			in:  "hello1",
			out: `hello1`,
		},
		{
			in:  "hello-",
			out: `hello_`,
		},
	}

	for _, test := range tests {
		if s := sanitize(test.in); s != test.out {
			t.Errorf("escape(%#v) => %#v != %#v", test.in, test.out, s)
		}
	}
}

func TestEscape(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "",
			out: ``,
		},
		{
			in:  "hello",
			out: `hello`,
		},
		{
			in:  "hello\n",
			out: `hello\n`,
		},
		{
			in:  "hello\"",
			out: `hello\"`,
		},
		{
			in:  "hello\\",
			out: `hello\\`,
		},
	}

	for _, test := range tests {
		if s := escape(test.in); s != test.out {
			t.Errorf("escape(%s) => %s != %s", test.in, test.out, s)
		}
	}
}

func TestLabelString(t *testing.T) {
	tests := []struct {
		label  Label
		string string
	}{
		{
			label:  Label{"hello", "world"},
			string: `hello="world"`,
		},
		{
			label:  Label{"hello-", "world"},
			string: `hello_="world"`,
		},
		{
			label:  Label{"hello", "\n"},
			string: `hello="\n"`,
		},
	}

	for _, test := range tests {
		if s := test.label.String(); s != test.string {
			t.Errorf("%#v => %#v != %#v", test.label, test.string, s)
		}
	}
}

func TestLabelsString(t *testing.T) {
	tests := []struct {
		labels Labels
		string string
	}{
		{
			labels: nil,
			string: ``,
		},
		{
			labels: Labels{},
			string: ``,
		},
		{
			labels: Labels{{"hello", "world"}},
			string: `{hello="world"}`,
		},
		{
			labels: Labels{{"hello-", "world"}},
			string: `{hello_="world"}`,
		},
		{
			labels: Labels{{"hello", "\n"}},
			string: `{hello="\n"}`,
		},
		{
			labels: Labels{{"hello", "world"}, {"answer", "42"}},
			string: `{hello="world", answer="42"}`,
		},
	}

	for _, test := range tests {
		if s := test.labels.String(); s != test.string {
			t.Errorf("%#v => %#v != %#v", test.labels, test.string, s)
		}
	}
}

func TestMakeLabels(t *testing.T) {
	tests := []struct {
		tags   stats.Tags
		labels Labels
	}{
		{
			tags:   nil,
			labels: nil,
		},
		{
			tags:   stats.Tags{},
			labels: Labels{},
		},
		{
			tags:   stats.Tags{{"hello", "world"}},
			labels: Labels{{"hello", "world"}},
		},
		{
			tags:   stats.Tags{{"hello", "world"}, {"answer", "42"}},
			labels: Labels{{"hello", "world"}, {"answer", "42"}},
		},
	}

	for _, test := range tests {
		if labels := makeLabels(test.tags); !reflect.DeepEqual(labels, test.labels) {
			t.Errorf("%#v => %#v != %#v", test.tags, test.labels, labels)
		}
	}
}
