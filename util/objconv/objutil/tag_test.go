package objutil

import "testing"

func TestParseTag(t *testing.T) {
	tests := []struct {
		tag string
		res Tag
	}{
		{
			tag: "",
			res: Tag{},
		},
		{
			tag: "hello",
			res: Tag{Name: "hello"},
		},
		{
			tag: ",omitempty",
			res: Tag{Omitempty: true},
		},
		{
			tag: "-",
			res: Tag{Name: "-"},
		},
		{
			tag: "hello,omitempty",
			res: Tag{Name: "hello", Omitempty: true},
		},
		{
			tag: "-,omitempty",
			res: Tag{Name: "-", Omitempty: true},
		},
		{
			tag: "hello,omitzero",
			res: Tag{Name: "hello", Omitzero: true},
		},
		{
			tag: "-,omitempty,omitzero",
			res: Tag{Name: "-", Omitempty: true, Omitzero: true},
		},
	}

	for _, test := range tests {
		t.Run(test.tag, func(t *testing.T) {
			if res := ParseTag(test.tag); res != test.res {
				t.Errorf("%s: %#v != %#v", test.tag, test.res, res)
			}
		})
	}
}

func TestParseTagJSON(t *testing.T) {
	tests := []struct {
		tag string
		res Tag
	}{
		{
			tag: "",
			res: Tag{},
		},
		{
			tag: "hello",
			res: Tag{Name: "hello"},
		},
		{
			tag: ",omitempty",
			res: Tag{Omitempty: true},
		},
		{
			tag: "-",
			res: Tag{Name: "-"},
		},
		{
			tag: "hello,omitempty",
			res: Tag{Name: "hello", Omitempty: true},
		},
		{
			tag: "-,omitempty",
			res: Tag{Name: "-", Omitempty: true},
		},
	}

	for _, test := range tests {
		t.Run(test.tag, func(t *testing.T) {
			if res := ParseTag(test.tag); res != test.res {
				t.Errorf("%s: %#v != %#v", test.tag, test.res, res)
			}
		})
	}
}
