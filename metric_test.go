package stats

import (
	"reflect"
	"testing"
)

func TestMakeOpts(t *testing.T) {
	tests := []struct {
		name string
		help string
		tags Tags
	}{
		{
			name: "",
			help: "",
			tags: nil,
		},
		{
			name: "hello",
			help: "world",
			tags: Tags{{"hello", "world"}},
		},
	}

	for _, test := range tests {
		opts := MakeOpts(test.name, test.help, test.tags...)

		if opts.Name != test.name {
			t.Errorf("invalid opts name: %#v != %#v", test.name, opts.Name)
		}

		if opts.Help != test.help {
			t.Errorf("invalid opts help: %#v != %#v", test.help, opts.Help)
		}

		if !reflect.DeepEqual(opts.Tags, test.tags) {
			t.Errorf("invalid opts tags: %#v != %#v", test.tags, opts.Tags)
		}
	}
}
