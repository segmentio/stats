package stats

import (
	"fmt"
	"reflect"
	"testing"
)

func TestConcatTags(t *testing.T) {
	tests := []struct {
		t1 Tags
		t2 Tags
		t3 Tags
	}{
		{
			t1: nil,
			t2: nil,
			t3: nil,
		},
		{
			t1: Tags{{"hello", "world"}},
			t2: nil,
			t3: Tags{{"hello", "world"}},
		},
		{
			t1: nil,
			t2: Tags{{"hello", "world"}},
			t3: Tags{{"hello", "world"}},
		},
		{
			t1: Tags{{"hello", "world"}},
			t2: Tags{{"hello", "world"}},
			t3: Tags{{"hello", "world"}, {"hello", "world"}},
		},
		{
			t1: Tags{{"answer", "42"}},
			t2: Tags{{"hello", "world"}},
			t3: Tags{{"answer", "42"}, {"hello", "world"}},
		},
	}

	for _, test := range tests {
		if tags := concatTags(test.t1, test.t2); !reflect.DeepEqual(tags, test.t3) {
			t.Errorf("concatenating tags produced an invalid result: %#v != %#v", test.t3, tags)
		}
	}
}

func TestTags(t *testing.T) {
	tests := []struct {
		value interface{}
		tags  Tags
	}{
		{
			value: nil,
			tags:  nil,
		},
		{
			value: Tags{{"hello", "world"}},
			tags:  Tags{{"hello", "world"}},
		},
		{
			value: []Tag{{"hello", "world"}},
			tags:  Tags{{"hello", "world"}},
		},
		{
			value: map[string]string{"answer": "42", "hello": "world"},
			tags:  Tags{{"answer", "42"}, {"hello", "world"}},
		},
		{
			value: map[string]interface{}{"answer": 42, "hello": "world"},
			tags:  Tags{{"answer", "42"}, {"hello", "world"}},
		},
		{
			value: map[string]int{"answer": 42},
			tags:  Tags{{"answer", "42"}},
		},
	}

	for _, test := range tests {
		if tags := NewTags(test.value); !reflect.DeepEqual(tags, test.tags) {
			t.Errorf("%#v: invalid tags: %#v != %#v", test.value, test.tags, tags)
		}
	}
}

func TestTagsPanic(t *testing.T) {
	defer func() { recover() }()

	NewTags("")
	t.Error("the expected panic wasn't raised")
}

func TestTagsString(t *testing.T) {
	tests := []struct {
		tags Tags
		json string
	}{
		{
			tags: nil,
			json: `{}`,
		},
		{
			tags: Tags{},
			json: `{}`,
		},
		{
			tags: Tags{{"hello", "world"}},
			json: `{"hello":"world"}`,
		},
		{
			tags: Tags{{"answer", "42"}, {"hello", "world"}},
			json: `{"answer":"42","hello":"world"}`,
		},
	}

	for _, test := range tests {
		if s := test.tags.String(); s != test.json {
			t.Errorf("%#v: invalid json: %s != %s", test.tags, test.json, s)
		}
	}
}

func TestTagsFormat(t *testing.T) {
	tests := []struct {
		tags   Tags
		format string
	}{
		{
			tags:   nil,
			format: "",
		},
		{
			tags:   Tags{},
			format: "",
		},
		{
			tags:   Tags{{"hello", "world"}},
			format: "hello=world",
		},
		{
			tags:   Tags{{"answer", "42"}, {"hello", "world"}},
			format: "answer=42 hello=world",
		},
	}

	for _, test := range tests {
		if s := fmt.Sprint(test.tags); s != test.format {
			t.Errorf("%#v: invalid format: %s != %s", test.tags, test.format, s)
		}
	}
}
